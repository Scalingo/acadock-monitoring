package docker

import (
	"context"
	"sync"

	docker "github.com/fsouza/go-dockerclient"

	"github.com/Scalingo/acadock-monitoring/config"
	"github.com/Scalingo/go-utils/errors/v3"
)

var (
	_client     *docker.Client
	_clientOnce sync.Once
)

func Client(ctx context.Context) (*docker.Client, error) {
	var err error

	_clientOnce.Do(func() {
		_client, err = docker.NewClient(config.ENV["DOCKER_URL"])
	})
	if err != nil {
		return nil, errors.Wrap(ctx, err, "create docker client")
	}

	return _client, nil
}
