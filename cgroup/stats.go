package cgroup

import (
	"context"
	"strconv"
	"strings"
	"time"

	statsV1 "github.com/containerd/cgroups/v3/cgroup1/stats"
	statsV2 "github.com/containerd/cgroups/v3/cgroup2/stats"

	"github.com/Scalingo/acadock-monitoring/v2/procfs"
	"github.com/Scalingo/go-utils/errors/v3"
)

type StatsReaderImpl struct {
	mountInfos procfs.MountInfos
}

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
	IOUsage        IOUsage
}

type IOUsage struct {
	Devices []IODeviceUsage
}

type IODeviceUsage struct {
	Device     string
	DevicePath string
	Mountpoint string
	Major      uint64
	Minor      uint64
	ReadBytes  uint64
	WriteBytes uint64
	ReadIOs    uint64
	WriteIOs   uint64
}

func NewStatsReader(mountInfos procfs.MountInfos) *StatsReaderImpl {
	return &StatsReaderImpl{mountInfos: mountInfos}
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
		IOUsage:     cgroupV2IOUsage(stats.Io, r.mountInfos),
	}, nil
}

func (r *StatsReaderImpl) getCgroupV1Stats(ctx context.Context, manager *Manager) (Stats, error) {
	stats, err := manager.V1Manager().Stat()
	if err != nil {
		return Stats{}, errors.Wrap(ctx, err, "get cgroup v1 stats")
	}

	return cgroupV1Stats(stats, r.mountInfos), nil
}

func cgroupV1Stats(stats *statsV1.Metrics, mountInfos procfs.MountInfos) Stats {
	cpuUsage := stats.GetCPU().GetUsage()
	memoryUsage := stats.GetMemory().GetUsage()
	memorySwap := stats.GetMemory().GetSwap()

	return Stats{
		CPUUsage:       time.Duration(cpuUsage.GetTotal()) * time.Nanosecond,
		MemoryUsage:    memoryUsage.GetUsage(),
		MemoryMaxUsage: memoryUsage.GetMax(),
		MemoryLimit:    memoryUsage.GetLimit(),
		// In cgroupv1, swap metrics is the sum of memory + swap, here we make it
		// independent them by making a difference
		SwapUsage:    cgroupV1SwapMetric(memorySwap.GetUsage(), memoryUsage.GetUsage()),
		SwapMaxUsage: cgroupV1SwapMetric(memorySwap.GetMax(), memoryUsage.GetMax()),
		SwapLimit:    cgroupV1SwapMetric(memorySwap.GetLimit(), memoryUsage.GetLimit()),
		IOUsage:      cgroupV1IOUsage(stats.GetBlkio(), mountInfos),
	}
}

func cgroupV1SwapMetric(memoryAndSwap uint64, memory uint64) uint64 {
	if memoryAndSwap < memory {
		return 0
	}

	return memoryAndSwap - memory
}

func cgroupV2IOUsage(ioStat *statsV2.IOStat, mountInfos procfs.MountInfos) IOUsage {
	if ioStat == nil {
		return IOUsage{}
	}

	devices := make([]IODeviceUsage, 0, len(ioStat.Usage))
	for _, entry := range ioStat.Usage {
		devices = append(devices, IODeviceUsage{
			DevicePath: mountInfos.DevicePath(entry.Major, entry.Minor),
			Mountpoint: mountInfos.Mountpoint(entry.Major, entry.Minor),
			Major:      entry.Major,
			Minor:      entry.Minor,
			ReadBytes:  entry.Rbytes,
			WriteBytes: entry.Wbytes,
			ReadIOs:    entry.Rios,
			WriteIOs:   entry.Wios,
		})
	}

	return IOUsage{Devices: devices}
}

func cgroupV1IOUsage(blkioStat *statsV1.BlkIOStat, mountInfos procfs.MountInfos) IOUsage {
	if blkioStat == nil {
		return IOUsage{}
	}

	devices := make(map[string]*IODeviceUsage)
	order := make([]string, 0)

	for _, entry := range blkioStat.IoServiceBytesRecursive {
		updateCgroupV1Device(devices, &order, entry, mountInfos, func(device *IODeviceUsage, op string, value uint64) {
			switch op {
			case "read":
				device.ReadBytes += value
			case "write":
				device.WriteBytes += value
			}
		})
	}

	for _, entry := range blkioStat.IoServicedRecursive {
		updateCgroupV1Device(devices, &order, entry, mountInfos, func(device *IODeviceUsage, op string, value uint64) {
			switch op {
			case "read":
				device.ReadIOs += value
			case "write":
				device.WriteIOs += value
			}
		})
	}

	usage := IOUsage{Devices: make([]IODeviceUsage, 0, len(order))}
	for _, key := range order {
		usage.Devices = append(usage.Devices, *devices[key])
	}

	return usage
}

func updateCgroupV1Device(devices map[string]*IODeviceUsage, order *[]string, entry *statsV1.BlkIOEntry, mountInfos procfs.MountInfos, update func(device *IODeviceUsage, op string, value uint64)) {
	op := strings.ToLower(entry.Op)
	if op != "read" && op != "write" {
		return
	}

	key := entry.Device
	if key == "" {
		key = strconv.FormatUint(entry.Major, 10) + ":" + strconv.FormatUint(entry.Minor, 10)
	}
	device, ok := devices[key]
	if !ok {
		device = &IODeviceUsage{
			Device:     entry.Device,
			DevicePath: mountInfos.DevicePath(entry.Major, entry.Minor),
			Mountpoint: mountInfos.Mountpoint(entry.Major, entry.Minor),
			Major:      entry.Major,
			Minor:      entry.Minor,
		}
		devices[key] = device
		*order = append(*order, key)
	}

	update(device, op, entry.Value)
}
