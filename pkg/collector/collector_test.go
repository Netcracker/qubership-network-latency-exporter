package collector

import (
	"context"
	"testing"

	"log/slog"

	"github.com/Netcracker/qubership-network-latency-exporter/pkg/metrics"
	"github.com/Netcracker/qubership-network-latency-exporter/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestGetCollectorStates(t *testing.T) {
	states := GetCollectorStates()
	assert.NotNil(t, states)
	assert.IsType(t, map[string]bool{}, states)
}

// Skipping TestGetCollector as it depends on internal init functions that may not run during testing

func TestType_String(t *testing.T) {
	assert.Equal(t, "node_collector", NodeType.String())
	assert.Equal(t, "pod_collector", PodType.String())
	assert.Equal(t, "", Type("invalid").String())
}

func TestAsType(t *testing.T) {
	assert.Equal(t, NodeType, AsType("node_collector"))
	assert.Equal(t, PodType, AsType("pod_collector"))
	assert.Equal(t, Type(""), AsType("invalid"))
	assert.Equal(t, Type("NODE_COLLECTOR"), AsType("NODE_COLLECTOR")) // preserves original case
}

func TestNewMetrics(t *testing.T) {
	metrics := NewMetrics()
	assert.NotNil(t, metrics.TotalScrapes)
	assert.NotNil(t, metrics.ScrapeErrors)
	assert.NotNil(t, metrics.Error)
}

func TestNewConfigContainer(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(nil, &slog.HandlerOptions{}))
	latencyTypes := []string{"node_collector", "pod_collector"}
	container := NewConfigContainer(latencyTypes, "test-namespace", logger)

	assert.NotNil(t, container)
	assert.Equal(t, latencyTypes, container.LatencyTypes)
	assert.Equal(t, "test-namespace", container.Namespace)
	assert.NotNil(t, container.CollectorConfigs)
	assert.Equal(t, logger, container.logger)
}

// Skipping TestContainer_UpdateTargets as it requires complex mocking of exporter and logger

func TestContainer_Initialize(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(nil, &slog.HandlerOptions{}))
	container := NewConfigContainer([]string{"node_collector"}, "test", logger)

	ctx := context.Background()
	targets := metrics.PingHostList{Targets: []metrics.PingHost{{IPAddress: "1.2.3.4", Name: "node1"}}}
	checkTargets := []*metrics.CheckTarget{{Protocol: "ICMP", Port: "1"}}

	// First initialize should work
	err := container.Initialize(ctx, "10", "1500", "3", checkTargets, targets, "/metrics")
	assert.NoError(t, err)

	// Second initialize should not change anything (once.Do)
	err = container.Initialize(ctx, "20", "2000", "5", checkTargets, targets, "/metrics")
	assert.NoError(t, err)

	config := container.GetConfig(ctx, NodeType)
	assert.NotNil(t, config)
	nodeConfig := config.(model.NodeCollector)
	assert.Equal(t, "10", nodeConfig.PacketsSent) // Should still be original value
}

func TestContainer_SetConfig(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(nil, &slog.HandlerOptions{}))
	container := NewConfigContainer([]string{"node_collector", "pod_collector"}, "test", logger)

	ctx := context.Background()
	targets := metrics.PingHostList{Targets: []metrics.PingHost{{IPAddress: "1.2.3.4", Name: "node1"}}}
	checkTargets := []*metrics.CheckTarget{{Protocol: "ICMP", Port: "1"}}

	err := container.SetConfig(ctx, "10", "1500", "3", checkTargets, targets, "/metrics")
	assert.NoError(t, err)

	// Check node config
	nodeConfig := container.GetConfig(ctx, NodeType)
	assert.NotNil(t, nodeConfig)
	nc := nodeConfig.(model.NodeCollector)
	assert.Equal(t, "10", nc.PacketsSent)
	assert.Equal(t, "1500", nc.PacketSize)
	assert.Equal(t, "3", nc.ProbeTimeout)
	assert.Equal(t, checkTargets, nc.CheckTargets)
	assert.Equal(t, targets, nc.Targets)
	assert.Equal(t, "/metrics", nc.MetricsPath)

	// Check pod config
	podConfig := container.GetConfig(ctx, PodType)
	assert.NotNil(t, podConfig)
	pc := podConfig.(model.PodCollector)
	assert.Equal(t, "10", pc.PacketsSent)
}

func TestContainer_GetConfig(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(nil, &slog.HandlerOptions{}))
	container := NewConfigContainer([]string{"node_collector"}, "test", logger)

	ctx := context.Background()
	targets := metrics.PingHostList{Targets: []metrics.PingHost{{IPAddress: "1.2.3.4", Name: "node1"}}}

	err := container.SetConfig(ctx, "10", "1500", "3", nil, targets, "/metrics")
	assert.NoError(t, err)

	// Test getting existing config
	config := container.GetConfig(ctx, NodeType)
	assert.NotNil(t, config)

	// Test getting non-existing config
	config = container.GetConfig(ctx, PodType)
	assert.Nil(t, config)
}

func TestContainer_SetConfig_InvalidType(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(nil, &slog.HandlerOptions{}))
	container := NewConfigContainer([]string{"invalid_type"}, "test", logger)

	ctx := context.Background()
	targets := metrics.PingHostList{}

	err := container.SetConfig(ctx, "10", "1500", "3", nil, targets, "/metrics")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown collector type")
}
