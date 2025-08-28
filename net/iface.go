package net

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Scalingo/acadock-monitoring/cgroup"
	"github.com/Scalingo/acadock-monitoring/config"

	"github.com/Scalingo/go-utils/errors/v3"
)

func getContainerIface(ctx context.Context, id string) (string, error) {
	ifaceID, err := getContainerIfaceID(ctx, id)
	if err != nil {
		return "", errors.Wrapf(ctx, err, "get container '%v' interface", id)
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

	return "", errors.Errorf(ctx, "interface not found for '%v', %v", id, ifaceID)
}

func getContainerIfaceID(ctx context.Context, id string) (string, error) {
	manager, err := cgroup.NewManager(ctx, id)
	if err != nil {
		return "", errors.Wrapf(ctx, err, "get cgroup manager for container '%v'", id)
	}
	pids, err := manager.Pids(ctx)
	if err != nil {
		return "", errors.Wrapf(ctx, err, "get pid of container '%v'", id)
	}
	if len(pids) == 0 {
		return "", errors.Errorf(ctx, "no pid found for container '%v'", id)
	}
	pid := pids[0]

	stdout := new(bytes.Buffer)
	cmd := exec.Command(os.Args[0], "-ns-iface-id", fmt.Sprintf("%d", pid))
	cmd.Env = []string{"PROC_DIR=" + config.ENV["PROC_DIR"], "PATH=" + os.Getenv("PATH")}
	cmd.Stdout = stdout
	cmd.Stderr = stdout
	err = cmd.Start()
	if err != nil {
		return "", err
	}
	err = cmd.Wait()
	if err != nil {
		return "", errors.Wrapf(ctx, err, "'%v' failed with '%v'", cmd, stdout.String())
	}

	return stdout.String(), nil
}
