package net

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"

	"github.com/Scalingo/go-netns"
)

func NsIfaceIDByPID(pid string) (int, error) {
	ns, err := netns.SetnsFromProcDir(os.Getenv("PROC_DIR") + "/" + pid)
	if err != nil {
		return -1, err
	}
	defer ns.Close()

	stdout := new(bytes.Buffer)
	cmd := exec.Command("ip", "link", "show", "eth0")
	cmd.Stdout = stdout
	cmd.Stderr = stdout
	err = cmd.Start()
	if err != nil {
		return -1, err
	}
	err = cmd.Wait()
	if err != nil {
		return -1, fmt.Errorf("%v: %v", err, stdout.String())
	}

	// Example of output
	// 614: eth0@if615: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP mode DEFAULT group default
	// We want the ID 615
	out := stdout.String()
	matches := regexp.MustCompile(`@if(\d+):`).FindAllStringSubmatch(out, -1)
	if len(matches) != 1 && len(matches[0]) != 2 {
		return -1, fmt.Errorf("parsing iface ID: no match in '%v'", out)
	}

	ifaceID, err := strconv.Atoi(matches[0][1])
	if err != nil {
		return -1, fmt.Errorf("parsing iface ID: %v", err)
	}
	return ifaceID, nil
}
