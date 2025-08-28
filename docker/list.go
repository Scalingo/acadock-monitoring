package docker

import (
	"context"

	"github.com/docker/docker/api/types/container"

	"github.com/Scalingo/go-utils/errors/v3"
)

func ListContainers(ctx context.Context) ([]container.Summary, error) {
	client, err := Client(ctx)
	if err != nil {
		return nil, errors.Wrap(ctx, err, "get docker client")
	}

	containers, err := client.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(ctx, err, "list docker containers")
	}

	return containers, nil
}
