package docker

import (
	"context"

	docker "github.com/fsouza/go-dockerclient"

	"github.com/Scalingo/go-utils/errors/v3"
)

func ListenNewContainers(ctx context.Context, ids chan string) error {
	client, err := Client(ctx)
	if err != nil {
		return errors.Wrap(ctx, err, "get docker client")
	}

	listener := make(chan *docker.APIEvents)
	err = client.AddEventListener(listener)
	if err != nil {
		return errors.Wrap(ctx, err, "add event listener")
	}

	for event := range listener {
		if event.Status == "start" {
			ids <- event.ID
		}
	}
	return nil
}
