package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"net/http"
	"net/http/pprof"

	"github.com/Scalingo/acadock-monitoring/client"
	"github.com/Scalingo/acadock-monitoring/config"
	"github.com/Scalingo/acadock-monitoring/cpu"
	"github.com/Scalingo/acadock-monitoring/docker"
	"github.com/Scalingo/acadock-monitoring/mem"
	"github.com/Scalingo/acadock-monitoring/net"
	log "github.com/Sirupsen/logrus"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
)

type JSONContentTypeMiddleware struct{}

func (m *JSONContentTypeMiddleware) ServeHTTP(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	res.Header().Set("Content-Type", "application/json")
	next(res, req)
}

func logHandler(next http.HandlerFunc) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		log.WithFields(log.Fields{"method": req.Method, "request_uri": req.RequestURI, "remote_addr": req.RemoteAddr, "status": 404}).Debug()
		next(res, req)
	}
}

func containerUsageHandler(netMonitor *net.NetMonitor) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		id := mux.Vars(req)["id"]
		usage := client.Usage{}

		memUsage, err := mem.GetUsage(id)
		if err != nil {
			res.WriteHeader(500)
			json.NewEncoder(res).Encode(&(map[string]string{"error": err.Error()}))
			return
		}
		usage.Memory = &memUsage.MemoryUsage

		cpuUsage, err := cpu.GetUsage(id)
		if err != nil {
			res.WriteHeader(500)
			json.NewEncoder(res).Encode(&(map[string]string{"error": err.Error()}))
			return
		}
		usage.Cpu = (*client.CpuUsage)(&cpuUsage)

		netUsage, err := netMonitor.GetUsage(id)
		if err != nil {
			res.WriteHeader(500)
			json.NewEncoder(res).Encode(&(map[string]string{"error": err.Error()}))
			return
		}
		usage.Net = (*client.NetUsage)(&netUsage)

		res.WriteHeader(200)
		json.NewEncoder(res).Encode(&usage)
	}
}

func containerMemUsageHandler(res http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]

	containerMemoryUsage, err := mem.GetUsage(id)
	if err != nil {
		res.WriteHeader(500)
		json.NewEncoder(res).Encode(&(map[string]string{"error": err.Error()}))
		return
	}

	res.WriteHeader(200)
	json.NewEncoder(res).Encode(&containerMemoryUsage)
}

func containerCpuUsageHandler(res http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]

	containerCpuUsage, err := cpu.GetUsage(id)
	if err != nil {
		res.WriteHeader(500)
		json.NewEncoder(res).Encode(&(map[string]string{"error": err.Error()}))
		return
	}
	res.WriteHeader(200)
	json.NewEncoder(res).Encode(&containerCpuUsage)
}

func containerNetUsageHandler(netMonitor *net.NetMonitor) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		id := mux.Vars(req)["id"]
		containerNet, err := netMonitor.GetUsage(id)
		if err != nil {
			res.WriteHeader(500)
			json.NewEncoder(res).Encode(&(map[string]string{"error": err.Error()}))
			return
		}

		res.WriteHeader(200)
		json.NewEncoder(res).Encode(&containerNet)
	}
}

func containersUsageHandler(netMonitor *net.NetMonitor) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		usage := client.NewContainersUsage()
		containers, err := docker.ListContainers()
		if err != nil {
			res.WriteHeader(500)
			log.WithError(err).Error("Fail to list containers")
			errors := map[string]string{"message": "fail to list containers", "error": err.Error()}
			json.NewEncoder(res).Encode(&errors)
			return
		}
		for _, container := range containers {
			cpuUsage, err := cpu.GetUsage(container.ID)
			if err != nil {
				log.WithError(err).Errorf("Fail to get CPU usage of '%v'", container.ID)
				continue
			}
			memUsage, err := mem.GetUsage(container.ID)
			if err != nil {
				log.WithError(err).Errorf("Fail to get Memory usage of '%v'", container.ID)
				continue
			}
			netUsage, err := netMonitor.GetUsage(container.ID)
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

	r := mux.NewRouter()

	r.HandleFunc("/containers/{id}/mem", logHandler(containerMemUsageHandler)).Methods("GET")
	r.HandleFunc("/containers/{id}/cpu", logHandler(containerCpuUsageHandler)).Methods("GET")
	r.HandleFunc("/containers/{id}/net", logHandler(containerNetUsageHandler(netMonitor))).Methods("GET")
	r.HandleFunc("/containers/{id}/usage", logHandler(containerUsageHandler(netMonitor))).Methods("GET")
	r.HandleFunc("/containers/usage", logHandler(containersUsageHandler(netMonitor))).Methods("GET")

	if *doProfile {
		log.Info("Enable profiling")
		r.HandleFunc("/debug/pprof", pprof.Index).Methods("GET")
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline).Methods("GET")
		r.HandleFunc("/debug/pprof/profile", pprof.Profile).Methods("GET")
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol).Methods("GET")
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol).Methods("POST")
		r.Handle("/debug/pprof/block", pprof.Handler("block")).Methods("GET")
		r.Handle("/debug/pprof/heap", pprof.Handler("heap")).Methods("GET")
		r.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine")).Methods("GET")
		r.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate")).Methods("GET")
	}

	r.HandleFunc("/{any:.*}", func(res http.ResponseWriter, req *http.Request) {
		log.WithFields(log.Fields{"method": req.Method, "request_uri": req.RequestURI, "remote_addr": req.RemoteAddr, "status": 404}).Info("not found")
		res.WriteHeader(404)
		res.Write([]byte(`{"error": "not found"}`))
	})

	if config.Debug {
		log.SetLevel(log.DebugLevel)
	}

	n := negroni.New(negroni.NewRecovery(), &JSONContentTypeMiddleware{})
	n.UseHandler(r)

	log.Fatal(gracehttp.Serve(&http.Server{
		Addr: ":" + config.ENV["PORT"], Handler: n,
	}))
}
