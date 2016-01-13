package docker

import (
	"sync"

	"github.com/Scalingo/acadock-monitoring/config"
	"github.com/fsouza/go-dockerclient"
	"gopkg.in/errgo.v1"
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
		return nil, errgo.Mask(err)
	}
	return _client, nil
}
