package cgroup

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	statsV1 "github.com/containerd/cgroups/v3/cgroup1/stats"
	"github.com/containerd/cgroups/v3/cgroup2"
	statsV2 "github.com/containerd/cgroups/v3/cgroup2/stats"
	"github.com/stretchr/testify/require"
)

type fakeMountInfos map[string]string

func (f fakeMountInfos) Mountpoint(major uint64, minor uint64) string {
	return f["mount:"+deviceKey(major, minor)]
}

func (f fakeMountInfos) DevicePath(major uint64, minor uint64) string {
	return f["device:"+deviceKey(major, minor)]
}

func deviceKey(major uint64, minor uint64) string {
	return strconv.FormatUint(major, 10) + ":" + strconv.FormatUint(minor, 10)
}

func TestGetCgroupV2StatsMapsPeakUsage(t *testing.T) {
	mountpoint := t.TempDir()
	cgroupPath := filepath.Join(mountpoint, "test")
	require.NoError(t, os.Mkdir(cgroupPath, 0o755))

	for filename, content := range map[string]string{
		"cpu.stat":            "usage_usec 42\n",
		"memory.stat":         "anon 0\n",
		"memory.current":      "10\n",
		"memory.max":          "30\n",
		"memory.peak":         "20\n",
		"memory.swap.current": "15\n",
		"memory.swap.max":     "41\n",
		"memory.swap.peak":    "27\n",
	} {
		require.NoError(t, os.WriteFile(filepath.Join(cgroupPath, filename), []byte(content), 0o600))
	}

	cgroupManager, err := cgroup2.Load("/test", cgroup2.WithMountpoint(mountpoint))
	require.NoError(t, err)

	stats, err := NewStatsReader(fakeMountInfos{}).getCgroupV2Stats(t.Context(), &Manager{
		cgroupV2Manager: cgroupManager,
	})
	require.NoError(t, err)
	require.Equal(t, Stats{
		CPUUsage:       42 * time.Microsecond,
		MemoryUsage:    10,
		MemoryMaxUsage: 20,
		MemoryLimit:    30,
		SwapUsage:      15,
		SwapMaxUsage:   27,
		SwapLimit:      41,
		IOUsage:        IOUsage{Devices: []IODeviceUsage{}},
	}, stats)
}

func TestCgroupV1StatsMapsStats(t *testing.T) {
	stats := cgroupV1Stats(&statsV1.Metrics{
		CPU: &statsV1.CPUStat{Usage: &statsV1.CPUUsage{Total: 42}},
		Memory: &statsV1.MemoryStat{
			Usage: &statsV1.MemoryEntry{Usage: 10, Max: 20, Limit: 30},
			Swap:  &statsV1.MemoryEntry{Usage: 15, Max: 27, Limit: 41},
		},
	}, fakeMountInfos{})

	require.Equal(t, Stats{
		CPUUsage:       42 * time.Nanosecond,
		MemoryUsage:    10,
		MemoryMaxUsage: 20,
		MemoryLimit:    30,
		SwapUsage:      5,
		SwapMaxUsage:   7,
		SwapLimit:      11,
		IOUsage:        IOUsage{},
	}, stats)
}

func TestCgroupV1StatsHandlesMissingSections(t *testing.T) {
	stats := cgroupV1Stats(&statsV1.Metrics{}, fakeMountInfos{})

	require.Equal(t, Stats{IOUsage: IOUsage{}}, stats)
}

func TestCgroupV1StatsHandlesMissingSwap(t *testing.T) {
	stats := cgroupV1Stats(&statsV1.Metrics{
		Memory: &statsV1.MemoryStat{
			Usage: &statsV1.MemoryEntry{Usage: 10, Max: 20, Limit: 30},
		},
	}, fakeMountInfos{})

	require.Equal(t, Stats{
		MemoryUsage:    10,
		MemoryMaxUsage: 20,
		MemoryLimit:    30,
		IOUsage:        IOUsage{},
	}, stats)
}

func TestCgroupV1IOUsageAggregatesReadAndWriteStats(t *testing.T) {
	usage := cgroupV1IOUsage(&statsV1.BlkIOStat{
		IoServiceBytesRecursive: []*statsV1.BlkIOEntry{
			{Device: "sda", Major: 8, Minor: 0, Op: "Read", Value: 10},
			{Device: "sda", Major: 8, Minor: 0, Op: "Write", Value: 20},
			{Device: "sda", Major: 8, Minor: 0, Op: "Total", Value: 30},
		},
		IoServicedRecursive: []*statsV1.BlkIOEntry{
			{Device: "sda", Major: 8, Minor: 0, Op: "Read", Value: 1},
			{Device: "sda", Major: 8, Minor: 0, Op: "Write", Value: 2},
			{Device: "sda", Major: 8, Minor: 0, Op: "Total", Value: 3},
		},
	}, fakeMountInfos{"mount:8:0": "/var/lib", "device:8:0": "/dev/sda"})

	require.Equal(t, IOUsage{Devices: []IODeviceUsage{
		{
			DevicePath: "/dev/sda",
			Mountpoint: "/var/lib",
			Major:      8,
			Minor:      0,
			ReadBytes:  10,
			WriteBytes: 20,
			ReadIOs:    1,
			WriteIOs:   2,
		},
	}}, usage)
}

func TestCgroupV2IOUsageMapsStats(t *testing.T) {
	usage := cgroupV2IOUsage(&statsV2.IOStat{
		Usage: []*statsV2.IOEntry{
			{Major: 8, Minor: 0, Rbytes: 10, Wbytes: 20, Rios: 1, Wios: 2},
		},
	}, fakeMountInfos{"mount:8:0": "/var/lib", "device:8:0": "/dev/sda"})

	require.Equal(t, IOUsage{Devices: []IODeviceUsage{
		{
			DevicePath: "/dev/sda",
			Mountpoint: "/var/lib",
			Major:      8,
			Minor:      0,
			ReadBytes:  10,
			WriteBytes: 20,
			ReadIOs:    1,
			WriteIOs:   2,
		},
	}}, usage)
}
