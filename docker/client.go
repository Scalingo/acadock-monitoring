package docker

import (
	"sync"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"

	"github.com/Scalingo/acadock-monitoring/config"
)

var (
	_client     *docker.Client
	_clientOnce sync.Once
)

func Client() (*docker.Client, error) {
	var err error

	_clientOnce.Do(func() {
		_client, err = docker.NewClient(config.ENV["DOCKER_URL"])
	})
	if err != nil {
		return nil, errors.Wrap(err, "fail to create docker client")
	}

	return _client, nil
}
