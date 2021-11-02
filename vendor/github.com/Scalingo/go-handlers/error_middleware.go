package handlers

import (
	"encoding/json"
	stderr "errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"

	"github.com/Scalingo/go-utils/errors"
	"github.com/Scalingo/go-utils/logger"
)

var ErrorMiddleware MiddlewareFunc = MiddlewareFunc(func(handler HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		log, ok := r.Context().Value("logger").(logrus.FieldLogger)
		if !ok {
			log = logrus.New()
		}

		defer func() {
			if rec := recover(); rec != nil {
				debug.PrintStack()
				err, ok := rec.(error)
				if !ok {
					err = stderr.New(rec.(string))
				}
				log.WithError(err).Error("Recover panic")
				w.WriteHeader(500)
				fmt.Fprintln(w, err)
			}
		}()

		rw := negroni.NewResponseWriter(w)
		err := handler(rw, r, vars)

		if ctxerr, ok := err.(errors.ErrCtx); ok {
			log = logger.Get(ctxerr.Ctx())
		}

		if err != nil {
			log = log.WithField("error", err)
			writeError(log, rw, err)
		}

		return err
	}
})

func writeError(log logrus.FieldLogger, w negroni.ResponseWriter, err error) {
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "text/plain")
	}

	isCauseValidationErrors := errors.IsRootCause(err, &errors.ValidationErrors{})
	if isCauseValidationErrors {
		log.Info("Request validation error")
		w.WriteHeader(422)
	} else if w.Status() == 0 {
		log.Error("Request error")
		// If the status is 0, it means WriteHeader has not been called and we've to
		// write it, otherwise it has been done in the handler with another response
		// code.
		w.WriteHeader(500)
	}

	if w.Header().Get("Content-Type") == "application/json" {
		if isCauseValidationErrors {
			json.NewEncoder(w).Encode(errors.RootCause(err))
		} else {
			json.NewEncoder(w).Encode(&(map[string]string{"error": err.Error()}))
		}
	} else {
		fmt.Fprintln(w, err)
	}
}
