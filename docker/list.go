package docker

import (
	"github.com/fsouza/go-dockerclient"
	"gopkg.in/errgo.v1"
)

func ListRunningContainers(ids chan string) error {
	client, err := Client()
	if err != nil {
		return errgo.Mask(err)
	}

	containers, err := client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return errgo.Mask(err)
	}

	for _, container := range containers {
		ids <- container.ID
	}
	close(ids)

	return nil
}
