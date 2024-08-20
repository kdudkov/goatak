package log

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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

type LoggerConfig struct {
	Name          string
	UserGetter    func(c *fiber.Ctx) string
	DoMetrics     bool
	LogErrorsOnly bool
}

func NewFiberLogger(conf *LoggerConfig) fiber.Handler {
	if conf == nil {
		conf = &LoggerConfig{Name: "http"}
	}

	logger := slog.Default().With(slog.String("logger", conf.Name))

	return func(c *fiber.Ctx) error {
		start := time.Now()
		chainErr := c.Next()
		wt := time.Since(start)

		if conf.DoMetrics {
			metrics(conf.Name, c, wt)
		}

		msg := fmt.Sprintf("%d %s %s %s", c.Response().StatusCode(), c.Method(), c.Path(), c.Request().URI().QueryArgs().String())
		l := logger

		if chainErr != nil {
			l = l.With(slog.Any("error", chainErr))
		}

		status := c.Response().StatusCode()

		var attrs []any
		if conf.UserGetter != nil {
			attrs = []any{
				slog.String("client", c.IP()+":"+c.Port()),
				slog.Int("status", status),
				slog.String("user", conf.UserGetter(c)),
				slog.Int64("ms", wt.Milliseconds()),
			}
		} else {
			attrs = []any{
				slog.String("client", c.IP()+":"+c.Port()),
				slog.Int("status", status),
				slog.Int64("ms", wt.Milliseconds()),
			}
		}

		if conf.LogErrorsOnly {
			switch {
			case status < 300:
				l.Debug(msg, attrs...)
			case status < 400:
				l.Info(msg, attrs...)
			default:
				l.Warn(msg, attrs...)
			}
		} else {
			l.Info(msg, attrs...)
		}

		return chainErr
	}
}

func metrics(api string, ctx *fiber.Ctx, t time.Duration) {
	httpRequestsDuration.With(prometheus.Labels{"api": api}).Observe(t.Seconds())

	httpRequestsCount.With(prometheus.Labels{
		"api":    api,
		"path":   ctx.Path(),
		"method": ctx.Method(),
		"code":   strconv.Itoa(ctx.Response().StatusCode()),
	}).Inc()
}
