package cgroup

import (
	"context"
	"fmt"

	"github.com/Scalingo/acadock-monitoring/config"

	"github.com/containerd/cgroups"
	cgroupsv2 "github.com/containerd/cgroups/v2"

	"github.com/Scalingo/go-utils/errors/v3"
)

type Manager struct {
	cgroupV1Manager cgroups.Cgroup
	cgroupV2Manager *cgroupsv2.Manager
	v2              bool
	systemd         bool
}

func NewManager(ctx context.Context, containerID string) (*Manager, error) {
	var err error
	manager := &Manager{
		v2:      config.IsUsingCgroupV2,
		systemd: config.ENV["CGROUP_SOURCE"] == "systemd" || config.IsUsingCgroupV2,
	}

	if manager.v2 {
		manager.cgroupV2Manager, err = cgroupsv2.LoadSystemd("/system.slice", fmt.Sprintf("docker-%s.scope", containerID))
	} else if manager.systemd {
		manager.cgroupV1Manager, err = cgroups.Load(cgroups.Systemd, cgroups.Slice("system.slice", fmt.Sprintf("docker-%s.scope", containerID)))
	} else {
		manager.cgroupV1Manager, err = cgroups.Load(cgroups.V1, cgroups.StaticPath("docker/"+containerID))
	}
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "load cgroup, systemd: %v, v2: %v", manager.systemd, manager.v2)
	}

	return manager, nil
}

func (m *Manager) IsV2() bool {
	return m.v2
}

func (m *Manager) V1Manager() cgroups.Cgroup {
	return m.cgroupV1Manager
}

func (m *Manager) V2Manager() *cgroupsv2.Manager {
	return m.cgroupV2Manager
}

func (m *Manager) Pids(ctx context.Context) ([]uint64, error) {
	if m.v2 {
		return m.cgroupV2Manager.Procs(false)
	}
	tasks, err := m.cgroupV1Manager.Tasks("memory", false)
	if err != nil {
		return nil, errors.Wrap(ctx, err, "get cgroup v1 tasks")
	}
	var pids []uint64
	for _, t := range tasks {
		pids = append(pids, uint64(t.Pid))
	}
	return pids, nil
}
