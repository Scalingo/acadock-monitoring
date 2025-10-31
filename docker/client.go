package docker

import (
	"context"

	docker "github.com/docker/docker/client"

	"github.com/Scalingo/acadock-monitoring/v2/config"
	"github.com/Scalingo/go-utils/errors/v3"
)

func Client(ctx context.Context) (*docker.Client, error) {
	client, err := docker.NewClientWithOpts(docker.WithHost(config.ENV["DOCKER_URL"]), docker.WithAPIVersionNegotiation())
	if err != nil {
		return nil, errors.Wrap(ctx, err, "create docker client")
	}

	return client, nil
}
