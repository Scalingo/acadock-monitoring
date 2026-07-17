package docker

import (
	"context"

	dockercontainer "github.com/moby/moby/api/types/container"
	dockerclient "github.com/moby/moby/client"

	"github.com/Scalingo/go-utils/errors/v3"
)

func ListContainers(ctx context.Context) ([]dockercontainer.Summary, error) {
	client, err := Client(ctx)
	if err != nil {
		return nil, errors.Wrap(ctx, err, "get docker client")
	}

	containers, err := client.ContainerList(ctx, dockerclient.ContainerListOptions{})
	if err != nil {
		return nil, errors.Wrap(ctx, err, "list docker containers")
	}

	return containers.Items, nil
}
