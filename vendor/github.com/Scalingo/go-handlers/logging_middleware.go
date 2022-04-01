package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"
)

var (
	loggerFuncMap = map[logrus.Level]func(logrus.FieldLogger, string, ...interface{}){
		logrus.DebugLevel: logrus.FieldLogger.Debugf,
		logrus.InfoLevel:  logrus.FieldLogger.Infof,
		logrus.WarnLevel:  logrus.FieldLogger.Warnf,
		logrus.ErrorLevel: logrus.FieldLogger.Errorf,
		logrus.FatalLevel: logrus.FieldLogger.Fatalf,
		logrus.PanicLevel: logrus.FieldLogger.Panicf,
	}
)

type patternInfo struct {
	re    *regexp.Regexp
	level logrus.Level
}

type LoggingMiddleware struct {
	logger             logrus.FieldLogger
	filtersEnabledInit sync.Once
	filtersEnabled     bool
	filters            []patternInfo
}

func NewLoggingMiddleware(logger logrus.FieldLogger) Middleware {
	m := &LoggingMiddleware{logger: logger, filters: []patternInfo{}}
	return m
}

func NewLoggingMiddlewareWithFilters(logger logrus.FieldLogger, filters map[string]logrus.Level) (*LoggingMiddleware, error) {
	refilters := []patternInfo{}
	for pattern, level := range filters {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regexp '%v': %v", pattern, err)
		}
		refilters = append(refilters, patternInfo{re: re, level: level})
	}
	m := &LoggingMiddleware{logger: logger, filters: refilters}
	return m, nil
}

func (l *LoggingMiddleware) Apply(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		logger := l.logger
		before := time.Now()

		id, ok := r.Context().Value("request_id").(string)
		if ok {
			logger = logger.WithField("request_id", id)
		}

		from := r.RemoteAddr
		if r.Header.Get("X-Forwarded-For") != "" {
			from = r.Header.Get("X-Forwarded-For")
		}
		proto := r.Proto
		if r.Header.Get("X-Forwarded-Proto") != "" {
			proto = r.Header.Get("X-Forwarded-Proto")
		}

		r = r.WithContext(context.WithValue(r.Context(), "logger", logger))

		fields := logrus.Fields{
			"method":     r.Method,
			"path":       r.URL.String(),
			"host":       r.Host,
			"from":       from,
			"protocol":   proto,
			"referer":    r.Referer(),
			"user_agent": r.UserAgent(),
		}
		for k, v := range fields {
			if v.(string) == "" {
				delete(fields, k)
			}
		}
		logger = logger.WithFields(fields)

		loglevel := logrus.InfoLevel

		// Filters are not enabled by default as we think logs should never be
		// filtered in production. Set HANDLERS_LOG_FILTERS=true environment
		// variable to enforce the feature
		if l.isFiltersEnabled() {
			for _, info := range l.filters {
				if info.re.MatchString(r.URL.Path) {
					loglevel = info.level
				}
			}
		}
		loggerFuncMap[loglevel](logger, "starting request")

		rw := negroni.NewResponseWriter(w)
		err := next(rw, r, vars)
		after := time.Now()

		status := rw.Status()
		if status == 0 {
			status = 200
		}

		logger = logger.WithFields(logrus.Fields{
			"status":   status,
			"duration": after.Sub(before).Seconds(),
			"bytes":    rw.Size(),
		})
		loggerFuncMap[loglevel](logger, "request completed")

		return err
	}
}

func (l *LoggingMiddleware) isFiltersEnabled() bool {
	l.filtersEnabledInit.Do(func() {
		if os.Getenv("HANDLERS_LOG_FILTERS") == "true" {
			l.filtersEnabled = true
		}
	})
	return l.filtersEnabled
}
