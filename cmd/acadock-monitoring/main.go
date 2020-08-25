package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/Scalingo/acadock-monitoring/config"
	"github.com/Scalingo/acadock-monitoring/cpu"
	"github.com/Scalingo/acadock-monitoring/filters"
	"github.com/Scalingo/acadock-monitoring/mem"
	"github.com/Scalingo/acadock-monitoring/net"
	"github.com/Scalingo/acadock-monitoring/procfs"
	"github.com/Scalingo/acadock-monitoring/webserver"
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

func main() {
	if config.Debug {
		os.Setenv("LOGGER_LEVEL", "debug")
	}

	log := logger.Default()
	ctx := logger.ToCtx(context.Background(), log)

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

	hostCPU := procfs.NewCPUStatReader()
	hostMemory := procfs.NewMemInfoReader()
	hostLoadAvg := procfs.NewLoadAvgReader()
	queueLength, err := filters.NewExponentialSmoothing(procfs.FilterWrap(hostLoadAvg),
		filters.WithQueueLength(config.QueueLengthElementsNeeded),
		filters.WithAverageConfig(config.QueueLengthPointsPerSample, config.QueueLengthSamplingInterval),
	)
	if err != nil {
		panic(err)
	}

	go queueLength.Start(ctx)
	cpu := cpu.NewCPUUsageMonitor(hostCPU)
	go cpu.Start(ctx)
	net := net.NewNetMonitor()
	go net.Start()
	mem := mem.NewMemoryUsageGetter()

	controller := webserver.NewController(mem, cpu, net, queueLength, hostMemory)

	globalRouter := mux.NewRouter()
	r := handlers.NewRouter(log)

	r.HandleFunc("/containers/{id}/mem", controller.ContainerMemUsageHandler).Methods("GET")
	r.HandleFunc("/containers/{id}/cpu", controller.ContainerCpuUsageHandler).Methods("GET")
	r.HandleFunc("/containers/{id}/net", controller.ContainerNetUsageHandler).Methods("GET")
	r.HandleFunc("/containers/{id}/usage", controller.ContainerUsageHandler).Methods("GET")
	r.HandleFunc("/containers/usage", controller.ContainersUsageHandler).Methods("GET")
	r.HandleFunc("/host/usage", controller.HostResources)

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

	log.Info("Listening on :" + config.ENV["PORT"])
	log.Fatal(gracehttp.Serve(&http.Server{
		Addr: ":" + config.ENV["PORT"], Handler: n,
	}))
}
