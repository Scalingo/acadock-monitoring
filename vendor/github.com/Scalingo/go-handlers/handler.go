package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
)

type HandlerFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string) error

func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	return f(w, r, vars)
}

type Handler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request, vars map[string]string) error
}

func ToHTTPHandler(h Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		h.ServeHTTP(w, r, vars)
	})
}
