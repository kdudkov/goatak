//nolint:gochecknoglobals
package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
)
