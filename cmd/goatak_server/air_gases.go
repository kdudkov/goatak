package main

import (
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"time"

	"github.com/aofei/air"
)

var (
	httpRequestsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "goatak",
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "The latency of the HTTP requests.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"api"})

	httpRequestsCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "goatak",
		Subsystem: "http",
		Name:      "request_count",
		Help:      "Number of the HTTP requests.",
	}, []string{"api", "path", "method", "code"})
)

func SslCheckHandlerGas(app *App) air.Gas {
	err := errors.New("unauthorized")
	return func(next air.Handler) air.Handler {
		return func(req *air.Request, res *air.Response) error {
			if h := req.HTTPRequest(); h != nil {
				if h.TLS != nil {
					user, serial := getCertUser(h.TLS)
					if app.users.UserIsValid(user, serial) {
						req.SetValue(usernameKey, user)
						return next(req, res)
					} else {
						app.Logger.Warnf("invalid user %s serial %s", user, serial)
					}
				}
			}
			time.Sleep(3 * time.Second)
			res.Status = http.StatusUnauthorized
			return err
		}
	}
}

func LoggerGas(log *zap.SugaredLogger, apiName string) air.Gas {
	logger := log.Named(apiName)
	return func(next air.Handler) air.Handler {
		return func(req *air.Request, res *air.Response) (err error) {
			startTime := time.Now()
			res.Defer(func() {
				endTime := time.Now()
				user := getUsernameFromReq(req)

				httpRequestsDuration.With(prometheus.Labels{"api": apiName}).Observe(endTime.Sub(startTime).Seconds())

				httpRequestsCount.With(prometheus.Labels{
					"api":    apiName,
					"path":   req.RawPath(),
					"method": req.Method,
					"code":   strconv.Itoa(res.Status)}).Inc()

				logger.With(zap.String("user", user), zap.Int("status", res.Status)).Infof(
					"%s %s, client: %s, time :%s",
					req.Method, req.Path, req.ClientAddress(), endTime.Sub(startTime))
			})

			return next(req, res)
		}
	}
}
