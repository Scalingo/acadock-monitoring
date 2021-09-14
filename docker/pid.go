package docker

import (
	"os"
	"strings"

	"github.com/Scalingo/acadock-monitoring/config"
)

func Pid(id string) (string, error) {
	path := config.CgroupPath("memory", id)
	content, err := os.ReadFile(path + "/tasks")
	if err != nil {
		return "", err
	}
	return strings.Split(string(content), "\n")[0], nil
}
