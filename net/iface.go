package net

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"

	"github.com/Scalingo/acadock-monitoring/config"
	"github.com/Scalingo/acadock-monitoring/docker"
)

func getContainerIface(id string) (string, error) {
	ifaceID, err := getContainerIfaceID(id)
	if err != nil {
		return "", errors.Wrapf(err, "fail to get container '%v' interface", id)
	}

	stdout := new(bytes.Buffer)
	cmd := exec.Command("ip", "link", "show")
	cmd.Stdout = stdout
	cmd.Stderr = stdout
	err = cmd.Start()
	if err != nil {
		return "", err
	}
	err = cmd.Wait()
	if err != nil {
		return "", fmt.Errorf("%v: %v", err, stdout.String())
	}

	for _, line := range strings.Split(stdout.String(), "\n") {
		if strings.HasPrefix(line, ifaceID) {
			return strings.TrimSpace(strings.Split(strings.Split(line, "@")[0], ":")[1]), nil
		}
	}

	return "", errors.Errorf("interface not found for '%v', %v", id, ifaceID)
}

func getContainerIfaceID(id string) (string, error) {
	pid, err := docker.Pid(id)
	if err != nil {
		return "", errors.Wrapf(err, "fail to get pid of container '%v'", id)
	}

	stdout := new(bytes.Buffer)
	cmd := exec.Command(os.Args[0], "-ns-iface-id", pid)
	cmd.Env = []string{"PROC_DIR=" + config.ENV["PROC_DIR"], "PATH=" + os.Getenv("PATH")}
	cmd.Stdout = stdout
	cmd.Stderr = stdout
	err = cmd.Start()
	if err != nil {
		return "", err
	}
	err = cmd.Wait()
	if err != nil {
		return "", errors.Wrapf(err, "'%v' failed with '%v'", cmd, stdout.String())
	}

	return stdout.String(), nil
}
