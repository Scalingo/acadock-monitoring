package docker

import (
	docker "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"
)

func ListenNewContainers(ids chan string) error {
	client, err := Client()
	if err != nil {
		return errors.Wrap(err, "fail to get docker client")
	}

	listener := make(chan *docker.APIEvents)
	err = client.AddEventListener(listener)
	if err != nil {
		return errors.Wrap(err, "fail to add event listener")
	}

	for event := range listener {
		if event.Status == "start" {
			ids <- event.ID
		}
	}
	return nil
}
