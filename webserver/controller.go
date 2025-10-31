package webserver

import (
	"github.com/Scalingo/acadock-monitoring/v2/cpu"
	"github.com/Scalingo/acadock-monitoring/v2/filters"
	"github.com/Scalingo/acadock-monitoring/v2/mem"
	"github.com/Scalingo/acadock-monitoring/v2/net"
	"github.com/Scalingo/acadock-monitoring/v2/procfs"
)

type Controller struct {
	mem          mem.MemoryUsageGetter
	cpu          *cpu.CPUUsageMonitor
	net          *net.NetMonitor
	queue        filters.MetricsReader
	procfsMemory procfs.MemInfoReader
}

func NewController(mem mem.MemoryUsageGetter, cpu *cpu.CPUUsageMonitor, net *net.NetMonitor,
	queue filters.MetricsReader, procfsMemory procfs.MemInfoReader) Controller {
	return Controller{
		mem:          mem,
		cpu:          cpu,
		net:          net,
		queue:        queue,
		procfsMemory: procfsMemory,
	}
}
