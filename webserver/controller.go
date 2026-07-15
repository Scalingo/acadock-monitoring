package webserver

import (
	"github.com/Scalingo/acadock-monitoring/v2/cpu"
	"github.com/Scalingo/acadock-monitoring/v2/filters"
	"github.com/Scalingo/acadock-monitoring/v2/net"
	"github.com/Scalingo/acadock-monitoring/v2/procfs"
	"github.com/Scalingo/acadock-monitoring/v2/resources"
)

type Controller struct {
	resources    resources.UsageGetter
	cpu          *cpu.CPUUsageMonitor
	net          *net.NetMonitor
	queue        filters.MetricsReader
	procfsMemory procfs.MemInfoReader
}

func NewController(resourceUsage resources.UsageGetter, cpu *cpu.CPUUsageMonitor, net *net.NetMonitor,
	queue filters.MetricsReader, procfsMemory procfs.MemInfoReader) Controller {
	return Controller{
		resources:    resourceUsage,
		cpu:          cpu,
		net:          net,
		queue:        queue,
		procfsMemory: procfsMemory,
	}
}
