package handlers

import (
	"os"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

	"github.com/Scalingo/go-utils/logger"
)

type Router struct {
	*mux.Router
	middlewares []Middleware
	// otelOptions are the options used for OpenTelemetry instrumentation
	otelOptions []otelmux.Option
	// Describe the name of the (virtual) server handling
	otelServiceName string
	// otelEnabled indicates if OpenTelemetry instrumentation is enabled (true by default)
	otelEnabled bool
}

const (
	otelDefaultServiceName = "http"
)

type RouterOption func(r *Router)

func WithOtelOptions(opts ...otelmux.Option) RouterOption {
	return func(r *Router) {
		r.otelOptions = opts
	}
}

func WithOtelServiceName(name string) RouterOption {
	return func(r *Router) {
		r.otelServiceName = name
	}
}

func WithoutOtelInstrumentation() RouterOption {
	return func(r *Router) {
		r.otelEnabled = false
	}
}

// NewRouter initializes a router. In containers 3 middleware by default, error
// catching, logging and OpenTelemetry instrumentation
func NewRouter(logger logrus.FieldLogger, options ...RouterOption) *Router {
	otelServiceName := otelDefaultServiceName
	if os.Getenv("OTEL_SERVICE_NAME") != "" {
		otelServiceName = os.Getenv("OTEL_SERVICE_NAME")
	}

	r := &Router{
		Router:          mux.NewRouter(),
		otelServiceName: otelServiceName,
		otelEnabled:     true,
		middlewares: []Middleware{
			NewLoggingMiddleware(logger),
			MiddlewareFunc(RequestIDMiddleware),
		},
	}
	for _, opt := range options {
		opt(r)
	}
	if r.otelEnabled {
		r.Router.Use(otelmux.Middleware(r.otelServiceName, r.otelOptions...))
	}
	return r
}

// Deprecated: Use NewRouter() instead
func New(options ...RouterOption) *Router {
	return NewRouter(logger.Default(), options...)
}

func (r *Router) HandleFunc(pattern string, f HandlerFunc) *mux.Route {
	for _, m := range r.middlewares {
		f = m.Apply(f)
	}

	stdHandler := ToHTTPHandler(f)
	return r.Router.Handle(pattern, stdHandler)
}

func (r *Router) Handle(pattern string, h Handler) *mux.Route {
	return r.HandleFunc(pattern, h.ServeHTTP)
}

func (r *Router) Use(m Middleware) {
	// Add at the beginning of the middleware stack
	// The last middleware is called first
	middlewares := r.middlewares
	r.middlewares = append([]Middleware(nil), m)
	r.middlewares = append(r.middlewares, middlewares...)
}
