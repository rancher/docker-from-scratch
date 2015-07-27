package dockerlaunch

import (
	"bufio"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/libnetwork/resolvconf"
	"github.com/rancher/docker-from-scratch/util"
	"github.com/rancher/netconf"
)

const defaultPrefix = "/usr"

var (
	mounts [][]string = [][]string{
		{"devtmpfs", "/dev", "devtmpfs", ""},
		{"none", "/dev/pts", "devpts", ""},
		{"none", "/proc", "proc", ""},
		{"none", "/run", "tmpfs", ""},
		{"none", "/sys", "sysfs", ""},
		{"none", "/sys/fs/cgroup", "tmpfs", ""},
		{"none", "/var/run", "tmpfs", ""},
	}
)

type Config struct {
	DnsConfig     netconf.DnsConfig
	BridgeName    string
	BridgeAddress string
	BridgeMtu     int
}

func createMounts(mounts ...[]string) error {
	for _, mount := range mounts {
		log.Debugf("Mounting %s %s %s %s", mount[0], mount[1], mount[2], mount[3])
		err := util.Mount(mount[0], mount[1], mount[2], mount[3])
		if err != nil {
			return err
		}
	}

	return nil
}

func createDirs(dirs ...string) error {
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			log.Debugf("Creating %s", dir)
			err = os.MkdirAll(dir, 0755)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func mountCgroups() error {
	f, err := os.Open("/proc/cgroups")
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	hierarchies := make(map[string][]string)

	for scanner.Scan() {
		fields := strings.SplitN(scanner.Text(), "\t", 2)
		cgroup := fields[0]
		if cgroup == "" || cgroup[0] == '#' || len(fields) < 2 {
			continue
		}

		hierarchy := fields[1]
		hierarchies[hierarchy] = append(hierarchies[hierarchy], cgroup)
	}

	for _, hierarchy := range hierarchies {
		if err := mountCgroup(strings.Join(hierarchy, ",")); err != nil {
			return err
		}
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	log.Debug("Done mouting cgroupfs")
	return nil
}

func createSymlink(src, dest string) error {
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		log.Debugf("Symlinking %s => %s", src, dest)
		if err = os.Symlink(src, dest); err != nil {
			return err
		}
	}

	return nil
}

func mountCgroup(cgroup string) error {
	if err := createDirs("/sys/fs/cgroup/" + cgroup); err != nil {
		return err
	}

	if err := createMounts([][]string{{"none", "/sys/fs/cgroup/" + cgroup, "cgroup", cgroup}}...); err != nil {
		return err
	}

	parts := strings.Split(cgroup, ",")
	if len(parts) > 1 {
		for _, part := range parts {
			if err := createSymlink("/sys/fs/cgroup/"+cgroup, "/sys/fs/cgroup/"+part); err != nil {
				return err
			}
		}
	}

	return nil
}

func execDocker(docker string, args []string) error {
	log.Debugf("Launching Docker %s %s", docker, args)
	return syscall.Exec(docker, append([]string{docker}, args...), os.Environ())
}

func copyDefault(folder, name string) error {
	defaultFile := path.Join(defaultPrefix, folder, name)
	if err := copyFile(defaultFile, folder, name); err != nil {
		return err
	}

	return nil
}

func defaultFiles(files ...string) error {
	for _, file := range files {
		dir := path.Dir(file)
		name := path.Base(file)
		if err := copyDefault(dir, name); err != nil {
			return err
		}
	}

	return nil
}

func copyFile(src, folder, name string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}

	dst := path.Join(folder, name)
	if _, err := os.Stat(dst); err == nil {
		return nil
	}

	if err := createDirs(folder); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func tryCreateFile(name, content string) error {
	if _, err := os.Stat(name); err == nil {
		return nil
	}

	return ioutil.WriteFile(name, []byte(content), 0644)
}

func createPasswd() error {
	return tryCreateFile("/etc/passwd", "root:x:0:0:root:/root:/bin/sh\n")
}

func createGroup() error {
	return tryCreateFile("/etc/group", "root:x:0:\n")
}

func setupNetworking(config *Config) error {
	if len(config.DnsConfig.Nameservers) != 0 {
		if err := resolvconf.Build("/etc/resolv.conf", config.DnsConfig.Nameservers, config.DnsConfig.Search); err != nil {
			return err
		}
	}

	if config.BridgeName != "" {
		log.Debugf("Creating bridge %s (%s)", config.BridgeName, config.BridgeAddress)
		if err := netconf.ApplyNetworkConfigs(&netconf.NetworkConfig{
			Interfaces: map[string]netconf.InterfaceConfig{
				config.BridgeName: {
					Address: config.BridgeAddress,
					MTU:     config.BridgeMtu,
					Bridge:  true,
				},
			},
		}); err != nil {
			return err
		}
	}

	return nil
}

func LaunchDocker(config *Config, docker string, args ...string) error {
	os.Setenv("PATH", "/sbin:/usr/sbin:/usr/bin")

	if err := defaultFiles(
		"/etc/ssl/certs/ca-certificates.crt",
		"/etc/passwd",
		"/etc/group",
	); err != nil {
		return err
	}

	if err := createPasswd(); err != nil {
		return err
	}

	if err := createGroup(); err != nil {
		return err
	}

	if err := createDirs("/tmp", "/root/.ssh"); err != nil {
		return err
	}

	if err := createMounts(mounts...); err != nil {
		return err
	}

	if err := mountCgroups(); err != nil {
		return err
	}

	if err := setupNetworking(config); err != nil {
		return err
	}

	return execDocker(docker, args)
}
