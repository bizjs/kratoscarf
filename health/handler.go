package health

import (
	"encoding/json"
	"net/http"
)

// NewLivenessHandler returns an http.Handler for /healthz.
func NewLivenessHandler(registry *Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		report := registry.CheckLiveness(r.Context())
		writeReport(w, report)
	})
}

// NewReadinessHandler returns an http.Handler for /readyz.
func NewReadinessHandler(registry *Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		report := registry.CheckReadiness(r.Context())
		writeReport(w, report)
	})
}

func writeReport(w http.ResponseWriter, report *HealthReport) {
	w.Header().Set("Content-Type", "application/json")
	if report.Status == StatusDown {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	_ = json.NewEncoder(w).Encode(report)
}
