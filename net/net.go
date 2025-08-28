package net

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/Scalingo/acadock-monitoring/client"
	"github.com/Scalingo/acadock-monitoring/config"
	"github.com/Scalingo/acadock-monitoring/docker"
	"github.com/Scalingo/go-netstat"
	"github.com/Scalingo/go-utils/logger"
)

type Usage client.NetUsage

type NetMonitor struct {
	containerRepository docker.ContainerRepository
	netUsages           map[string]netstat.NetworkStat
	previousNetUsages   map[string]netstat.NetworkStat
	netUsagesMutex      *sync.Mutex

	containerIfaces      map[string]string
	containerIfacesMutex *sync.Mutex
}

func NewNetMonitor(ctx context.Context, containerRepository docker.ContainerRepository) *NetMonitor {
	monitor := &NetMonitor{
		containerRepository:  containerRepository,
		netUsages:            map[string]netstat.NetworkStat{},
		previousNetUsages:    map[string]netstat.NetworkStat{},
		netUsagesMutex:       &sync.Mutex{},
		containerIfaces:      map[string]string{},
		containerIfacesMutex: &sync.Mutex{},
	}
	go monitor.listeningNewInterfaces(ctx)
	return monitor
}

func (monitor *NetMonitor) Start() {
	tick := time.NewTicker(time.Duration(config.RefreshTime) * time.Second)
	defer tick.Stop()
	for {
		<-tick.C
		stats, err := netstat.Stats()
		if err != nil {
			log.WithError(err).Info("Fail to get network stats")
			continue
		}
		for _, stat := range stats {
			monitor.containerIfacesMutex.Lock()
			containerID := monitor.containerIfaces[stat.Interface]
			monitor.containerIfacesMutex.Unlock()

			if containerID == "" {
				continue
			}

			monitor.netUsagesMutex.Lock()
			monitor.previousNetUsages[containerID] = monitor.netUsages[containerID]
			monitor.netUsages[containerID] = stat
			monitor.netUsagesMutex.Unlock()
		}
	}
	// unreachable code
}

func (monitor *NetMonitor) listeningNewInterfaces(ctx context.Context) {
	containerIDs := monitor.containerRepository.RegisterToContainersStream(ctx)
	for containerID := range containerIDs {
		ctx, _ := logger.WithFieldToCtx(ctx, "container_id", containerID)
		iface, err := getContainerIface(ctx, containerID)
		if err != nil {
			log.WithError(err).Errorf("Fail to get network interface of '%v'", containerID)
			continue
		}
		monitor.containerIfacesMutex.Lock()
		monitor.containerIfaces[iface] = containerID
		monitor.containerIfacesMutex.Unlock()
	}
}

func (monitor *NetMonitor) GetUsage(id string) (Usage, error) {
	netUsages := monitor.netUsages
	previousNetUsages := monitor.previousNetUsages

	usage := Usage{}

	// Actually for containers veth### are inversing Received, Transmit
	// Transmit data are the data uploaded to the container, aka downloads by processes in the container
	// Received is the opposit, what is uploaded by processes in the container

	monitor.netUsagesMutex.Lock()
	usage.NetworkStat = netUsages[id]

	previousRxBps := previousNetUsages[id].Received.Bytes
	previousTxBps := previousNetUsages[id].Transmit.Bytes
	if previousRxBps > 0 {
		usage.RxBps = int64(float64(netUsages[id].Received.Bytes-previousRxBps) / float64(config.RefreshTime))
	}
	if previousTxBps > 0 {
		usage.TxBps = int64(float64(netUsages[id].Transmit.Bytes-previousTxBps) / float64(config.RefreshTime))
	}

	monitor.netUsagesMutex.Unlock()

	return usage, nil
}
