package net

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Scalingo/acadock-monitoring/client"
	"github.com/Scalingo/acadock-monitoring/config"
	"github.com/Scalingo/acadock-monitoring/docker"
	"github.com/Scalingo/acadock-monitoring/runner"
	"github.com/Scalingo/go-netstat"
)

var (
	netUsages         = make(map[string]netstat.NetworkStat)
	previousNetUsages = make(map[string]netstat.NetworkStat)
	netUsagesMutex    = &sync.Mutex{}
	nsMutex           = &sync.Mutex{}
)

func Monitor(iface string) {
	containers := docker.RegisterToContainersStream()
	for c := range containers {
		fmt.Println("monitor", iface, "of", c)
		go monitorContainer(c, iface)
	}
}

func monitorContainer(id string, iface string) {
	pid, err := docker.Pid(id)
	if err != nil {
		log.Println("Fail to get PID of", id, ":", err)
		return
	}

	tick := time.NewTicker(time.Duration(config.RefreshTime) * time.Second)
	for {
		<-tick.C
		stats, err := runner.NetStatsRunner(pid)
		if err != nil {
			if os.IsNotExist(err) {
				stopMonitoring(id)
			} else {
				log.Println("Error fetching stats of", id, ":", err)
			}
			return
		}
		for _, stat := range stats {
			if stat.Interface == iface {
				netUsagesMutex.Lock()
				previousNetUsages[id] = netUsages[id]
				netUsages[id] = stat
				netUsagesMutex.Unlock()
				break
			}
		}
	}
}

type Usage client.NetUsage

func GetUsage(id string) (Usage, error) {
	id, err := docker.ExpandId(id)
	if err != nil {
		return Usage{}, err
	}
	usage := Usage{}
	usage.NetworkStat = netUsages[id]
	usage.RxBps = int64(float64(netUsages[id].Received.Bytes-previousNetUsages[id].Received.Bytes) / float64(config.RefreshTime))
	usage.TxBps = int64(float64(netUsages[id].Transmit.Bytes-previousNetUsages[id].Transmit.Bytes) / float64(config.RefreshTime))
	return usage, nil
}

func stopMonitoring(id string) {
	log.Println("End of network monitoring for", id)
	netUsagesMutex.Lock()
	delete(previousNetUsages, id)
	delete(netUsages, id)
	netUsagesMutex.Unlock()
}
