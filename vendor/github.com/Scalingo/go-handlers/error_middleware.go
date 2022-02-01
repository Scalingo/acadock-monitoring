package handlers

import (
	"encoding/json"
	stderr "errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

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

	// If the body has already been partially written, do not write anything else
	if w.Size() != 0 {
		return
	}

	if isContentTypeJSON(w.Header().Get("Content-Type")) {
		if isCauseValidationErrors {
			json.NewEncoder(w).Encode(errors.RootCause(err))
		} else {
			json.NewEncoder(w).Encode(&(map[string]string{"error": err.Error()}))
		}
	} else {
		fmt.Fprintln(w, err)
	}
}

// isContentTypeJSON returns true if the given string is a valid JSON value for the HTTP Content-Type header. Various values can be used to state that a payload is a JSON:
// - The RFC 4627 defines the Content-Type "application/json" (https://datatracker.ietf.org/doc/html/rfc4627)
// - The RFC 6839 defines the suffix "+json":
//     The suffix "+json" MAY be used with any media type whose representation follows that established for "application/json"
//     (https://datatracker.ietf.org/doc/html/rfc6839#page-4)
func isContentTypeJSON(contentType string) bool {
	return contentType == "application/json" || strings.HasSuffix(contentType, "+json")
}
