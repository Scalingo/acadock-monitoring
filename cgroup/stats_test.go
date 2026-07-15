package cgroup

import (
	"testing"

	statsV1 "github.com/containerd/cgroups/v3/cgroup1/stats"
	statsV2 "github.com/containerd/cgroups/v3/cgroup2/stats"
	"github.com/stretchr/testify/require"
)

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
	})

	require.Equal(t, IOUsage{Devices: []IODeviceUsage{
		{
			Device:     "sda",
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
	})

	require.Equal(t, IOUsage{Devices: []IODeviceUsage{
		{
			Major:      8,
			Minor:      0,
			ReadBytes:  10,
			WriteBytes: 20,
			ReadIOs:    1,
			WriteIOs:   2,
		},
	}}, usage)
}
