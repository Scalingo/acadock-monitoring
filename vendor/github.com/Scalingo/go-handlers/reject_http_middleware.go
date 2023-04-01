package handlers

import (
	"net/http"

	"github.com/Scalingo/go-utils/logger"
)

var RejectHTTPMiddleware = MiddlewareFunc(func(handler HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		forwardedProto := r.Header.Get("X-Forwarded-Proto")
		if forwardedProto != "https" {
			w.WriteHeader(400)
			log := logger.Get(r.Context())
			log.Info("HTTP request received on HTTPS only endpoint")
			return nil
		}
		return handler(w, r, vars)
	}
})
