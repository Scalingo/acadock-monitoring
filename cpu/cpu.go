package cpu

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Scalingo/acadock-monitoring/client"
	"github.com/Scalingo/acadock-monitoring/config"
	"github.com/Scalingo/acadock-monitoring/docker"
)

const (
	LXC_CPUACCT_USAGE_FILE = "cpuacct.usage"
)

type Usage client.CpuUsage

var (
	numCPU              = runtime.NumCPU()
	currentSystemUsage  = make(map[string]int64)
	previousSystemUsage = make(map[string]int64)
	previousCpuUsages   = make(map[string]int64)
	cpuUsages           = make(map[string]int64)
	cpuUsagesMutex      = &sync.Mutex{}
)

func cpuacctUsage(container string) (int64, error) {
	file := config.CgroupPath("cpuacct", container) + "/" + LXC_CPUACCT_USAGE_FILE
	f, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	buffer := make([]byte, 16)
	n, err := f.Read(buffer)
	buffer = buffer[:n]

	bufferStr := string(buffer)
	bufferStr = bufferStr[:len(bufferStr)-1]

	res, err := strconv.ParseInt(bufferStr, 10, 64)
	if err != nil {
		log.Println("Failed to parse : ", err)
		return 0, err
	}
	return res, nil
}

func Monitor() {
	containers := docker.RegisterToContainersStream()
	for c := range containers {
		fmt.Println("monitor cpu", c)
		go monitorContainer(c)
	}
}

func monitorContainer(id string) {
	tick := time.NewTicker(time.Duration(config.RefreshTime) * time.Second)
	for {
		<-tick.C
		var err error
		usage, err := cpuacctUsage(id)
		cpuUsagesMutex.Lock()
		if err != nil {
			if _, ok := cpuUsages[id]; ok {
				delete(cpuUsages, id)
			}
			if _, ok := previousCpuUsages[id]; ok {
				delete(previousCpuUsages, id)
			}
			log.Println("stop monitoring", id, "reason:", err)
			cpuUsagesMutex.Unlock()
			return
		}

		previousCpuUsages[id] = cpuUsages[id]
		cpuUsages[id] = usage

		previousSystemUsage[id] = currentSystemUsage[id]
		currentSystemUsage[id], err = systemUsage()
		if err != nil {
			log.Println(err)
		}
		cpuUsagesMutex.Unlock()
	}
}

func GetUsage(id string) (Usage, error) {
	id, err := docker.ExpandId(id)
	if err != nil {
		log.Println("Error when expanding id:", err)
		return Usage{}, err
	}

	cpuUsagesMutex.Lock()
	defer cpuUsagesMutex.Unlock()

	if _, ok := previousCpuUsages[id]; !ok {
		return Usage{}, nil
	}
	deltaCpuUsage := float64(cpuUsages[id] - previousCpuUsages[id])
	deltaSystemCpuUsage := float64(currentSystemUsage[id] - previousSystemUsage[id])

	percents := int(deltaCpuUsage / deltaSystemCpuUsage * 100 * float64(numCPU))
	return Usage{
		UsageInPercents: percents,
	}, nil
}

func systemUsage() (int64, error) {
	f, err := os.OpenFile("/proc/stat", os.O_RDONLY, 0600)
	if err != nil {
		return -1, err
	}

	var line string
	buffer := bufio.NewReader(f)
	for {
		lineBytes, _, err := buffer.ReadLine()
		if err != nil {
			return -1, err
		}
		line = string(lineBytes)
		if strings.HasPrefix(line, "cpu ") {
			break
		}
	}

	err = f.Close()
	if err != nil {
		return -1, err
	}

	fields := strings.Fields(string(line))
	if len(fields) < 8 {
		return -1, errors.New("invalid cpu line in /stat/proc: " + string(line))
	}

	sum := int64(0)
	for i := 1; i < 8; i++ {
		n, err := strconv.ParseInt(fields[i], 10, 64)
		if err != nil {
			return -1, err
		}
		sum += n
	}

	return sum * 1e9 / 100, nil
}
