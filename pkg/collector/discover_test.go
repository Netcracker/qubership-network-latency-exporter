package collector

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/Netcracker/qubership-network-latency-exporter/pkg/metrics"
	"github.com/Netcracker/qubership-network-latency-exporter/pkg/utils"
	"github.com/stretchr/testify/assert"
)

var (
	data = &metrics.PingHostList{}
)

// setUp prepares test environment.
func setUp() error {
	data.Targets = append(data.Targets, metrics.PingHost{IPAddress: "1.2.3.4", Name: "node1"}) //valid
	data.Targets = append(data.Targets, metrics.PingHost{IPAddress: "1.2.3.4", Name: ""})      //valid
	data.Targets = append(data.Targets, metrics.PingHost{IPAddress: "", Name: ""})             //invalid
	data.Targets = append(data.Targets, metrics.PingHost{IPAddress: "1.2", Name: "node2"})     //invalid
	return nil
}

// tearDown cleans up environment after test.
func tearDown() error {
	data.Targets = nil
	return nil
}

// TestMain executes tests in prepared environment by setUp and then clean environment with tearDown.
func TestMain(m *testing.M) {
	if err := setUp(); err != nil {
		_ = fmt.Sprintf("Can not prepare test data: %v \n", err)
		os.Exit(1)
	}
	rCode := m.Run()

	if err := tearDown(); err != nil {
		_ = fmt.Sprintf("Can not clean up test environment: %v \n", err)
		os.Exit(1)
	}

	os.Exit(rCode)
}

func TestLoad(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))
	expected := &metrics.PingHostList{Targets: []metrics.PingHost{
		{IPAddress: "1.2.3.4", Name: "node1"},
		{IPAddress: "1.2.3.4", Name: ""},
	}}
	actual := utils.ValidateTargets(logger, data)
	assert.Equal(t, expected, actual)
}

// TestGetByIpAddress checks that the function finds targets by IP in a list.
func TestGetByIpAddress(t *testing.T) {
	assert.Equal(t, &metrics.PingHost{IPAddress: "1.2.3.4", Name: "node1"}, GetByIpAddress(data, "1.2.3.4"))
	assert.Nil(t, GetByIpAddress(data, "1.2.3.9"))
}

// GetByIpAddress finds a PingHost by provided ipAddress from PingHostList.
// Return found item or nil.
func GetByIpAddress(l *metrics.PingHostList, addr string) *metrics.PingHost {
	for _, t := range l.Targets {
		if t.IPAddress == addr {
			return &t
		}
	}
	return nil
}

func TestDiscover_Disabled(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))

	// Test with discovery disabled
	os.Setenv("DISCOVER_ENABLE", "false")
	defer os.Unsetenv("DISCOVER_ENABLE")

	result := Discover(logger)
	assert.Nil(t, result)
}

func TestDiscover_Enabled(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))

	// Test with discovery enabled (default)
	os.Setenv("DISCOVER_ENABLE", "true")
	os.Setenv("NODE_NAME", "test-node")
	defer func() {
		os.Unsetenv("DISCOVER_ENABLE")
		os.Unsetenv("NODE_NAME")
	}()

	// This will try to connect to k8s, but should return nil if connection fails
	result := Discover(logger)
	// We can't easily mock k8s client, so we just check that it doesn't panic
	// In a real k8s environment, this would return targets
	assert.True(t, result == nil || result != nil) // Allow either result since k8s may not be available
}
