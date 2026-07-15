package cgroup

import (
	"context"
	"time"

	"github.com/containerd/cgroups/v3/cgroup1/stats"

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

func NewStatsReaderError(err error) StatsReaderError {
	return StatsReaderError{err: err}
}

func (r *StatsReaderImpl) GetStats(ctx context.Context, containerID string) (Stats, error) {
	manager, err := NewManager(ctx, containerID)
	if err != nil {
		return Stats{}, NewStatsReaderError(errors.Wrap(ctx, err, "create cgroup manager"))
	}
	var stats Stats
	if manager.IsV2() {
		stats, err = r.getCgroupV2Stats(ctx, manager)
	} else {
		stats, err = r.getCgroupV1Stats(ctx, manager)
	}
	if err != nil {
		return Stats{}, NewStatsReaderError(errors.Wrap(ctx, err, "get cgroup stats"))
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
	cgroupStats, err := manager.V1Manager().Stat()
	if err != nil {
		return Stats{}, errors.Wrap(ctx, err, "get cgroup v1 stats")
	}

	memoryStats := &stats.MemoryEntry{}
	swapStats := &stats.MemoryEntry{}
	if cgroupStats.Memory != nil && cgroupStats.Memory.Usage != nil {
		memoryStats = cgroupStats.Memory.Usage
		swapStats = cgroupStats.Memory.Swap
	}
	return Stats{
		CPUUsage:       time.Duration(cgroupStats.CPU.Usage.Total) * time.Nanosecond,
		MemoryUsage:    memoryStats.Usage,
		MemoryMaxUsage: memoryStats.Max,
		MemoryLimit:    memoryStats.Limit,
		// In cgroupv1, swap metrics is the sum of memory + swap, here we make it
		// independent them by making a difference
		SwapUsage:    swapStats.Usage - memoryStats.Usage,
		SwapMaxUsage: swapStats.Max - memoryStats.Max,
		SwapLimit:    swapStats.Limit - memoryStats.Limit,
	}, nil
}
