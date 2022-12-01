package webserver

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"

	"github.com/Scalingo/acadock-monitoring/client"
	"github.com/Scalingo/acadock-monitoring/docker"
	"github.com/Scalingo/go-utils/logger"
)

func (c Controller) ContainerUsageHandler(res http.ResponseWriter, req *http.Request, params map[string]string) error {
	log := logger.Get(req.Context())
	id := params["id"]
	usage := client.Usage{}

	memUsage, err := c.mem.GetMemoryUsage(id)
	if err != nil {
		return errors.Wrapf(err, "fail to get container memory usage")
	}
	usage.Memory = &memUsage.MemoryUsage

	cpuUsage, err := c.cpu.GetContainerUsage(id)
	if err != nil {
		return errors.Wrapf(err, "fail to get container cpu usage")
	}
	usage.Cpu = (*client.CpuUsage)(&cpuUsage)

	netUsage, err := c.net.GetUsage(id)
	if err != nil {
		return errors.Wrapf(err, "fail to get container network usage")
	}
	usage.Net = (*client.NetUsage)(&netUsage)

	res.WriteHeader(200)
	err = json.NewEncoder(res).Encode(&usage)
	if err != nil {
		log.WithError(err).Error("Fail to encode container usage payload")
	}
	return nil
}

func (c Controller) ContainerMemUsageHandler(res http.ResponseWriter, req *http.Request, params map[string]string) error {
	log := logger.Get(req.Context())
	id := params["id"]

	containerMemoryUsage, err := c.mem.GetMemoryUsage(id)
	if err != nil {
		return errors.Wrapf(err, "fail to get container memory usage")
	}

	res.WriteHeader(200)
	err = json.NewEncoder(res).Encode(&containerMemoryUsage)
	if err != nil {
		log.WithError(err).Error("Fail to encode container memory usage payload")
	}
	return nil
}

func (c Controller) ContainerCPUUsageHandler(res http.ResponseWriter, req *http.Request, params map[string]string) error {
	log := logger.Get(req.Context())
	id := params["id"]

	containerCpuUsage, err := c.cpu.GetContainerUsage(id)
	if err != nil {
		return errors.Wrapf(err, "fail to get container cpu usage")
	}

	res.WriteHeader(200)
	err = json.NewEncoder(res).Encode(&containerCpuUsage)
	if err != nil {
		log.WithError(err).Error("Fail to encode container cpu usage payload")
	}
	return nil
}

func (c Controller) ContainerNetUsageHandler(res http.ResponseWriter, req *http.Request, params map[string]string) error {
	log := logger.Get(req.Context())
	id := params["id"]

	containerNet, err := c.net.GetUsage(id)
	if err != nil {
		return errors.Wrapf(err, "fail to get container network usage")
	}

	res.WriteHeader(200)
	err = json.NewEncoder(res).Encode(&containerNet)
	if err != nil {
		log.WithError(err).Error("Fail to encode container network usage payload")
	}
	return nil
}

func (c Controller) ContainersUsageHandler(res http.ResponseWriter, req *http.Request, _ map[string]string) error {
	log := logger.Get(req.Context())

	usage := client.NewContainersUsage()
	containers, err := docker.ListContainers()
	if err != nil {
		log.WithError(err).Error("Fail to list containers")

		res.WriteHeader(500)
		errs := map[string]string{"message": "fail to list containers", "error": err.Error()}
		err := json.NewEncoder(res).Encode(&errs)
		if err != nil {
			log.WithError(err).Error("Fail to encode containers usage error payload")
		}
		return nil
	}

	for _, container := range containers {
		cpuUsage, err := c.cpu.GetContainerUsage(container.ID)
		if err != nil {
			log.WithError(err).Infof("Fail to get CPU usage of '%v'", container.ID)
			continue
		}

		memUsage, err := c.mem.GetMemoryUsage(container.ID)
		if err != nil {
			log.WithError(err).Infof("Fail to get Memory usage of '%v'", container.ID)
			continue
		}

		netUsage, err := c.net.GetUsage(container.ID)
		if err != nil {
			log.WithError(err).Infof("Fail to get Network usage of '%v'", container.ID)
			continue
		}

		usage[container.ID] = client.Usage{
			Cpu:    (*client.CpuUsage)(&cpuUsage),
			Memory: &memUsage.MemoryUsage,
			Net:    (*client.NetUsage)(&netUsage),
			Labels: container.Labels,
		}
	}

	res.WriteHeader(200)
	err = json.NewEncoder(res).Encode(&usage)
	if err != nil {
		log.WithError(err).Error("Fail to encode containers usage payload")
	}
	return nil
}
