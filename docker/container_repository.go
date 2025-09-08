package docker

import (
	"context"
	"io"
	"sync"
	"time"

	dockerevents "github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/go-utils/logger"
)

type ContainerEvent struct {
	ContainerID string
	Action      ContainerAction
}

type ContainerAction = dockerevents.Action

const (
	ContainerActionStart = dockerevents.ActionStart
	ContainerActionStop  = dockerevents.ActionStop
)

type ContainerRepository interface {
	RegisterToContainersStream(ctx context.Context) <-chan ContainerEvent
}

type ContainerRepositoryImpl struct {
	registeredChans   []chan ContainerEvent
	registrationMutex *sync.Mutex
}

func NewContainerRepository() *ContainerRepositoryImpl {
	return &ContainerRepositoryImpl{
		registeredChans:   make([]chan ContainerEvent, 0),
		registrationMutex: &sync.Mutex{},
	}
}

func (r *ContainerRepositoryImpl) StartListeningToNewContainers(ctx context.Context) {
	eventsChan := make(chan ContainerEvent)
	go r.listenToDockerEvents(ctx, eventsChan)
	go func() {
		for c := range eventsChan {
			r.registrationMutex.Lock()
			for _, registeredChan := range r.registeredChans {
				registeredChan <- c
			}
			r.registrationMutex.Unlock()
		}
	}()
}

func (r *ContainerRepositoryImpl) RegisterToContainersStream(ctx context.Context) <-chan ContainerEvent {
	log := logger.Get(ctx)
	registration := make(chan ContainerEvent, 1)
	r.registrationMutex.Lock()
	defer r.registrationMutex.Unlock()
	r.registeredChans = append(r.registeredChans, registration)
	go func(registration chan ContainerEvent) {
		containers, err := ListContainers(ctx)
		if err != nil {
			log.WithError(err).Warn("register-chan fail to list containers")
			return
		}
		for _, container := range containers {
			registration <- ContainerEvent{
				ContainerID: container.ID,
				Action:      ContainerActionStart,
			}
		}
	}(registration)
	return registration
}

func (r *ContainerRepositoryImpl) listenToDockerEvents(ctx context.Context, events chan ContainerEvent) error {
	log := logger.Get(ctx)
	client, err := Client(ctx)
	if err != nil {
		return errors.Wrap(ctx, err, "get docker client")
	}
	defer client.Close()

	filters := filters.NewArgs()
	filters.Add("type", "container")
	filters.Add("event", string(ContainerActionStart))
	filters.Add("event", string(ContainerActionStop))

	for {
		dockerEventsReceiver, errs := client.Events(ctx, dockerevents.ListOptions{
			Filters: filters,
		})
		if err != nil {
			return errors.Wrap(ctx, err, "add event listener")
		}

		go func() {
			for event := range dockerEventsReceiver {
				events <- ContainerEvent{
					ContainerID: event.Actor.ID,
					Action:      event.Action,
				}
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
