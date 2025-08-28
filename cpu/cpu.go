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
	"github.com/Scalingo/go-utils/logger"
)

type Usage client.CpuUsage

type CPUUsageMonitor struct {
	numCPU                 int
	currentHostUsage       *procfs.SingleCPUStat
	previousHostUsage      *procfs.SingleCPUStat
	currentSystemUsage     map[string]time.Duration
	previousSystemUsage    map[string]time.Duration
	currentContainerStats  map[string]cgroup.Stats
	previousContainerStats map[string]cgroup.Stats
	cpuUsagesMutex         *sync.Mutex
	cpuStatReader          procfs.CPUStat
	cgroupStatsReader      *cgroup.StatsReader
}

func NewCPUUsageMonitor(cpustat procfs.CPUStat) *CPUUsageMonitor {
	return &CPUUsageMonitor{
		numCPU:                 runtime.NumCPU(),
		currentSystemUsage:     make(map[string]time.Duration),
		previousSystemUsage:    make(map[string]time.Duration),
		previousContainerStats: make(map[string]cgroup.Stats),
		currentContainerStats:  make(map[string]cgroup.Stats),
		cpuUsagesMutex:         &sync.Mutex{},
		cpuStatReader:          cpustat,
		cgroupStatsReader:      cgroup.NewStatsReader(),
	}
}

func (m *CPUUsageMonitor) Start(ctx context.Context) {
	log := logger.Get(ctx)

	go m.monitorHostUsage(ctx)

	containers := docker.RegisterToContainersStream(ctx)
	for c := range containers {
		log.Infof("Monitoring CPU of %v", c)
		go m.monitorContainer(ctx, c)
	}
}

func (m *CPUUsageMonitor) monitorHostUsage(ctx context.Context) {
	log := logger.Get(ctx)

	tick := time.NewTicker(time.Second)
	for {
		<-tick.C
		current, err := m.cpuStatReader.Read(ctx)
		if err != nil {
			log.WithError(err).Error("fail to get container ")
			continue
		}
		currentCPUUsage := current.All()
		m.cpuUsagesMutex.Lock()
		m.previousHostUsage = m.currentHostUsage
		m.currentHostUsage = &currentCPUUsage
		m.cpuUsagesMutex.Unlock()
	}
}

func (m *CPUUsageMonitor) monitorContainer(ctx context.Context, id string) {
	log := logger.Get(ctx)

	tick := time.NewTicker(time.Duration(config.RefreshTime) * time.Second)
	for {
		<-tick.C
		var err error
		stats, err := m.cgroupStatsReader.GetStats(ctx, id)
		m.cpuUsagesMutex.Lock()
		if err != nil {
			delete(m.currentContainerStats, id)
			delete(m.previousContainerStats, id)
			log.Infof("Stop monitoring CPU of '%v', reason: '%v'", id, err)
			m.cpuUsagesMutex.Unlock()
			return
		}

		m.previousContainerStats[id] = m.currentContainerStats[id]
		m.currentContainerStats[id] = stats

		m.previousSystemUsage[id] = m.currentSystemUsage[id]
		systemUsage, err := m.cpuStatReader.Read(ctx)
		if err != nil {
			log.WithError(err).Warn("fail to read system CPU usage")
		}
		m.currentSystemUsage[id] = systemUsage.All().Sum()
		m.cpuUsagesMutex.Unlock()
	}
}

func (m CPUUsageMonitor) GetHostUsage() (client.HostCpuUsage, error) {
	m.cpuUsagesMutex.Lock()
	defer m.cpuUsagesMutex.Unlock()
	if m.previousContainerStats == nil || m.currentHostUsage == nil {
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
