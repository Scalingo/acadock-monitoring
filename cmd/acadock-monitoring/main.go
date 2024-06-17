package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudflare/tableflip"
	"github.com/gorilla/mux"
	"github.com/urfave/negroni"

	"github.com/Scalingo/acadock-monitoring/config"
	"github.com/Scalingo/acadock-monitoring/cpu"
	"github.com/Scalingo/acadock-monitoring/filters"
	"github.com/Scalingo/acadock-monitoring/mem"
	"github.com/Scalingo/acadock-monitoring/net"
	"github.com/Scalingo/acadock-monitoring/procfs"
	"github.com/Scalingo/acadock-monitoring/webserver"
	"github.com/Scalingo/go-handlers"
	"github.com/Scalingo/go-utils/errors/v2"
	"github.com/Scalingo/go-utils/logger"
)

type JSONContentTypeMiddleware struct{}

func (m *JSONContentTypeMiddleware) ServeHTTP(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	res.Header().Set("Content-Type", "application/json")
	next(res, req)
}

func main() {
	if config.Debug {
		_ = os.Setenv("LOGGER_LEVEL", "debug")
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
		log.Fatalln(err)
	}

	go queueLength.Start(ctx)
	cpuMonitor := cpu.NewCPUUsageMonitor(hostCPU)
	go cpuMonitor.Start(ctx)
	netMonitor := net.NewNetMonitor()
	go netMonitor.Start()
	memMonitor := mem.NewMemoryUsageGetter()

	controller := webserver.NewController(memMonitor, cpuMonitor, netMonitor, queueLength, hostMemory)

	globalRouter := mux.NewRouter()
	r := handlers.NewRouter(log)
	if config.ENV["HTTP_USERNAME"] != "" && config.ENV["HTTP_PASSWORD"] != "" {
		r.Use(handlers.AuthMiddleware(func(user, password string) bool {
			return user == config.ENV["HTTP_USERNAME"] && password == config.ENV["HTTP_PASSWORD"]
		}))
	}
	r.Use(handlers.ErrorMiddleware)

	r.HandleFunc("/containers/{id}/mem", controller.ContainerMemUsageHandler).Methods("GET")
	r.HandleFunc("/containers/{id}/cpu", controller.ContainerCPUUsageHandler).Methods("GET")
	r.HandleFunc("/containers/{id}/net", controller.ContainerNetUsageHandler).Methods("GET")
	r.HandleFunc("/containers/{id}/usage", controller.ContainerUsageHandler).Methods("GET")
	r.HandleFunc("/containers/usage", controller.ContainersUsageHandler).Methods("GET")
	r.HandleFunc("/host/usage", controller.HostResourcesHandler).Methods("GET")

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
		_, _ = res.Write([]byte(`{"error": "not found"}`))
		return nil
	})

	globalRouter.Handle("/{any:.+}", r)

	n := negroni.New(negroni.NewRecovery(), &JSONContentTypeMiddleware{})
	n.UseHandler(globalRouter)

	// Use tableflip to handle graceful restart requests
	upg, err := tableflip.New(tableflip.Options{
		UpgradeTimeout: config.GracefulUpgradeTimeout,
		PIDFile:        config.GracefulPidFile,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Handle SIGHUP and SIGINT signals
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT /*, syscall.SIGTERM*/)
		for s := range sig {
			switch s {
			case syscall.SIGHUP:
				log.Infoln("upgrade requested")
				err := upg.Upgrade()
				if err != nil {
					log.Error("upgrade failed:", err)
					continue
				}
			case syscall.SIGINT:
				upg.Stop()
				log.Infoln("stopping")
				return
			}
		}
	}()

	// Listen must be called before Ready
	ln, err := upg.Listen("tcp", ":"+config.ENV["PORT"])
	if err != nil {
		upg.Stop()
		log.Fatalln("cannot listen:", err)
	}
	log.Info("listening on :" + config.ENV["PORT"])

	server := http.Server{
		Handler: n,
	}

	go func() {
		err := server.Serve(ln)
		if !errors.Is(err, http.ErrServerClosed) {
			log.Println("HTTP server:", err)
		}
	}()

	log.Printf("ready")
	if err := upg.Ready(); err != nil {
		upg.Stop()
		log.Fatalln(err)
	}

	err = upg.WaitForParent(ctx)
	log.Printf("parent exited: %v", err)
	if err != nil {
		upg.Stop()
		log.Fatalln(err)
	}

	defer upg.Stop()
	<-upg.Exit()

	// Make sure to set a deadline on exiting the process
	// after upg.Exit() is closed. No new upgrades can be
	// performed if the parent doesn't exit.
	time.AfterFunc(config.GracefulShutdownTimeout, func() {
		log.Println("Graceful shutdown timed out")
		os.Exit(1)
	})

	// Wait for connections to drain.
	_ = server.Shutdown(context.Background())
}
