package docker

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
)

var (
	registeredChans   []chan string
	registrationMutex *sync.Mutex
)

func init() {
	// Should be put into a struct and initialized in main
	ctx := context.TODO()
	registrationMutex = &sync.Mutex{}
	containers := make(chan string)
	go ListenNewContainers(ctx, containers)
	go func() {
		for c := range containers {
			registrationMutex.Lock()
			for _, registeredChan := range registeredChans {
				registeredChan <- c
			}
			registrationMutex.Unlock()
		}
	}()
}

func RegisterToContainersStream(ctx context.Context) chan string {
	c := make(chan string, 1)
	registrationMutex.Lock()
	defer registrationMutex.Unlock()
	registeredChans = append(registeredChans, c)
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
