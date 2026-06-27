// Package observability provides shared Prometheus registry and OTel tracing setup.
package observability

import "github.com/prometheus/client_golang/prometheus"

// Registry is the application-wide Prometheus registry.
// All subsystems register their metrics here instead of the default global registry
// to avoid conflicts with test pollution and third-party auto-registrations.
var Registry = prometheus.NewRegistry()

// MustRegister registers one or more Collectors, panicking on error.
func MustRegister(cs ...prometheus.Collector) {
	Registry.MustRegister(cs...)
}
