package docker

import (
	"github.com/fsouza/go-dockerclient"
	"gopkg.in/errgo.v1"
)

func ListContainers() ([]docker.APIContainers, error) {
	client, err := Client()
	if err != nil {
		return nil, errgo.Mask(err)
	}

	containers, err := client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return nil, errgo.Mask(err)
	}

	return containers, nil
}
