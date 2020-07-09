package handlers

import (
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type Router struct {
	*mux.Router
	middlewares []Middleware
}

func NewRouter(logger logrus.FieldLogger) *Router {
	r := &Router{}
	r.Router = mux.NewRouter()
	r.Use(MiddlewareFunc(RequestIDMiddleware))
	r.Use(NewLoggingMiddleware(logger))
	return r
}

func New() *Router {
	r := &Router{}
	r.Router = mux.NewRouter()
	r.Use(MiddlewareFunc(RequestIDMiddleware))
	return r
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
