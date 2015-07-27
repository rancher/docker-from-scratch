package main

import (
	"os"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	dockerlaunch "github.com/rancher/docker-from-scratch"
)

func getValue(index int, args []string) string {
	val := args[index]
	parts := strings.SplitN(val, "=", 2)
	if len(parts) == 1 {
		if len(args) > index+1 {
			return args[index+1]
		} else {
			return ""
		}
	} else {
		return parts[2]
	}
}

func main() {
	if os.Getenv("DOCKER_LAUNCH_DEBUG") == "true" {
		log.SetLevel(log.DebugLevel)
	}

	if len(os.Args) < 2 {
		log.Fatalf("Usage Example: %s /usr/bin/docker -d -D", os.Args[0])
	}

	args := []string{}
	if len(os.Args) > 1 {
		args = os.Args[2:]
	}

	var config dockerlaunch.Config

	for i, arg := range args {
		if strings.HasPrefix(arg, "--bip") {
			config.BridgeAddress = getValue(i, args)
		} else if strings.HasPrefix(arg, "-b") || strings.HasPrefix(arg, "--bridge") {
			config.BridgeName = getValue(i, args)
		} else if strings.HasPrefix(arg, "--mtu") {
			mtu, err := strconv.Atoi(getValue(i, args))
			if err != nil {
				config.BridgeMtu = mtu
			}
		}
	}

	if config.BridgeName != "" && config.BridgeAddress != "" {
		newArgs := []string{}
		skip := false
		for _, arg := range args {
			if skip {
				skip = false
				continue
			}

			if arg == "--bip" {
				skip = true
				continue
			} else if strings.HasPrefix(arg, "--bip=") {
				continue
			}

			newArgs = append(newArgs, arg)
		}

		args = newArgs
	}

	log.Debugf("Launch config %#v", config)

	err := dockerlaunch.LaunchDocker(&config, os.Args[1], args...)
	if err != nil {
		log.Fatal(err)
	}
}
