package netns

import (
	"fmt"
	"testing"

	"github.com/fsouza/go-dockerclient"
)

func TestSetns(t *testing.T) {
	container := startContainer()
	defer destroyContainer(container)
	pid := container.State.Pid

	ns, err := Setns(fmt.Sprintf("%d", pid))
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	stats, err := netstat.Stats()
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if len(stats) != 2 {
		t.Errorf("expected len(stats) == 2, got %v", len(stats))
	}
	if stats[0].Interface != "eth0" {
		t.Errorf("expected stats[0].Interface == eth0, got %v", stats[0].Interface)
	}
	if stats[1].Interface != "lo" {
		t.Errorf("expected stats[1].Interface == lo, got %v", stats[1].Interface)
	}
	err = ns.Close()
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func startContainer() *docker.Container {
	client := dockerClient()
	opts := docker.PullImageOptions{}
	opts.Repository = "busybox"
	opts.Tag = "latest"
	auth := docker.AuthConfiguration{}
	err := client.PullImage(opts, auth)
	panicOnError(err)
	createOpts := docker.CreateContainerOptions{}
	createOpts.Config = &docker.Config{
		Image: "busybox:latest",
		Cmd:   []string{"sleep", "60"},
	}
	container, err := client.CreateContainer(createOpts)
	panicOnError(err)

	hostOpts := &docker.HostConfig{}
	err = client.StartContainer(container.ID, hostOpts)
	panicOnError(err)

	container, err = client.InspectContainer(container.ID)
	panicOnError(err)

	return container
}

func destroyContainer(container *docker.Container) {
	client := dockerClient()
	err := client.KillContainer(docker.KillContainerOptions{ID: container.ID})
	panicOnError(err)
	err = client.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID})
	panicOnError(err)
}

func dockerClient() *docker.Client {
	client, err := docker.NewClient("unix:///var/run/docker.sock")
	panicOnError(err)
	return client
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
