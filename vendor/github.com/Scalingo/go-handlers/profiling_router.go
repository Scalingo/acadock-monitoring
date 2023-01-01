package handlers

import (
	"context"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"

	"gopkg.in/errgo.v1"

	"github.com/Scalingo/go-utils/logger"
)

const PprofRoutePrefix = "/debug/pprof"

type profiling struct {
	enable bool
	auth   pprofAuthentication
}

type pprofAuthentication struct {
	username string
	password string
}

func NewProfilingRouter(ctx context.Context) (*Router, error) {
	log := logger.Get(ctx)

	prof := new(profiling)

	err := prof.initialize()
	if err != nil {
		return nil, errgo.Notef(err, "fail to initialize pprof profiling")
	}

	if !prof.isActivable() {
		log.Info("Profiling router is not activable")
		return new(Router), nil
	}

	r := NewRouter(log)

	log.Info("Add basic authentication middleware to access profiling routes")
	r.Use(ErrorMiddleware)
	r.Use(AuthMiddleware(func(user, password string) bool {
		return user == prof.auth.username && password == prof.auth.password
	}))

	log.Info("Enabling pprof endpoints under " + PprofRoutePrefix)

	r.HandleFunc(PprofRoutePrefix+"/", index)
	r.HandleFunc(PprofRoutePrefix+"/profile", profile)
	r.HandleFunc(PprofRoutePrefix+"/symbol", symbol)
	r.HandleFunc(PprofRoutePrefix+"/cmdline", cmdline)
	r.HandleFunc(PprofRoutePrefix+"/trace", trace)
	r.HandleFunc(PprofRoutePrefix+"/allocs", allocs)
	r.HandleFunc(PprofRoutePrefix+"/heap", heap)
	r.HandleFunc(PprofRoutePrefix+"/mutex", mutex)
	r.HandleFunc(PprofRoutePrefix+"/goroutine", goroutine)
	r.HandleFunc(PprofRoutePrefix+"/threadcreate", threadcreate)
	r.HandleFunc(PprofRoutePrefix+"/block", block)

	return r, nil
}

func (prof *profiling) initialize() error {
	pprofEnable := os.Getenv("PPROF_ENABLED")
	if pprofEnable == "" {
		return nil
	}

	var err error
	prof.enable, err = strconv.ParseBool(pprofEnable)
	if err != nil {
		return errgo.Notef(err, "fail to parse environment variable PPROF_ENABLED")
	}
	prof.auth.username = os.Getenv("PPROF_USERNAME")
	prof.auth.password = os.Getenv("PPROF_PASSWORD")

	return nil
}

func (prof *profiling) isActivable() bool {
	return prof.enable && prof.auth.username != "" && prof.auth.password != ""
}

func index(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	pprof.Index(w, r)
	return nil
}

func profile(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	pprof.Profile(w, r)
	return nil
}

func symbol(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	pprof.Symbol(w, r)
	return nil
}

func cmdline(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	pprof.Cmdline(w, r)
	return nil
}

func trace(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	pprof.Trace(w, r)
	return nil
}

func allocs(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	h := pprof.Handler("allocs")
	h.ServeHTTP(w, r)
	return nil
}

func heap(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	h := pprof.Handler("heap")
	h.ServeHTTP(w, r)
	return nil
}

func goroutine(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	h := pprof.Handler("goroutine")
	h.ServeHTTP(w, r)
	return nil
}

func mutex(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	h := pprof.Handler("mutex")
	h.ServeHTTP(w, r)
	return nil
}

func block(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	h := pprof.Handler("block")
	h.ServeHTTP(w, r)
	return nil
}

func threadcreate(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	h := pprof.Handler("threadcreate")
	h.ServeHTTP(w, r)
	return nil
}
