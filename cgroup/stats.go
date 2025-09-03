package cgroup

import (
	"context"
	"time"

	"github.com/Scalingo/go-utils/errors/v3"
)

type StatsReaderImpl struct{}

type StatsReader interface {
	GetStats(ctx context.Context, containerID string) (Stats, error)
}

type Stats struct {
	CPUUsage       time.Duration
	MemoryUsage    uint64
	MemoryMaxUsage uint64
	MemoryLimit    uint64
	SwapUsage      uint64
	SwapMaxUsage   uint64
	SwapLimit      uint64
}

func NewStatsReader() *StatsReaderImpl {
	return &StatsReaderImpl{}
}

type StatsReaderError struct {
	err error
}

func (e StatsReaderError) Error() string {
	return e.err.Error()
}

func (e StatsReaderError) Unwrap() error {
	return e.err
}

func (r *StatsReaderImpl) GetStats(ctx context.Context, containerID string) (Stats, error) {
	manager, err := NewManager(ctx, containerID)
	if err != nil {
		return Stats{}, errors.Wrap(ctx, err, "create cgroup manager")
	}
	var stats Stats
	if manager.IsV2() {
		stats, err = r.getCgroupV2Stats(ctx, manager)
	} else {
		stats, err = r.getCgroupV1Stats(ctx, manager)
	}
	if err != nil {
		return Stats{}, StatsReaderError{err: errors.Wrap(ctx, err, "get cgroup stats")}
	}
	return stats, nil
}

func (r *StatsReaderImpl) getCgroupV2Stats(ctx context.Context, manager *Manager) (Stats, error) {
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

func (r *StatsReaderImpl) getCgroupV1Stats(ctx context.Context, manager *Manager) (Stats, error) {
	stats, err := manager.V1Manager().Stat()
	if err != nil {
		return Stats{}, errors.Wrap(ctx, err, "get cgroup v1 stats")
	}
	return Stats{
		CPUUsage:       time.Duration(stats.CPU.Usage.Total) * time.Nanosecond,
		MemoryUsage:    stats.Memory.Usage.Usage,
		MemoryMaxUsage: stats.Memory.Usage.Max,
		MemoryLimit:    stats.Memory.Usage.Limit,
		// In cgroupv1, swap metrics is the sum of memory + swap, here we make it
		// independent them by making a difference
		SwapUsage:    stats.Memory.Swap.Usage - stats.Memory.Usage.Usage,
		SwapMaxUsage: stats.Memory.Swap.Max - stats.Memory.Usage.Max,
		SwapLimit:    stats.Memory.Swap.Limit - stats.Memory.Usage.Limit,
	}, nil
}
