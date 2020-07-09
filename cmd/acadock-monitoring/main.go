package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"net/http"
	"net/http/pprof"

	"github.com/Scalingo/acadock-monitoring/client"
	"github.com/Scalingo/acadock-monitoring/config"
	"github.com/Scalingo/acadock-monitoring/cpu"
	"github.com/Scalingo/acadock-monitoring/docker"
	"github.com/Scalingo/acadock-monitoring/mem"
	"github.com/Scalingo/acadock-monitoring/net"
	"github.com/Scalingo/go-handlers"
	"github.com/Scalingo/go-utils/logger"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
)

type JSONContentTypeMiddleware struct{}

func (m *JSONContentTypeMiddleware) ServeHTTP(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	res.Header().Set("Content-Type", "application/json")
	next(res, req)
}

func (c *NetController) containerUsageHandler(res http.ResponseWriter, req *http.Request, params map[string]string) error {
	id := params["id"]
	usage := client.Usage{}

	memUsage, err := mem.GetUsage(id)
	if err != nil {
		return err
	}
	usage.Memory = &memUsage.MemoryUsage

	cpuUsage, err := cpu.GetUsage(id)
	if err != nil {
		return err
	}
	usage.Cpu = (*client.CpuUsage)(&cpuUsage)

	netUsage, err := c.netMonitor.GetUsage(id)
	if err != nil {
		return err
	}
	usage.Net = (*client.NetUsage)(&netUsage)

	res.WriteHeader(200)
	json.NewEncoder(res).Encode(&usage)
	return nil
}

func containerMemUsageHandler(res http.ResponseWriter, req *http.Request, params map[string]string) error {
	id := mux.Vars(req)["id"]

	containerMemoryUsage, err := mem.GetUsage(id)
	if err != nil {
		return err
	}

	res.WriteHeader(200)
	json.NewEncoder(res).Encode(&containerMemoryUsage)
	return nil
}

func containerCpuUsageHandler(res http.ResponseWriter, req *http.Request, params map[string]string) error {
	id := params["id"]

	containerCpuUsage, err := cpu.GetUsage(id)
	if err != nil {
		return err
	}
	res.WriteHeader(200)
	json.NewEncoder(res).Encode(&containerCpuUsage)
	return nil
}

type NetController struct {
	netMonitor *net.NetMonitor
}

func (c *NetController) containerNetUsageHandler(res http.ResponseWriter, req *http.Request, params map[string]string) error {
	id := params["id"]
	containerNet, err := c.netMonitor.GetUsage(id)
	if err != nil {
		return err
	}

	res.WriteHeader(200)
	json.NewEncoder(res).Encode(&containerNet)
	return nil
}

func (c *NetController) containersUsageHandler(res http.ResponseWriter, req *http.Request, params map[string]string) error {
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
		netUsage, err := c.netMonitor.GetUsage(container.ID)
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

func main() {
	if config.Debug {
		os.Setenv("LOGGER_LEVEL", "debug")
	}

	log := logger.Default()

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

	netController := &NetController{
		netMonitor: netMonitor,
	}

	globalRouter := mux.NewRouter()
	r := handlers.NewRouter(log)
	if config.ENV["HTTP_USERNAME"] != "" && config.ENV["HTTP_PASSWORD"] != "" {
		r.Use(handlers.AuthMiddleware(func(user, password string) bool {
			return user == config.ENV["HTTP_USERNAME"] && password == config.ENV["HTTP_PASSWORD"]
		}))
	}

	r.HandleFunc("/containers/{id}/mem", containerMemUsageHandler).Methods("GET")
	r.HandleFunc("/containers/{id}/cpu", containerCpuUsageHandler).Methods("GET")
	r.HandleFunc("/containers/{id}/net", netController.containerNetUsageHandler).Methods("GET")
	r.HandleFunc("/containers/{id}/usage", netController.containerUsageHandler).Methods("GET")
	r.HandleFunc("/containers/usage", netController.containersUsageHandler).Methods("GET")

	if *doProfile {
		pprofRouter := mux.NewRouter()
		log.Info("Enable profiling")
		pprofRouter.HandleFunc("/debug/pprof", pprof.Index).Methods("GET")
		pprofRouter.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline).Methods("GET")
		pprofRouter.HandleFunc("/debug/pprof/profile", pprof.Profile).Methods("GET")
		pprofRouter.HandleFunc("/debug/pprof/symbol", pprof.Symbol).Methods("GET")
		pprofRouter.HandleFunc("/debug/pprof/symbol", pprof.Symbol).Methods("POST")
		pprofRouter.Handle("/debug/pprof/block", pprof.Handler("block")).Methods("GET")
		pprofRouter.Handle("/debug/pprof/heap", pprof.Handler("heap")).Methods("GET")
		pprofRouter.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine")).Methods("GET")
		pprofRouter.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate")).Methods("GET")

		globalRouter.Handle("/debug/pprof/{prop:.*}", pprofRouter)
	}

	r.HandleFunc("/{any:.*}", func(res http.ResponseWriter, req *http.Request, params map[string]string) error {
		res.WriteHeader(404)
		res.Write([]byte(`{"error": "not found"}`))
		return nil
	})

	globalRouter.Handle("/{any:.+}", r)

	n := negroni.New(negroni.NewRecovery(), &JSONContentTypeMiddleware{})
	n.UseHandler(globalRouter)

	log.Fatal(gracehttp.Serve(&http.Server{
		Addr: ":" + config.ENV["PORT"], Handler: n,
	}))
}
