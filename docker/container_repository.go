package docker

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/go-utils/logger"
)

type ContainerRepository interface {
	RegisterToContainersStream(ctx context.Context) <-chan string
}

type ContainerRepositoryImpl struct {
	registeredChans   []chan string
	registrationMutex *sync.Mutex
}

func NewContainerRepository() *ContainerRepositoryImpl {
	return &ContainerRepositoryImpl{
		registeredChans:   make([]chan string, 0),
		registrationMutex: &sync.Mutex{},
	}
}

func (r *ContainerRepositoryImpl) StartListeningToNewContainers(ctx context.Context) {
	containers := make(chan string)
	go r.listenNewContainers(ctx, containers)
	go func() {
		for c := range containers {
			r.registrationMutex.Lock()
			for _, registeredChan := range r.registeredChans {
				registeredChan <- c
			}
			r.registrationMutex.Unlock()
		}
	}()
}

func (r *ContainerRepositoryImpl) RegisterToContainersStream(ctx context.Context) <-chan string {
	log := logger.Get(ctx)
	c := make(chan string, 1)
	r.registrationMutex.Lock()
	defer r.registrationMutex.Unlock()
	r.registeredChans = append(r.registeredChans, c)
	go func(c chan string) {
		containers, err := ListContainers(ctx)
		if err != nil {
			log.WithError(err).Warn("register-chan fail to list containers")
			return
		}
		for _, container := range containers {
			c <- container.ID
		}
	}(c)
	return c
}

func (r *ContainerRepositoryImpl) listenNewContainers(ctx context.Context, ids chan string) error {
	log := logger.Get(ctx)
	client, err := Client(ctx)
	if err != nil {
		return errors.Wrap(ctx, err, "get docker client")
	}
	defer client.Close()

	filters := filters.NewArgs()
	filters.Add("type", "container")
	filters.Add("event", "start")

	for {
		events, errs := client.Events(ctx, events.ListOptions{
			Filters: filters,
		})
		if err != nil {
			return errors.Wrap(ctx, err, "add event listener")
		}

		go func() {
			for event := range events {
				ids <- event.ID
			}
		}()

		err := <-errs
		if err == io.ErrUnexpectedEOF || err == io.EOF {
			log.WithError(err).Info("Connection lost to docker, reconnecting immediately...")
			// Not really immediately to prevent high CPU usage infinite loop during
			// docker restart which can last few dozens of seconds
			time.Sleep(250 * time.Millisecond)
		} else if err != nil {
			log.WithError(err).Error("Docker event listener error, restarting in 5 seconds...")
			time.Sleep(5 * time.Second)
		}
	}
}
