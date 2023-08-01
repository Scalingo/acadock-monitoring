package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gofrs/uuid/v5"
)

func RequestIDMiddleware(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			uuid, err := uuid.NewV4()
			if err != nil {
				return fmt.Errorf("fail to generate UUID for X-Request-ID: %v", err)
			}
			id = uuid.String()
			r.Header.Set("X-Request-ID", id)
		}
		r = r.WithContext(context.WithValue(r.Context(), "request_id", id))
		return next(w, r, vars)
	}
}
