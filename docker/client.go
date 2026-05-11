package docker

import (
	"context"

	dockerclient "github.com/moby/moby/client"

	"github.com/Scalingo/acadock-monitoring/v2/config"
	"github.com/Scalingo/go-utils/errors/v3"
)

func Client(ctx context.Context) (*dockerclient.Client, error) {
	client, err := dockerclient.New(dockerclient.WithHost(config.ENV["DOCKER_URL"]))
	if err != nil {
		return nil, errors.Wrap(ctx, err, "create docker client")
	}

	return client, nil
}
