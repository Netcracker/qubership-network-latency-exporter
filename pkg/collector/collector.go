package collector

import (
	"context"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
)

type contextKey string

const (
	defaultEnabled              = true
	ContextKey       contextKey = "ctxKey"
	commonLabel                 = "dashboard"
	commonLabelValue            = "network-latency-exporter"
)

var (
	factories      = make(map[string]func(logger *slog.Logger) (Collector, error))
	collectorState = make(map[string]bool)
)

// Collector is minimal interface that let you add new prometheus metrics to network_latency_exporter.
type Collector interface {
	// Name of the Scraper. Should be unique.
	Name() string

	Type() Type

	Initialize(ctx context.Context, config interface{}) error

	// Scrape collects data from database connection and sends it over channel as prometheus metric.
	Scrape(ctx context.Context, metrics *Metrics, ch chan<- prometheus.Metric) error

	Close()
}

func GetCollectorStates() map[string]bool {
	return collectorState
}

func GetCollector(name string, logger *slog.Logger) (Collector, error) {
	return factories[name](logger)
}
