package handlers

import (
	"context"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"

	"github.com/Scalingo/go-utils/errors/v3"
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

	prof := profiling{}

	err := prof.initializeFromEnv(ctx)
	if err != nil {
		return nil, errors.Wrap(ctx, err, "initialize pprof profiling")
	}

	pprofRouter := NewRouter(log)

	if !prof.isActivable() {
		log.Info("Profiling router is not activable")
		return pprofRouter, nil
	}

	log.Info("Add Basic Auth middleware to access profiling routes")
	pprofRouter.Use(ErrorMiddleware)
	pprofRouter.Use(AuthMiddleware(func(user, password string) bool {
		return user == prof.auth.username && password == prof.auth.password
	}))

	log.Info("Enabling pprof endpoints under " + PprofRoutePrefix)

	pprofRouter.HandleFunc(PprofRoutePrefix, redirectToIndex)
	pprofRouter.HandleFunc(PprofRoutePrefix+"/", index)
	pprofRouter.HandleFunc(PprofRoutePrefix+"/profile", profile)
	pprofRouter.HandleFunc(PprofRoutePrefix+"/symbol", symbol)
	pprofRouter.HandleFunc(PprofRoutePrefix+"/cmdline", cmdline)
	pprofRouter.HandleFunc(PprofRoutePrefix+"/trace", trace)
	pprofRouter.HandleFunc(PprofRoutePrefix+"/allocs", allocs)
	pprofRouter.HandleFunc(PprofRoutePrefix+"/heap", heap)
	pprofRouter.HandleFunc(PprofRoutePrefix+"/mutex", mutex)
	pprofRouter.HandleFunc(PprofRoutePrefix+"/goroutine", goroutine)
	pprofRouter.HandleFunc(PprofRoutePrefix+"/threadcreate", threadcreate)
	pprofRouter.HandleFunc(PprofRoutePrefix+"/block", block)

	return pprofRouter, nil
}

func (prof *profiling) initializeFromEnv(ctx context.Context) error {
	pprofEnable := os.Getenv("PPROF_ENABLED")
	if pprofEnable == "" {
		return nil
	}

	var err error
	prof.enable, err = strconv.ParseBool(pprofEnable)
	if err != nil {
		return errors.Wrap(ctx, err, "parse environment variable PPROF_ENABLED")
	}
	prof.auth.username = os.Getenv("PPROF_USERNAME")
	prof.auth.password = os.Getenv("PPROF_PASSWORD")

	return nil
}

func (prof *profiling) isActivable() bool {
	return prof.enable && prof.auth.username != "" && prof.auth.password != ""
}

func redirectToIndex(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	http.Redirect(w, r, PprofRoutePrefix+"/", http.StatusPermanentRedirect)
	return nil
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
