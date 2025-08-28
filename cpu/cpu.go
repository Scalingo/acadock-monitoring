package cpu

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/Scalingo/acadock-monitoring/client"
	"github.com/Scalingo/acadock-monitoring/config"
	"github.com/Scalingo/acadock-monitoring/docker"
	"github.com/Scalingo/acadock-monitoring/procfs"
	"github.com/Scalingo/go-utils/logger"
)

const (
	LXC_CPUACCT_USAGE_FILE = "cpuacct.usage"
)

type Usage client.CpuUsage

type CPUUsageMonitor struct {
	numCPU              int
	currentHostUsage    *procfs.SingleCPUStat
	previousHostUsage   *procfs.SingleCPUStat
	currentSystemUsage  map[string]time.Duration
	previousSystemUsage map[string]time.Duration
	previousCPUUsages   map[string]time.Duration
	cpuUsages           map[string]time.Duration
	cpuUsagesMutex      *sync.Mutex
	cpuStatReader       procfs.CPUStat
}

func NewCPUUsageMonitor(cpustat procfs.CPUStat) *CPUUsageMonitor {
	return &CPUUsageMonitor{
		numCPU:              runtime.NumCPU(),
		currentSystemUsage:  make(map[string]time.Duration),
		previousSystemUsage: make(map[string]time.Duration),
		previousCPUUsages:   make(map[string]time.Duration),
		cpuUsages:           make(map[string]time.Duration),
		cpuUsagesMutex:      &sync.Mutex{},
		cpuStatReader:       cpustat,
	}
}

func (m *CPUUsageMonitor) cpuacctUsage(container string) (time.Duration, error) {
	file := config.CgroupPath("cpuacct", container) + "/" + LXC_CPUACCT_USAGE_FILE
	f, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	buffer := make([]byte, 64)
	n, err := f.Read(buffer)
	buffer = buffer[:n]

	bufferStr := string(buffer)
	bufferStr = bufferStr[:len(bufferStr)-1]

	res, err := strconv.ParseInt(bufferStr, 10, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "fail to parse '%v'", file)
	}
	return time.Duration(res) * time.Nanosecond, nil
}

func (m *CPUUsageMonitor) Start(ctx context.Context) {
	log := logger.Get(ctx)

	go m.monitorHostUsage(ctx)

	containers := docker.RegisterToContainersStream()
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
		usage, err := m.cpuacctUsage(id)
		m.cpuUsagesMutex.Lock()
		if err != nil {
			if _, ok := m.cpuUsages[id]; ok {
				delete(m.cpuUsages, id)
			}
			if _, ok := m.previousCPUUsages[id]; ok {
				delete(m.previousCPUUsages, id)
			}
			log.Infof("Stop monitoring CPU of '%v', reason: '%v'", id, err)
			m.cpuUsagesMutex.Unlock()
			return
		}

		m.previousCPUUsages[id] = m.cpuUsages[id]
		m.cpuUsages[id] = usage

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
	id, err := docker.ExpandId(id)
	if err != nil {
		return Usage{}, errors.Wrapf(err, "fail to expand ID '%v'", id)
	}

	m.cpuUsagesMutex.Lock()
	defer m.cpuUsagesMutex.Unlock()

	if _, ok := m.previousCPUUsages[id]; !ok {
		return Usage{}, nil
	}

	deltaCPUUsage := float64(m.cpuUsages[id] - m.previousCPUUsages[id])
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
