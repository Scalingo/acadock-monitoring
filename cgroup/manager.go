package cgroup

import (
	"context"
	"fmt"

	"github.com/Scalingo/acadock-monitoring/v2/config"

	"github.com/containerd/cgroups/v3/cgroup1"
	"github.com/containerd/cgroups/v3/cgroup2"

	"github.com/Scalingo/go-utils/errors/v3"
)

type Manager struct {
	cgroupV1Manager cgroup1.Cgroup
	cgroupV2Manager *cgroup2.Manager
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
		manager.cgroupV2Manager, err = cgroup2.LoadSystemd("/system.slice", fmt.Sprintf("docker-%s.scope", containerID))
	} else if manager.systemd {
		manager.cgroupV1Manager, err = cgroup1.Load(
			cgroup1.Slice("system.slice", fmt.Sprintf("docker-%s.scope", containerID)),
			cgroup1.WithHierarchy(cgroup1.Systemd),
		)
	} else {
		manager.cgroupV1Manager, err = cgroup1.Load(cgroup1.StaticPath("docker/" + containerID))
	}
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "load cgroup, systemd: %v, v2: %v", manager.systemd, manager.v2)
	}

	return manager, nil
}

func (m *Manager) IsV2() bool {
	return m.v2
}

func (m *Manager) V1Manager() cgroup1.Cgroup {
	return m.cgroupV1Manager
}

func (m *Manager) V2Manager() *cgroup2.Manager {
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
