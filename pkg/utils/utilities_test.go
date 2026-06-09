package utils

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"log/slog"

	"github.com/Netcracker/qubership-network-latency-exporter/pkg/metrics"
	"github.com/stretchr/testify/assert"
)

func TestGetEnvWithDefaultValue(t *testing.T) {
	// Test with existing env var
	_ = os.Setenv("TEST_VAR", "value")
	defer func() { _ = os.Unsetenv("TEST_VAR") }()
	assert.Equal(t, "value", GetEnvWithDefaultValue("TEST_VAR", "default"))

	// Test with empty env var
	_ = os.Setenv("TEST_EMPTY", "")
	defer func() { _ = os.Unsetenv("TEST_EMPTY") }()
	assert.Equal(t, "default", GetEnvWithDefaultValue("TEST_EMPTY", "default"))

	// Test with non-existing env var
	assert.Equal(t, "default", GetEnvWithDefaultValue("NON_EXISTING", "default"))
}

func TestValidateTargets(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))

	// Test with valid and invalid targets
	targets := &metrics.PingHostList{
		Targets: []metrics.PingHost{
			{IPAddress: "192.168.1.1", Name: "valid1"},
			{IPAddress: "10.0.0.1", Name: "valid2"},
			{IPAddress: "", Name: "invalid1"},          // empty IP
			{IPAddress: "invalid", Name: "invalid2"},   // invalid IP
			{IPAddress: "256.1.1.1", Name: "invalid3"}, // invalid IP
		},
	}

	result := ValidateTargets(logger, targets)

	// Should only contain valid targets
	assert.Len(t, result.Targets, 2)
	assert.Equal(t, "192.168.1.1", result.Targets[0].IPAddress)
	assert.Equal(t, "valid1", result.Targets[0].Name)
	assert.Equal(t, "10.0.0.1", result.Targets[1].IPAddress)
	assert.Equal(t, "valid2", result.Targets[1].Name)
}

func TestGetNamespace(t *testing.T) {
	// Test default namespace when file doesn't exist
	// Since the function reads from an absolute path, we can only test the default case
	result := GetNamespace()
	assert.Equal(t, "monitoring", result)
}

func TestAddHSTSHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	hstsHandler := AddHSTSHeader(handler)

	// Test HTTPS request
	httpsReq := httptest.NewRequest("GET", "https://example.com", nil)
	httpsW := httptest.NewRecorder()
	hstsHandler.ServeHTTP(httpsW, httpsReq)

	assert.Equal(t, "max-age=31536000; includeSubDomains; preload", httpsW.Header().Get("Strict-Transport-Security"))

	// Test HTTP request (should not add header)
	httpReq := httptest.NewRequest("GET", "http://example.com", nil)
	httpW := httptest.NewRecorder()
	hstsHandler.ServeHTTP(httpW, httpReq)

	assert.Empty(t, httpW.Header().Get("Strict-Transport-Security"))
}
