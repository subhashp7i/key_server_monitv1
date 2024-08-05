package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp" // Import promhttp for metrics testing
)

// TestHandleKeyRequest tests the key generation service
func TestHandleKeyRequest(t *testing.T) {
	// Define test cases with name, requested key length, expected HTTP status code, and whether a key is expected
	tests := []struct {
		name         string
		keyLength    string
		expectedCode int
		expectKey    bool
	}{
		{"Valid key length", "16", http.StatusOK, true},
		{"Zero key length", "0", http.StatusBadRequest, false},
		{"Negative key length", "-1", http.StatusBadRequest, false},
		{"Non-numeric key length", "abc", http.StatusBadRequest, false},
		{"Maximum valid key length", strconv.Itoa(*maxSize), http.StatusOK, true},
		{"Exceeds maximum key length", strconv.Itoa(*maxSize + 1), http.StatusBadRequest, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Log the start of each test
			t.Logf("Running test: %s", tt.name)

			// Create a new HTTP request for each test case
			req, err := http.NewRequest("GET", "/key/"+tt.keyLength, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Create a response recorder to capture the response
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(handleKeyRequest)

			// Serve the HTTP request
			handler.ServeHTTP(rr, req)

			// Check if the response status code matches the expected status code
			if rr.Code != tt.expectedCode {
				t.Errorf("Test %s: expected status code %d, got %d", tt.name, tt.expectedCode, rr.Code)
			} else {
				t.Logf("Test %s: received expected status code %d", tt.name, tt.expectedCode)
			}

			// Check if the response body contains a key if expected
			if tt.expectKey {
				body := rr.Body.String()
				// Parse the keyLength from string to integer
				expectedLength, _ := strconv.Atoi(tt.keyLength)
				if len(body) != 2*expectedLength { // Hex encoding doubles the length
					t.Errorf("Test %s: expected key length %d, got %d", tt.name, 2*expectedLength, len(body))
				} else {
					t.Logf("Test %s: received expected key length %d", tt.name, 2*expectedLength)
				}
			}
		})
	}
}

// TestMetricsEndpoint tests the /metrics endpoint served by Prometheus handler
func TestMetricsEndpoint(t *testing.T) {
	t.Log("Testing /metrics endpoint")

	// Create a new HTTP request for the /metrics endpoint
	req, err := http.NewRequest("GET", "/metrics", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create a response recorder to capture the response
	rr := httptest.NewRecorder()
	handler := promhttp.Handler() // Use promhttp.Handler to handle /metrics endpoint

	// Serve the HTTP request
	handler.ServeHTTP(rr, req)

	// Check if the response status code is 200 OK
	if rr.Code != http.StatusOK {
		t.Errorf("Metrics handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	} else {
		t.Log("Metrics endpoint test passed")
	}
}