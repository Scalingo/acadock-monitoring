package webserver

import (
	"encoding/json"
	"net/http"

	"github.com/Scalingo/acadock-monitoring/docker"

	"github.com/Scalingo/acadock-monitoring/client"
	"github.com/Scalingo/go-utils/logger"
	"github.com/gorilla/mux"
)

func (c Controller) ContainerUsageHandler(res http.ResponseWriter, req *http.Request, params map[string]string) error {
	id := params["id"]
	usage := client.Usage{}

	memUsage, err := c.mem.GetMemoryUsage(id)
	if err != nil {
		return err
	}
	usage.Memory = &memUsage.MemoryUsage

	cpuUsage, err := c.cpu.GetContainerUsage(id)
	if err != nil {
		return err
	}
	usage.Cpu = (*client.CpuUsage)(&cpuUsage)

	netUsage, err := c.net.GetUsage(id)
	if err != nil {
		return err
	}
	usage.Net = (*client.NetUsage)(&netUsage)

	res.WriteHeader(200)
	json.NewEncoder(res).Encode(&usage)
	return nil
}

func (c Controller) ContainerMemUsageHandler(res http.ResponseWriter, req *http.Request, params map[string]string) error {
	id := mux.Vars(req)["id"]

	containerMemoryUsage, err := c.mem.GetMemoryUsage(id)
	if err != nil {
		return err
	}

	res.WriteHeader(200)
	json.NewEncoder(res).Encode(&containerMemoryUsage)
	return nil
}

func (c Controller) ContainerCpuUsageHandler(res http.ResponseWriter, req *http.Request, params map[string]string) error {
	id := params["id"]

	containerCpuUsage, err := c.cpu.GetContainerUsage(id)
	if err != nil {
		return err
	}
	res.WriteHeader(200)
	json.NewEncoder(res).Encode(&containerCpuUsage)
	return nil
}

func (c Controller) ContainerNetUsageHandler(res http.ResponseWriter, req *http.Request, params map[string]string) error {
	id := params["id"]
	containerNet, err := c.net.GetUsage(id)
	if err != nil {
		return err
	}

	res.WriteHeader(200)
	json.NewEncoder(res).Encode(&containerNet)
	return nil
}

func (c Controller) ContainersUsageHandler(res http.ResponseWriter, req *http.Request, params map[string]string) error {
	log := logger.Get(req.Context())
	usage := client.NewContainersUsage()
	containers, err := docker.ListContainers()
	if err != nil {
		res.WriteHeader(500)
		log.WithError(err).Error("Fail to list containers")
		errors := map[string]string{"message": "fail to list containers", "error": err.Error()}
		json.NewEncoder(res).Encode(&errors)
		return nil
	}
	for _, container := range containers {
		cpuUsage, err := c.cpu.GetContainerUsage(container.ID)
		if err != nil {
			log.WithError(err).Errorf("Fail to get CPU usage of '%v'", container.ID)
			continue
		}
		memUsage, err := c.mem.GetMemoryUsage(container.ID)
		if err != nil {
			log.WithError(err).Errorf("Fail to get Memory usage of '%v'", container.ID)
			continue
		}
		netUsage, err := c.net.GetUsage(container.ID)
		if err != nil {
			log.WithError(err).Errorf("Fail to get Network usage of '%v'", container.ID)
			continue
		}
		usage[container.ID] = client.Usage{
			Cpu:    (*client.CpuUsage)(&cpuUsage),
			Memory: &memUsage.MemoryUsage,
			Net:    (*client.NetUsage)(&netUsage),
			Labels: container.Labels,
		}
	}
	json.NewEncoder(res).Encode(&usage)
	return nil
}
