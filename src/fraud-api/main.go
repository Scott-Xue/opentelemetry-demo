// Command fraud-api is a simulated third-party fraud screening vendor.
//
// It exists to be opaque. There is no OpenTelemetry SDK here and there never should be:
// the point is that the caller gets a client span with no corresponding server span, the
// way a real SaaS edge looks. See
// docs/superpowers/specs/2026-10-08-believable-fraud-detection-design.md in the
// telemetry-operations-contract repo.
package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

// behaviour holds the misbehaviour knobs. A vendor that cannot be made slow or broken is
// not a useful fault surface.
type behaviour struct {
	latency     time.Duration
	errorRate   float64
	timeoutRate float64
}

func behaviourFromEnv() behaviour {
	return behaviour{
		latency:     time.Duration(envInt("LATENCY_MS", 0)) * time.Millisecond,
		errorRate:   envFloat("ERROR_RATE", 0),
		timeoutRate: envFloat("TIMEOUT_RATE", 0),
	}
}

func newServer(b behaviour) http.Handler {
	mux := http.NewServeMux()

	// Deliberately exempt from the knobs: if healthz failed during a fault injection,
	// Kubernetes would restart the pod and the injected fault would disappear.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/v1/screen", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if b.timeoutRate > 0 && rand.Float64() < b.timeoutRate {
			// Longer than any sane client timeout: the caller gives up first, which is
			// what a hung vendor actually looks like.
			time.Sleep(60 * time.Second)
		}
		if b.latency > 0 {
			time.Sleep(b.latency)
		}
		if b.errorRate > 0 && rand.Float64() < b.errorRate {
			http.Error(w, "upstream scoring unavailable", http.StatusServiceUnavailable)
			return
		}

		var req ScreenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "malformed request body", http.StatusBadRequest)
			return
		}
		if req.OrderID == "" {
			http.Error(w, "order_id is required", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(Screen(req)); err != nil {
			log.Printf("encode response: %v", err)
		}
	})

	return mux
}

func main() {
	port := os.Getenv("FRAUD_API_PORT")
	if port == "" {
		port = "8080"
	}
	b := behaviourFromEnv()
	log.Printf("fraud-api listening on :%s (latency=%s error_rate=%.2f timeout_rate=%.2f)",
		port, b.latency, b.errorRate, b.timeoutRate)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           newServer(b),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
		log.Printf("ignoring unparseable %s=%q", key, v)
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
		log.Printf("ignoring unparseable %s=%q", key, v)
	}
	return def
}
