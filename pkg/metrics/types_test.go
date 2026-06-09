package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNetworkLatencyMetric(t *testing.T) {
	metric := NewNetworkLatencyMetric("destination", "192.168.1.1", "tcp", "80", "10")

	assert.NotNil(t, metric)
	assert.Equal(t, "destination", metric.Tags.Dest)
	assert.Equal(t, "192.168.1.1", metric.Tags.DestIp)
	assert.Equal(t, "tcp", metric.Tags.Protocol)
	assert.Equal(t, "80", metric.Tags.Port)
	assert.Equal(t, 10, metric.Fields.TotalSent)
	assert.Equal(t, StatusUnreachable, metric.Fields.Status)
	assert.Equal(t, 0, metric.Fields.TotalReceived)
	assert.Equal(t, 0.0, metric.Fields.RttMean)
	assert.Equal(t, 0.0, metric.Fields.RttMax)
	assert.Equal(t, 0.0, metric.Fields.RttMin)
	assert.Equal(t, 0.0, metric.Fields.RttDeviation)
}
