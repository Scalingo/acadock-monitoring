package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"net/http"
	"net/http/pprof"

	"github.com/Scalingo/acadock-monitoring/client"
	"github.com/Scalingo/acadock-monitoring/cpu"
	"github.com/Scalingo/acadock-monitoring/docker"
	"github.com/Scalingo/acadock-monitoring/mem"
	"github.com/Scalingo/acadock-monitoring/net"
	"github.com/codegangsta/martini"
)

func containerUsageHandler(netMonitor *net.NetMonitor) func(res http.ResponseWriter, req *http.Request, params martini.Params) {
	return func(res http.ResponseWriter, req *http.Request, params martini.Params) {
		id := params["id"]
		usage := client.Usage{}

		memUsage, err := mem.GetUsage(id)
		if err != nil {
			res.WriteHeader(500)
			res.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(res, err.Error())
			return
		}
		usage.Memory = &memUsage.MemoryUsage

		cpuUsage, err := cpu.GetUsage(id)
		if err != nil {
			res.WriteHeader(500)
			res.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(res, err.Error())
			return
		}
		usage.Cpu = (*client.CpuUsage)(&cpuUsage)

		netUsage, err := netMonitor.GetUsage(id)
		if err != nil {
			res.WriteHeader(500)
			res.Header().Set("Content-Type", "text/plain")
			res.Write([]byte(err.Error()))
			return
		}
		usage.Net = (*client.NetUsage)(&netUsage)

		res.WriteHeader(200)
		json.NewEncoder(res).Encode(&usage)
	}
}

func containerMemUsageHandler(params martini.Params, res http.ResponseWriter) {
	id := params["id"]

	containerMemoryUsage, err := mem.GetUsage(id)
	if err != nil {
		res.WriteHeader(500)
		res.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(res, err.Error())
		return
	}

	res.WriteHeader(200)
	res.Header().Set("Content-Type", "application/json")

	json.NewEncoder(res).Encode(&containerMemoryUsage)
}

func containerCpuUsageHandler(res http.ResponseWriter, req *http.Request, params martini.Params) {
	id := params["id"]

	containerCpuUsage, err := cpu.GetUsage(id)
	if err != nil {
		res.WriteHeader(500)
		res.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(res, err.Error())
		return
	}
	res.WriteHeader(200)
	json.NewEncoder(res).Encode(&containerCpuUsage)
}

func containerNetUsageHandler(netMonitor *net.NetMonitor) func(params martini.Params, res http.ResponseWriter) {
	return func(params martini.Params, res http.ResponseWriter) {
		id := params["id"]
		containerNet, err := netMonitor.GetUsage(id)
		if err != nil {
			res.WriteHeader(500)
			res.Header().Set("Content-Type", "text/plain")
			res.Write([]byte(err.Error()))
			return
		}

		res.WriteHeader(200)
		json.NewEncoder(res).Encode(&containerNet)
	}
}

func containersUsageHandler(netMonitor *net.NetMonitor) func(res http.ResponseWriter) {
	return func(res http.ResponseWriter) {
		usage := client.NewContainersUsage()
		containers, err := docker.ListContainers()
		if err != nil {
			res.WriteHeader(500)
			log.Println("containers-usage, fail to list containers", err)
			errors := map[string]string{"message": "fail to list containers", "error": err.Error()}
			json.NewEncoder(res).Encode(&errors)
			return
		}
		for _, container := range containers {
			cpuUsage, err := cpu.GetUsage(container.ID)
			if err != nil {
				log.Println("Error getting cpu usage of ", container.ID, ":", err)
				continue
			}
			memUsage, err := mem.GetUsage(container.ID)
			if err != nil {
				log.Println("Error getting mem usage of ", container.ID, ":", err)
				continue
			}
			netUsage, err := netMonitor.GetUsage(container.ID)
			if err != nil {
				log.Println("Error getting net usage of ", container.ID, ":", err)
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
	}
}

func main() {
	doProfile := flag.Bool("profile", false, "profile app")
	nsIfaceID := flag.String("ns-iface-id", "", "<pid>")
	flag.Parse()

	if *nsIfaceID != "" {
		ifaceID, err := net.NsIfaceIDByPID(*nsIfaceID)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Print(ifaceID)
		return
	}

	go cpu.Monitor()

	netMonitor := net.NewNetMonitor()
	go netMonitor.Start()

	r := martini.Classic()

	r.Get("/containers/:id/mem", containerMemUsageHandler)
	r.Get("/containers/:id/cpu", containerCpuUsageHandler)
	r.Get("/containers/:id/net", containerNetUsageHandler(netMonitor))
	r.Get("/containers/:id/usage", containerUsageHandler(netMonitor))
	r.Get("/containers/usage", containersUsageHandler(netMonitor))

	if *doProfile {
		log.Println("Enable profiling")
		r.Get("/debug/pprof", pprof.Index)
		r.Get("/debug/pprof/cmdline", pprof.Cmdline)
		r.Get("/debug/pprof/profile", pprof.Profile)
		r.Get("/debug/pprof/symbol", pprof.Symbol)
		r.Post("/debug/pprof/symbol", pprof.Symbol)
		r.Get("/debug/pprof/block", pprof.Handler("block").ServeHTTP)
		r.Get("/debug/pprof/heap", pprof.Handler("heap").ServeHTTP)
		r.Get("/debug/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
		r.Get("/debug/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
	}
	r.Run()
}
