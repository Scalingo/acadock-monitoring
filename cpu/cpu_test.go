package cpu

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/Scalingo/acadock-monitoring/cgroup"
	"github.com/Scalingo/acadock-monitoring/cgroup/cgroupmock"
	"github.com/Scalingo/acadock-monitoring/config"
	"github.com/Scalingo/acadock-monitoring/docker"
	"github.com/Scalingo/acadock-monitoring/docker/dockermock"
	"github.com/Scalingo/acadock-monitoring/procfs"
)

func TestCPUUsageMonitor_Start(t *testing.T) {
	ctrl := gomock.NewController(t)

	cgroupStatsReader := cgroupmock.NewMockStatsReader(ctrl)
	cpuStatsReader := procfs.NewMockCPUStat(ctrl)
	containerRepository := dockermock.NewMockContainerRepository(ctrl)
	dockerID := "1"
	config.RefreshTime = 100 * time.Millisecond

	// 10% usage
	for i := 1; i < 10; i++ {
		expectedCPUStats := procfs.CPUStats{
			CPUs: map[string]procfs.SingleCPUStat{"cpu": {Name: "cpu", User: time.Duration(i * 10 * 10e6), IDLE: time.Duration(i * 90 * 10e6)}},
		}
		cpuStatsReader.EXPECT().Read(gomock.Any()).Return(expectedCPUStats, nil).MaxTimes(1)

		cgroupStats := cgroup.Stats{CPUUsage: time.Duration(i * 10 * 10e6)}
		cgroupStatsReader.EXPECT().GetStats(gomock.Any(), dockerID).Return(cgroupStats, nil).MaxTimes(1)
	}

	eventsChan := make(chan docker.ContainerEvent)
	containerRepository.EXPECT().RegisterToContainersStream(gomock.Any()).Return(eventsChan)
	go func() {
		eventsChan <- docker.ContainerEvent{ContainerID: dockerID, Action: docker.ContainerActionStart}
		time.Sleep(4 * config.RefreshTime)
		eventsChan <- docker.ContainerEvent{ContainerID: dockerID, Action: docker.ContainerActionStop}
		close(eventsChan)
	}()

	monitor := NewCPUUsageMonitor(containerRepository, cpuStatsReader, cgroupStatsReader)
	// Override numCPU to have tests reliable whatever the environment
	monitor.numCPU = 1

	ctx, cancel := context.WithCancel(t.Context())
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		monitor.Start(ctx)
		wg.Done()
	}()

	// After 2 cycles it must have accurate CPU information
	time.Sleep(2 * config.RefreshTime)
	usage, err := monitor.GetContainerUsage(dockerID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, usage.UsageInPercents, 10)

	// Check that the context does stop the Monitor
	cancel()
	wg.Wait()
}
