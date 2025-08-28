package docker

import (
	"context"

	docker "github.com/fsouza/go-dockerclient"

	"github.com/Scalingo/go-utils/errors/v3"
)

func ListContainers(ctx context.Context) ([]docker.APIContainers, error) {
	client, err := Client(ctx)
	if err != nil {
		return nil, errors.Wrap(ctx, err, "get docker client")
	}

	containers, err := client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return nil, errors.Wrap(ctx, err, "list docker containers")
	}

	return containers, nil
}
