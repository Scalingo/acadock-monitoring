package handlers

import (
	"net/http"
)

var ContentTypeJSONMiddleware = MiddlewareFunc(func(handler HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		w.Header().Set("Content-Type", "application/json")
		return handler(w, r, vars)
	}
})
