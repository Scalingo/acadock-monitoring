package mem

import (
	"context"

	"github.com/Scalingo/acadock-monitoring/cgroup"
	"github.com/Scalingo/acadock-monitoring/client"

	"github.com/Scalingo/go-utils/errors/v3"
)

type Usage struct {
	client.MemoryUsage
}

type MemoryUsageGetter struct {
	cgroupStatsReader cgroup.StatsReader
}

func NewMemoryUsageGetter() MemoryUsageGetter {
	return MemoryUsageGetter{
		cgroupStatsReader: *cgroup.NewStatsReader(),
	}
}

func (m MemoryUsageGetter) GetMemoryUsage(ctx context.Context, id string) (client.MemoryUsage, error) {
	stats, err := m.cgroupStatsReader.GetStats(ctx, id)
	if err != nil {
		return client.MemoryUsage{}, errors.Wrap(ctx, err, "get cgroup stats")
	}

	return client.MemoryUsage{
		MemoryUsage:    stats.MemoryUsage,
		MemoryLimit:    stats.MemoryLimit,
		MaxMemoryUsage: stats.MemoryMaxUsage,
		SwapUsage:      stats.SwapUsage,
		SwapLimit:      stats.SwapLimit,
		MaxSwapUsage:   stats.SwapMaxUsage,
	}, nil
}
