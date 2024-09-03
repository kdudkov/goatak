//nolint:gochecknoglobals
package main

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/savsgio/gotils/strings"
)

var (
	messagesMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "goatak",
		Name:      "cots_processed",
		Help:      "The total number of cots processed",
	}, []string{"scope", "msg_type"})

	dropMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "goatak",
		Name:      "cots_dropped",
		Help:      "The total size of cots processed",
	}, []string{"scope", "reason"})

	connectionsMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "goatak",
		Name:      "connections",
		Help:      "The total number of connections",
	}, []string{"scope"})

	httpRequestsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "goatak",
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "The latency of the HTTP requests.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"api", "method", "code"})
)

func NewMetricHandler(api string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		m := strings.Copy(c.Method())
		chainErr := c.Next()
		t := time.Since(start)

		httpRequestsDuration.With(prometheus.Labels{
			"api":    api,
			"method": m,
			"code":   strconv.Itoa(c.Response().StatusCode()),
		}).Observe(t.Seconds())

		return chainErr
	}
}
