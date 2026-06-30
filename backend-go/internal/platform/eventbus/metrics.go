// Package eventbus provides an in-process publish/subscribe event bus for
// asynchronous communication between agents, business modules, and infrastructure.
package eventbus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// eventsPublished tracks the total number of events published, labeled by
	// topic and delivery status (published, delivered, failed).
	eventsPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "multisell_events_published_total",
			Help: "Total events published.",
		},
		[]string{"topic", "status"},
	)

	// eventsHandlerErrors tracks the total number of handler errors (including
	// panics), labeled by topic.
	eventsHandlerErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "multisell_events_handler_errors_total",
			Help: "Total handler errors.",
		},
		[]string{"topic"},
	)

	// eventsQueueDepth tracks the current depth of the event priority queue.
	eventsQueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "multisell_events_queue_depth",
			Help: "Current event queue depth.",
		},
	)

	// eventsDLQTotal tracks the total number of events moved to the dead-letter
	// queue, labeled by topic.
	eventsDLQTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "multisell_events_dlq_total",
			Help: "Total events moved to dead-letter queue.",
		},
		[]string{"topic"},
	)
)
