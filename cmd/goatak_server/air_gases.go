package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/air-gases/authenticator"
	"github.com/aofei/air"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	usernameKey = "username"
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
		Name:      "requests_total",
		Help:      "Number of the HTTP requests.",
	}, []string{"api", "path", "method", "code"})
)

func AuthGas(app *App) air.Gas {
	return authenticator.BasicAuthGas(authenticator.BasicAuthGasConfig{
		Validator: func(username string, password string, req *air.Request, _ *air.Response) (bool, error) {
			req.SetValue(usernameKey, username)
			return app.users.CheckUserAuth(username, password), nil
		},
	})
}

func SSLCheckHandlerGas(app *App) air.Gas {
	err := errors.New("unauthorized")

	return func(next air.Handler) air.Handler {
		return func(req *air.Request, res *air.Response) error {
			if h := req.HTTPRequest(); h != nil {
				if h.TLS != nil {
					user, serial := getCertUser(h.TLS)
					if app.users.UserIsValid(user, serial) {
						req.SetValue(usernameKey, user)

						return next(req, res)
					}

					app.Logger.Warn(fmt.Sprintf("invalid user %s serial %s", user, serial))
				}
			}

			time.Sleep(3 * time.Second)

			res.Status = http.StatusUnauthorized

			return err
		}
	}
}

func LoggerGas(apiName string) air.Gas {
	logger := slog.Default().With("logger", apiName)

	return func(next air.Handler) air.Handler {
		return func(req *air.Request, res *air.Response) error {
			startTime := time.Now()

			res.Defer(func() {
				endTime := time.Now()
				username := getUsernameFromReq(req)

				httpRequestsDuration.With(prometheus.Labels{"api": apiName}).Observe(endTime.Sub(startTime).Seconds())

				httpRequestsCount.With(prometheus.Labels{
					"api":    apiName,
					"path":   req.RawPath(),
					"method": req.Method,
					"code":   strconv.Itoa(res.Status),
				}).Inc()

				logger.With(slog.String("user", username), slog.Int("status", res.Status)).Info(
					fmt.Sprintf("%s %s, client: %s, time :%s",
						req.Method, req.Path, req.ClientAddress(), endTime.Sub(startTime)))
			})

			return next(req, res)
		}
	}
}

func getUsernameFromReq(req *air.Request) string {
	if u := req.Value(usernameKey); u != nil {
		return u.(string)
	}

	return ""
}
