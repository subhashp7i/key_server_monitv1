// key-server.go
package main

import (
    "crypto/rand"   // Used for generating secure random numbers
    "encoding/hex"  // Used for encoding bytes to hex strings
    "flag"          // Used for command-line argument parsing
    "fmt"           // Used for formatted I/O operations
    "log"           // Used for logging errors and informational messages
    "net/http"      // Used for creating HTTP servers and handling HTTP requests
    "strconv"       // Used for converting strings to integers
    "github.com/prometheus/client_golang/prometheus"      // Prometheus client library for Go
    "github.com/prometheus/client_golang/prometheus/promhttp" // Used to expose Prometheus metrics via HTTP
)

// Command-line arguments for configuring server port and max key size
var (
    srvPort  = flag.Int("srv-port", 1123, "server listening port (default 1123)")
    maxSize  = flag.Int("max-size", 1024, "maximum key size (default 1024)")
)

// Prometheus metrics
var (
    keyLengthHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name:    "key_length_distribution",
        Help:    "Distribution of key lengths requested",
        Buckets: prometheus.LinearBuckets(0, float64(*maxSize)/20, 20),
    })
    httpStatusCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_status_codes",
            Help: "Counter of HTTP status codes",
        },
        []string{"code"},
    )
)

func init() {
    // Register Prometheus metrics
    prometheus.MustRegister(keyLengthHistogram)
    prometheus.MustRegister(httpStatusCounter)
}

func main() {
    // Parse command-line flags
    flag.Parse()

    // Handle key generation requests
    http.HandleFunc("/key/", handleKeyRequest)

    // Expose Prometheus metrics at /metrics endpoint
    http.Handle("/metrics", promhttp.Handler())

    // Start the HTTP server
    log.Printf("Starting server on port %d...\n", *srvPort)
    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *srvPort), nil))
}

// handleKeyRequest handles HTTP requests to generate random keys
func handleKeyRequest(w http.ResponseWriter, r *http.Request) {
    lengthStr := r.URL.Path[len("/key/"):] // Extract key length from the request URL

    length, err := strconv.Atoi(lengthStr) // Convert key length to an integer
    if err != nil || length <= 0 || length > *maxSize {
        // Handle invalid key length request
        http.Error(w, "Invalid key length", http.StatusBadRequest)
        httpStatusCounter.WithLabelValues("400").Inc()
        return
    }

    // Generate random bytes of specified length
    keyBytes := make([]byte, length)
    _, err = rand.Read(keyBytes)
    if err != nil {
        // Handle random byte generation error
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        httpStatusCounter.WithLabelValues("500").Inc()
        return
    }

    // Encode bytes to hex string
    hexKey := hex.EncodeToString(keyBytes)

    // Write response
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(hexKey))

    // Update Prometheus metrics
    httpStatusCounter.WithLabelValues("200").Inc()
    keyLengthHistogram.Observe(float64(length))
}