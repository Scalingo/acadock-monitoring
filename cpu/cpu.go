package cpu

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/Scalingo/acadock-monitoring/cgroup"
	"github.com/Scalingo/acadock-monitoring/client"
	"github.com/Scalingo/acadock-monitoring/config"
	"github.com/Scalingo/acadock-monitoring/docker"
	"github.com/Scalingo/acadock-monitoring/procfs"
	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/go-utils/logger"
)

type Usage client.CpuUsage

type CPUUsageMonitor struct {
	containerRepository    docker.ContainerRepository
	numCPU                 int
	currentHostUsage       *procfs.SingleCPUStat
	previousHostUsage      *procfs.SingleCPUStat
	currentSystemUsage     map[string]time.Duration
	previousSystemUsage    map[string]time.Duration
	currentContainerStats  map[string]cgroup.Stats
	previousContainerStats map[string]cgroup.Stats
	cpuUsagesMutex         *sync.Mutex
	cpuStatReader          procfs.CPUStat
	cgroupStatsReader      cgroup.StatsReader
}

func NewCPUUsageMonitor(containerRepository docker.ContainerRepository, cpustat procfs.CPUStat, cgroupStatsReader cgroup.StatsReader) *CPUUsageMonitor {
	return &CPUUsageMonitor{
		containerRepository:    containerRepository,
		numCPU:                 runtime.NumCPU(),
		currentSystemUsage:     make(map[string]time.Duration),
		previousSystemUsage:    make(map[string]time.Duration),
		previousContainerStats: make(map[string]cgroup.Stats),
		currentContainerStats:  make(map[string]cgroup.Stats),
		cpuUsagesMutex:         &sync.Mutex{},
		cpuStatReader:          cpustat,
		cgroupStatsReader:      cgroupStatsReader,
	}
}

func (m *CPUUsageMonitor) Start(ctx context.Context) {
	go m.monitorHostUsage(ctx)

	cancels := map[string]context.CancelFunc{}
	events := m.containerRepository.RegisterToContainersStream(ctx)
	for event := range events {
		ctx, log := logger.WithFieldToCtx(ctx, "container_id", event.ContainerID)
		switch event.Action {
		case docker.ContainerActionStart:
			ctx, cancel := context.WithCancel(ctx)
			cancels[event.ContainerID] = cancel
			log.Infof("Start monitoring CPU")
			go m.monitorContainerCPU(ctx, event.ContainerID)
		case docker.ContainerActionStop:
			log.Info("Stop monitoring CPU")
			cancel, ok := cancels[event.ContainerID]
			if ok {
				cancel()
				delete(cancels, event.ContainerID)
			}
		default:
			log.WithField("action", event.Action).Info("Unknown container action")
		}
	}
}

func (m *CPUUsageMonitor) monitorHostUsage(ctx context.Context) {
	log := logger.Get(ctx)

	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info("Host CPU Monitoring done")
			return
		case <-tick.C:
			err := m.updateHostCPUUsage(ctx)
			if err != nil {
				log.WithError(err).Error("Fail to update host CPU usage")
			}
		}
	}
}

func (m *CPUUsageMonitor) updateHostCPUUsage(ctx context.Context) error {
	current, err := m.cpuStatReader.Read(ctx)
	if err != nil {
		return errors.Wrap(ctx, err, "get host CPU stats")
	}
	currentCPUUsage := current.All()
	m.cpuUsagesMutex.Lock()
	m.previousHostUsage = m.currentHostUsage
	m.currentHostUsage = &currentCPUUsage
	m.cpuUsagesMutex.Unlock()

	return nil
}

func (m *CPUUsageMonitor) monitorContainerCPU(ctx context.Context, id string) {
	log := logger.Get(ctx)

	tick := time.NewTicker(config.RefreshTime)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			m.cleanMonitoringData(id)
			log.Info("CPU Monitoring stopped - Context done")
			return
		case <-tick.C:
			var cgroupStatsErr cgroup.StatsReaderError
			log.Debug("Refresh CPU Usage")
			err := m.updateContainerCPUUsage(ctx, id)
			if errors.As(err, &cgroupStatsErr) {
				log.WithError(err).Infof("Stop monitoring CPU with error")
				m.cleanMonitoringData(id)
			} else if err != nil {
				// No Error logging to prevent spamming
				log.WithError(err).Info("Fail to update container CPU usage")
			}
		}
	}
}

func (m *CPUUsageMonitor) updateContainerCPUUsage(ctx context.Context, id string) error {
	stats, err := m.cgroupStatsReader.GetStats(ctx, id)
	if err != nil {
		return errors.Wrap(ctx, err, "get cgroup stats")
	}
	systemUsage, err := m.cpuStatReader.Read(ctx)
	if err != nil {
		return errors.Wrap(ctx, err, "read container CPU usage")
	}

	m.cpuUsagesMutex.Lock()
	m.previousContainerStats[id] = m.currentContainerStats[id]
	m.currentContainerStats[id] = stats
	m.previousSystemUsage[id] = m.currentSystemUsage[id]
	m.currentSystemUsage[id] = systemUsage.All().Sum()
	m.cpuUsagesMutex.Unlock()

	return nil
}

func (m *CPUUsageMonitor) cleanMonitoringData(id string) {
	m.cpuUsagesMutex.Lock()
	delete(m.currentContainerStats, id)
	delete(m.previousContainerStats, id)
	m.cpuUsagesMutex.Unlock()
}

func (m CPUUsageMonitor) GetHostUsage() (client.HostCpuUsage, error) {
	m.cpuUsagesMutex.Lock()
	defer m.cpuUsagesMutex.Unlock()
	if m.previousHostUsage == nil || m.currentHostUsage == nil {
		return client.HostCpuUsage{}, nil
	}

	deltaSum := float64(m.currentHostUsage.Sum() - m.previousHostUsage.Sum())
	deltaIdled := float64(m.currentHostUsage.IDLE - m.previousHostUsage.IDLE)

	if deltaIdled < 0 || deltaSum < 0 {
		return client.HostCpuUsage{}, nil
	}

	usage := (deltaSum - deltaIdled) / deltaSum
	return client.HostCpuUsage{
		Usage:                            usage,
		Amount:                           m.numCPU,
		QueueLengthExponentiallySmoothed: 0,
	}, nil
}

func (m CPUUsageMonitor) GetContainerUsage(id string) (Usage, error) {
	m.cpuUsagesMutex.Lock()
	defer m.cpuUsagesMutex.Unlock()

	if _, ok := m.previousContainerStats[id]; !ok {
		return Usage{}, nil
	}

	deltaCPUUsage := float64(m.currentContainerStats[id].CPUUsage - m.previousContainerStats[id].CPUUsage)
	deltaSystemCPUUsage := float64(m.currentSystemUsage[id] - m.previousSystemUsage[id])

	var percents int
	// If both values are positive, the first values are over
	if deltaCPUUsage > 0.0 && deltaSystemCPUUsage > 0.0 {
		percents = int((deltaCPUUsage / deltaSystemCPUUsage) * 100 * float64(m.numCPU))
	}

	return Usage{
		UsageInPercents: percents,
	}, nil
}
