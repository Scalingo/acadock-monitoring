package webserver

import (
	"github.com/Scalingo/acadock-monitoring/cpu"
	"github.com/Scalingo/acadock-monitoring/filters"
	"github.com/Scalingo/acadock-monitoring/mem"
	"github.com/Scalingo/acadock-monitoring/net"
	"github.com/Scalingo/acadock-monitoring/procfs"
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
