package net

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

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
	nshandler, err := netns.GetFromDocker(id)
	if err != nil {
		return "", errors.Wrapf(ctx, err, "could not get network namespace")
	}
	defer nshandler.Close()
	nlhandler, err := netlink.NewHandleAt(nshandler)
	if err != nil {
		return "", errors.Wrapf(ctx, err, "could not create netlink handle")
	}
	defer nlhandler.Close()

	containerVethLink, err := nlhandler.LinkByName("eth0")
	if err != nil {
		return "", errors.Wrapf(ctx, err, "could not get eth0 link")
	}

	parentContainerVethLinkIndex := containerVethLink.Attrs().ParentIndex
	if parentContainerVethLinkIndex == 0 {
		return "", errors.Errorf(ctx, "could not get veth parent index")
	}

	return strconv.Itoa(parentContainerVethLinkIndex), nil
}
