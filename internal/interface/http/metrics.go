package httpiface

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var defaultDurationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5}

type metricKey struct {
	method string
	route  string
	status string
}

type requestStats struct {
	count   uint64
	errors  uint64
	sum     float64
	buckets []uint64
}

type Metrics struct {
	mu                  sync.Mutex
	buckets             []float64
	stats               map[metricKey]*requestStats
	expirySweeps        uint64
	expiredReservations uint64
}

func NewMetrics() *Metrics {
	buckets := append([]float64(nil), defaultDurationBuckets...)
	return &Metrics{
		buckets: buckets,
		stats:   make(map[metricKey]*requestStats),
	}
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		m.record(r.Method, routeLabel(r), recorder.status, time.Since(started))
	})
}

func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = w.Write([]byte(m.render()))
	})
}

func (m *Metrics) RecordExpirySweep(expired int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.expirySweeps++
	if expired > 0 {
		m.expiredReservations += uint64(expired)
	}
}

func (m *Metrics) record(method, route string, status int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := metricKey{method: method, route: route, status: strconv.Itoa(status)}
	stats := m.stats[key]
	if stats == nil {
		stats = &requestStats{buckets: make([]uint64, len(m.buckets))}
		m.stats[key] = stats
	}

	seconds := duration.Seconds()
	stats.count++
	stats.sum += seconds
	if status >= http.StatusInternalServerError {
		stats.errors++
	}
	for i, boundary := range m.buckets {
		if seconds <= boundary {
			stats.buckets[i]++
		}
	}
}

func (m *Metrics) render() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	keys := make([]metricKey, 0, len(m.stats))
	for key := range m.stats {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].route != keys[j].route {
			return keys[i].route < keys[j].route
		}
		if keys[i].method != keys[j].method {
			return keys[i].method < keys[j].method
		}
		return keys[i].status < keys[j].status
	})

	var b strings.Builder
	b.WriteString("# HELP inventory_expiry_sweeps_total Explicit expiry sweeps run by the background worker.\n")
	b.WriteString("# TYPE inventory_expiry_sweeps_total counter\n")
	fmt.Fprintf(&b, "inventory_expiry_sweeps_total %d\n", m.expirySweeps)

	b.WriteString("# HELP inventory_expired_reservations_total Reservations expired by explicit sweeps.\n")
	b.WriteString("# TYPE inventory_expired_reservations_total counter\n")
	fmt.Fprintf(&b, "inventory_expired_reservations_total %d\n", m.expiredReservations)

	b.WriteString("# HELP inventory_http_requests_total Total HTTP requests handled by the inventory service.\n")
	b.WriteString("# TYPE inventory_http_requests_total counter\n")
	for _, key := range keys {
		stats := m.stats[key]
		fmt.Fprintf(&b, "inventory_http_requests_total{%s} %d\n", key.labels(), stats.count)
	}

	b.WriteString("# HELP inventory_http_request_errors_total HTTP requests that returned 5xx responses.\n")
	b.WriteString("# TYPE inventory_http_request_errors_total counter\n")
	for _, key := range keys {
		stats := m.stats[key]
		fmt.Fprintf(&b, "inventory_http_request_errors_total{%s} %d\n", key.labels(), stats.errors)
	}

	b.WriteString("# HELP inventory_http_request_duration_seconds HTTP request duration in seconds.\n")
	b.WriteString("# TYPE inventory_http_request_duration_seconds histogram\n")
	for _, key := range keys {
		stats := m.stats[key]
		for i, boundary := range m.buckets {
			fmt.Fprintf(&b, "inventory_http_request_duration_seconds_bucket{%s,le=%q} %d\n", key.labels(), formatBucket(boundary), stats.buckets[i])
		}
		fmt.Fprintf(&b, "inventory_http_request_duration_seconds_bucket{%s,le=\"+Inf\"} %d\n", key.labels(), stats.count)
		fmt.Fprintf(&b, "inventory_http_request_duration_seconds_sum{%s} %.9f\n", key.labels(), stats.sum)
		fmt.Fprintf(&b, "inventory_http_request_duration_seconds_count{%s} %d\n", key.labels(), stats.count)
	}

	return b.String()
}

func (k metricKey) labels() string {
	return fmt.Sprintf("method=%q,route=%q,status=%q", escapeLabel(k.method), escapeLabel(k.route), escapeLabel(k.status))
}

func routeLabel(r *http.Request) string {
	path := r.URL.Path
	switch {
	case r.Method == http.MethodPost && path == "/reservations":
		return "/reservations"
	case r.Method == http.MethodPost && strings.HasPrefix(path, "/reservations/") && strings.HasSuffix(path, "/confirm"):
		return "/reservations/{id}/confirm"
	case r.Method == http.MethodPost && strings.HasPrefix(path, "/reservations/") && strings.HasSuffix(path, "/cancel"):
		return "/reservations/{id}/cancel"
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/reservations/"):
		return "/reservations/{id}"
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/products/") && strings.HasSuffix(path, "/stock"):
		return "/products/{id}/stock"
	default:
		return "unknown"
	}
}

func formatBucket(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func escapeLabel(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
