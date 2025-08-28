package cgroup

import (
	"context"
	"time"

	"github.com/Scalingo/go-utils/errors/v3"
)

type StatsReader struct{}

type Stats struct {
	CPUUsage       time.Duration
	MemoryUsage    uint64
	MemoryMaxUsage uint64
	MemoryLimit    uint64
	SwapUsage      uint64
	SwapMaxUsage   uint64
	SwapLimit      uint64
}

func NewStatsReader() *StatsReader {
	return &StatsReader{}
}

func (r *StatsReader) GetStats(ctx context.Context, containerID string) (Stats, error) {
	manager, err := NewManager(ctx, containerID)
	if err != nil {
		return Stats{}, errors.Wrap(ctx, err, "create cgroup manager")
	}
	if manager.IsV2() {
		return r.getCgroupV2Stats(ctx, manager)
	}
	return r.getCgroupV1Stats(ctx, manager)
}

func (r *StatsReader) getCgroupV2Stats(ctx context.Context, manager *Manager) (Stats, error) {
	stats, err := manager.V2Manager().Stat()
	if err != nil {
		return Stats{}, errors.Wrap(ctx, err, "get cgroup v2 stats")
	}

	return Stats{
		CPUUsage:    time.Duration(stats.CPU.UsageUsec) * time.Microsecond,
		MemoryUsage: stats.Memory.Usage,
		MemoryLimit: stats.Memory.UsageLimit,
		SwapUsage:   stats.Memory.SwapUsage,
		SwapLimit:   stats.Memory.SwapLimit,
	}, nil
}

func (r *StatsReader) getCgroupV1Stats(ctx context.Context, manager *Manager) (Stats, error) {
	stats, err := manager.V1Manager().Stat()
	if err != nil {
		return Stats{}, errors.Wrap(ctx, err, "get cgroup v1 stats")
	}
	return Stats{
		CPUUsage:       time.Duration(stats.CPU.Usage.Total) * time.Nanosecond,
		MemoryUsage:    stats.Memory.Usage.Usage,
		MemoryMaxUsage: stats.Memory.Usage.Max,
		MemoryLimit:    stats.Memory.Usage.Limit,
		SwapUsage:      stats.Memory.Swap.Usage,
		SwapMaxUsage:   stats.Memory.Swap.Max,
		SwapLimit:      stats.Memory.Swap.Limit,
	}, nil
}
