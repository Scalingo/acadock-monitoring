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
	"github.com/Scalingo/acadock-monitoring/mem"
	"github.com/Scalingo/acadock-monitoring/net"
	"github.com/codegangsta/martini"
)

func containerUsageHandler(res http.ResponseWriter, req *http.Request, params martini.Params) {
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

	if req.URL.Query().Get("net") == "true" {
		netUsage, err := net.GetUsage(id)
		if err != nil {
			res.WriteHeader(500)
			res.Header().Set("Content-Type", "text/plain")
			res.Write([]byte(err.Error()))
			return
		}
		usage.Net = (*client.NetUsage)(&netUsage)
	}

	res.WriteHeader(200)
	json.NewEncoder(res).Encode(&usage)
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

func containerNetUsageHandler(params martini.Params, res http.ResponseWriter) {
	id := params["id"]

	containerNet, err := net.GetUsage(id)
	if err != nil {
		res.WriteHeader(500)
		res.Header().Set("Content-Type", "text/plain")
		res.Write([]byte(err.Error()))
		return
	}

	res.WriteHeader(200)
	json.NewEncoder(res).Encode(&containerNet)
}

func main() {
	doProfile := flag.Bool("profile", false, "profile app")
	flag.Parse()
	go cpu.Monitor()
	go net.Monitor("eth0")

	r := martini.Classic()

	r.Get("/containers/:id/mem", containerMemUsageHandler)
	r.Get("/containers/:id/cpu", containerCpuUsageHandler)
	r.Get("/containers/:id/net", containerNetUsageHandler)
	r.Get("/containers/:id/usage", containerUsageHandler)

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
