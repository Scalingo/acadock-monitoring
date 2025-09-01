package handlers

import (
	"encoding/json"
	stderr "errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni/v3"

	v2errors "github.com/Scalingo/go-utils/errors/v2"
	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/go-utils/security"
)

var ErrorMiddleware = MiddlewareFunc(func(handler HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		ctx := r.Context()
		log, ok := ctx.Value("logger").(logrus.FieldLogger)
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

		ctx = errors.RootCtxOrFallback(ctx, err)
		log = logger.Get(ctx)

		if err != nil {
			log = log.WithError(err)
			writeError(log, r, rw, err)
		}

		return err
	}
})

func writeError(log logrus.FieldLogger, req *http.Request, w negroni.ResponseWriter, err error) {
	var validationErrors *errors.ValidationErrors
	var v2validationErrors *v2errors.ValidationErrors
	var badRequestError *BadRequestError

	if w.Header().Get("Content-Type") == "" {
		if req != nil && req.Header != nil && isAcceptingJSON(req.Header.Get("Accept")) {
			w.Header().Set("Content-Type", "application/json")
		} else {
			w.Header().Set("Content-Type", "text/plain")
		}
	}

	isCauseValidationErrors := errors.As(err, &validationErrors)
	// Keep backward compatibility
	if errors.As(err, &v2validationErrors) {
		isCauseValidationErrors = true
	}

	isCauseBadRequestError := errors.As(err, &badRequestError)
	if isCauseValidationErrors {
		w.WriteHeader(422)
	} else if isCauseBadRequestError {
		w.WriteHeader(400)
	} else if w.Status() == 0 {
		// If the status is 0, it means WriteHeader has not been called and we've to
		// write it. Otherwise it has been done in the handler with another response
		// code.
		// In this case, we want to return a 401 error if it's an invalid token error and 500 in other cases.
		if isInvalidTokenError(err) {
			w.WriteHeader(401)
		} else {
			w.WriteHeader(500)
		}
	}

	// We log at error level for all 5xx errors as it means there has been an internal service error. With this logging level, we send a Rollbar error.
	// In all other cases, we log at info level. The status code is most probably a 4xx (i.e. due to a user issue). We don't want a Rollbar error in this case but still want to be informed in the logs.
	if w.Status()/100 == 5 {
		log.Error("Request error")
	} else if isCauseValidationErrors {
		log.Info("Request validation error")
	} else {
		log.Info("Request error")
	}

	// If the body has already been partially written, do not write anything else
	if w.Size() != 0 {
		return
	}

	if !isContentTypeJSON(w.Header().Get("Content-Type")) {
		fmt.Fprintln(w, err)
		return
	}

	if isCauseValidationErrors {
		if v2validationErrors != nil {
			json.NewEncoder(w).Encode(v2validationErrors)
			return
		}

		json.NewEncoder(w).Encode(validationErrors)
		return
	}

	json.NewEncoder(w).Encode(&(map[string]string{"error": err.Error()}))
}

// isContentTypeJSON returns true if the given string is a valid JSON value for the HTTP Content-Type header. Various values can be used to state that a payload is a JSON:
//   - The RFC 4627 defines the Content-Type "application/json" (https://datatracker.ietf.org/doc/html/rfc4627)
//   - The RFC 6839 defines the suffix "+json":
//     The suffix "+json" MAY be used with any media type whose representation follows that established for "application/json"
//     (https://datatracker.ietf.org/doc/html/rfc6839#page-4)
func isContentTypeJSON(contentType string) bool {
	return contentType == "application/json" || strings.HasSuffix(contentType, "+json")
}

// This is a crude way of knowing if the client is accepting JSON as the answer.
// Content Negotiation algorithm doesn't exist in the golang standard lib (yet: https://github.com/golang/go/issues/19307)
func isAcceptingJSON(accept string) bool {
	// If the user did not specify an accept header, we assume they accept everything.
	// However the old versions of this lib assumed that in case of no headers we'll fallback to plaintext.
	// Hence to keep this backward compatibility if no headers are present, we return false.
	if accept == "" {
		return false
	}

	accepts := strings.Split(accept, ",")
	for _, accept := range accepts {
		accept = strings.TrimSpace(accept)
		accept = strings.Split(accept, ";")[0] // Remove any parameters

		if accept == "*" ||
			accept == "*/*" ||
			accept == "application/*" ||
			accept == "application/json" ||
			(strings.HasPrefix(accept, "application/") && strings.HasSuffix(accept, "+json")) {
			return true
		}
	}
	return false
}

func isInvalidTokenError(err error) bool {
	return errors.Is(err, security.ErrFutureTimestamp) ||
		errors.Is(err, security.ErrInvalidTimestamp) ||
		errors.Is(err, security.ErrTokenExpired)
}
