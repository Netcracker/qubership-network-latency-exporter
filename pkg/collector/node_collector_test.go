package collector

import (
	"context"
	"testing"

	"log/slog"

	"github.com/Netcracker/qubership-network-latency-exporter/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestNewNodeCollector(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(nil, &slog.HandlerOptions{}))
	collector, err := newNodeCollector(logger)

	assert.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Equal(t, "node_collector", collector.Name())
	assert.Equal(t, NodeType, collector.Type())
}

func TestNodeCollector_Name(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(nil, &slog.HandlerOptions{}))
	collector, _ := newNodeCollector(logger)

	name := collector.Name()
	assert.Equal(t, "node_collector", name)
}

func TestNodeCollector_Type(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(nil, &slog.HandlerOptions{}))
	collector, _ := newNodeCollector(logger)

	collectorType := collector.Type()
	assert.Equal(t, NodeType, collectorType)
}

func TestNodeCollector_Initialize(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(nil, &slog.HandlerOptions{}))
	collector, _ := newNodeCollector(logger)

	ctx := context.Background()
	config := model.NodeCollector{
		PacketsSent:  "10",
		PacketSize:   "1500",
		ProbeTimeout: "3",
	}

	err := collector.Initialize(ctx, config)
	assert.NoError(t, err)
}

func TestNodeCollector_Close(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(nil, &slog.HandlerOptions{}))
	collector, _ := newNodeCollector(logger)

	// Close should not panic
	assert.NotPanics(t, func() {
		collector.Close()
	})
}
