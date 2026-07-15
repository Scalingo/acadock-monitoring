package resources

import (
	"context"

	"github.com/Scalingo/acadock-monitoring/v2/cgroup"
	"github.com/Scalingo/acadock-monitoring/v2/client"

	"github.com/Scalingo/go-utils/errors/v3"
)

type UsageGetter struct {
	cgroupStatsReader cgroup.StatsReader
}

type Usage struct {
	Memory client.MemoryUsage
	IO     client.IOUsage
}

func NewUsageGetter(cgroupStatsReader cgroup.StatsReader) UsageGetter {
	return UsageGetter{
		cgroupStatsReader: cgroupStatsReader,
	}
}

func (g UsageGetter) GetMemoryUsage(ctx context.Context, id string) (client.MemoryUsage, error) {
	stats, err := g.cgroupStatsReader.GetStats(ctx, id)
	if err != nil {
		return client.MemoryUsage{}, errors.Wrap(ctx, err, "get cgroup stats")
	}

	return memoryUsageFromStats(stats), nil
}

func (g UsageGetter) GetUsage(ctx context.Context, id string) (Usage, error) {
	stats, err := g.cgroupStatsReader.GetStats(ctx, id)
	if err != nil {
		return Usage{}, errors.Wrap(ctx, err, "get cgroup stats")
	}

	return Usage{
		Memory: memoryUsageFromStats(stats),
		IO:     ioUsageFromStats(stats),
	}, nil
}

func (g UsageGetter) GetIOUsage(ctx context.Context, id string) (client.IOUsage, error) {
	stats, err := g.cgroupStatsReader.GetStats(ctx, id)
	if err != nil {
		return client.IOUsage{}, errors.Wrap(ctx, err, "get cgroup stats")
	}

	return ioUsageFromStats(stats), nil
}

func memoryUsageFromStats(stats cgroup.Stats) client.MemoryUsage {
	return client.MemoryUsage{
		MemoryUsage:    stats.MemoryUsage,
		MemoryLimit:    stats.MemoryLimit,
		MaxMemoryUsage: stats.MemoryMaxUsage,
		SwapUsage:      stats.SwapUsage,
		SwapLimit:      stats.SwapLimit,
		MaxSwapUsage:   stats.SwapMaxUsage,
	}
}

func ioUsageFromStats(stats cgroup.Stats) client.IOUsage {
	devices := make([]client.IODeviceUsage, 0, len(stats.IOUsage.Devices))
	for _, device := range stats.IOUsage.Devices {
		devices = append(devices, client.IODeviceUsage{
			DevicePath: device.DevicePath,
			Mountpoint: device.Mountpoint,
			Major:      device.Major,
			Minor:      device.Minor,
			ReadBytes:  device.ReadBytes,
			WriteBytes: device.WriteBytes,
			ReadIOs:    device.ReadIOs,
			WriteIOs:   device.WriteIOs,
		})
	}

	return client.IOUsage{Devices: devices}
}
