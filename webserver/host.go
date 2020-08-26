package webserver

import (
	"encoding/json"
	"net/http"

	"github.com/Scalingo/acadock-monitoring/filters"

	"github.com/Scalingo/acadock-monitoring/docker"

	"github.com/Scalingo/acadock-monitoring/client"

	"github.com/pkg/errors"
)

func (c Controller) HostResources(res http.ResponseWriter, req *http.Request, params map[string]string) error {
	ctx := req.Context()
	cpu, err := c.cpu.GetHostUsage()
	if err != nil {
		return errors.Wrap(err, "fail to get host cpu usage")
	}
	queueLength, err := c.queue.Read(ctx)
	if err != nil && err != filters.ErrNotEnoughMetrics {
		return errors.Wrap(err, "fail to get current queue length")
	}
	cpu.QueueLengthExponentiallySmoothed = queueLength
	hostMemory, err := c.procfsMemory.Read(ctx)
	if err != nil {
		return errors.Wrap(err, "fail to get host memory usage")
	}

	containers, err := docker.ListContainers()
	if err != nil {
		return errors.Wrap(err, "fail to list docker containers")
	}
	labelFilter := req.URL.Query().Get("include_container_if_label")

	memory := client.HostMemoryUsage{
		Free:  hostMemory.FreeBuffers() / 1024 / 1024,
		Total: hostMemory.MemTotal / 1024 / 1024,
		Swap:  hostMemory.SwapUsed() / 1024 / 1024,
	}

	for _, container := range containers {
		if labelFilter != "" {
			_, ok := container.Labels[labelFilter]
			if !ok {
				continue
			}
		}

		usage, err := c.mem.GetMemoryUsage(container.ID)
		if err != nil {
			return errors.Wrapf(err, "fail to get memory for %s", container.ID)
		}

		memory.MemoryUsage += uint64(usage.MemoryUsage.MemoryUsage)
		memory.MemoryCommitted += uint64(usage.MemoryLimit)
		memory.MaxMemoryUsage += uint64(usage.MaxMemoryUsage)
		memory.SwapCommitted += uint64(usage.SwapLimit)
		memory.SwapUsage += uint64(usage.SwapUsage)
		memory.MaxSwapUsage += uint64(usage.MaxSwapUsage)
	}

	result := client.HostUsage{
		CPU:    cpu,
		Memory: memory,
	}

	json.NewEncoder(res).Encode(&result)
	return nil
}
