package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	messagesMetric = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "goatak",
		Name:      "cots_processed",
		Help:      "The total number of cots processed",
	})
)
