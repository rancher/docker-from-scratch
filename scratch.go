package dockerlaunch

import (
	"bufio"
	"io"
	"os"
	"path"
	"strings"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/docker-from-scratch/util"
)

const caCrtDir = "/etc/ssl/certs"
const caCrtName = "ca-certificates.crt"
const caCrtDefault = "/usr/etc/ssl/certs/ca-certificates.crt"

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

	for scanner.Scan() {
		cgroup := strings.SplitN(scanner.Text(), "\t", 2)[0]
		if cgroup == "" || cgroup[0] == '#' {
			continue
		}

		if err := mountCgroup(cgroup); err != nil {
			return err
		}
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	log.Debug("Done mouting cgroupfs")
	return nil
}

func mountCgroup(cgroup string) error {
	if err := createDirs("/sys/fs/cgroup/" + cgroup); err != nil {
		return err
	}

	if err := createMounts([][]string{{"none", "/sys/fs/cgroup/" + cgroup, "cgroup", cgroup}}...); err != nil {
		return err
	}

	return nil
}

func execDocker(docker string, args []string) error {
	log.Debugf("Launching Docker %s %s", docker, args)
	return syscall.Exec(docker, append([]string{docker}, args...), os.Environ())
}

func copyFile(src, folder, name string) error {
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

func createPasswd() error {
	if _, err := os.Stat("/etc/passwd"); err == nil {
		return nil
	}

	f, err := os.Create("/etc/passwd")
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.Write([]byte("root:x:0:0:root:/root:/bin/bash\n"))
	return err
}

func LaunchDocker(docker string, args ...string) error {
	os.Setenv("PATH", "/sbin:/usr/sbin:/usr/bin")

	if err := createPasswd(); err != nil {
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

	if err := copyFile(caCrtDefault, caCrtDir, caCrtName); err != nil {
		return err
	}

	return execDocker(docker, args)
}
