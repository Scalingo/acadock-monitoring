package netstat

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type NetworkStats []NetworkStat

type networkInfo struct {
	Bytes      uint64 `json:"bytes"`
	Packets    uint64 `json:"packets"`
	Drop       uint64 `json:"drop"`
	Errs       uint64 `json:"errs"`
	Fifo       uint64 `json:"fifo"`
	Frame      uint64 `json:"frame"`
	Compressed uint64 `json:"compressed"`
	Multicast  uint64 `json:"multicast"`
}

type NetworkStat struct {
	Interface string      `json:"interface"`
	Received  networkInfo `json:"received"`
	Transmit  networkInfo `json:"transmit"`
}

func Stats() (NetworkStats, error) {
	netStatsFile, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	defer netStatsFile.Close()

	var stats NetworkStats
	reader := bufio.NewReader(netStatsFile)

	// Pass the header
	// Inter-|   Receive                                                |  Transmit
	//  face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
	reader.ReadString('\n')
	reader.ReadString('\n')

	var line string
	for err == nil {
		line, err = reader.ReadString('\n')
		if line == "" {
			continue
		}
		stats = append(stats, buildNetworkStat(line))
	}
	return stats, nil
}

func buildNetworkStat(line string) NetworkStat {
	fields := strings.Fields(line)
	interfaceName := strings.TrimSuffix(fields[0], ":")
	return NetworkStat{
		Interface: interfaceName,
		Received:  toNetworkInfo(fields[1:9]),
		Transmit:  toNetworkInfo(fields[9:17]),
	}
}

func toNetworkInfo(fields []string) networkInfo {
	return networkInfo{
		Bytes:      toInt(fields[0]),
		Packets:    toInt(fields[1]),
		Errs:       toInt(fields[2]),
		Drop:       toInt(fields[3]),
		Fifo:       toInt(fields[4]),
		Frame:      toInt(fields[5]),
		Compressed: toInt(fields[6]),
		Multicast:  toInt(fields[7]),
	}
}

func toInt(str string) uint64 {
	res, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		panic(err)
	}
	return res
}
