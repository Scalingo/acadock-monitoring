package docker

import (
	docker "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"
)

func ListContainers() ([]docker.APIContainers, error) {
	client, err := Client()
	if err != nil {
		return nil, errors.Wrap(err, "fail to get docker client")
	}

	containers, err := client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "fail to list docker containers")
	}

	return containers, nil
}
