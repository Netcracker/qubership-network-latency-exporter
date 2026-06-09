package collector

import (
	"context"
	"testing"

	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestNewExporter(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(nil, &slog.HandlerOptions{}))
	metrics := NewMetrics()
	collectors := []Collector{}

	exporter := New(context.Background(), metrics, collectors, logger)

	assert.NotNil(t, exporter)
	assert.Equal(t, logger, exporter.logger)
	assert.Equal(t, metrics, exporter.metrics)
	assert.Equal(t, collectors, exporter.Collectors)
}

func TestExporter_Describe(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(nil, &slog.HandlerOptions{}))
	metrics := NewMetrics()
	collectors := []Collector{}

	exporter := New(context.Background(), metrics, collectors, logger)

	ch := make(chan *prometheus.Desc, 10)
	go func() {
		exporter.Describe(ch)
		close(ch)
	}()

	// Should receive at least the scrapeDurationDesc
	found := false
	for desc := range ch {
		if desc.String() == scrapeDurationDesc.String() {
			found = true
			break
		}
	}
	assert.True(t, found, "scrapeDurationDesc should be described")
}
