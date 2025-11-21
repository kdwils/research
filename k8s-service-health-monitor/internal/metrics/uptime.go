package metrics

import (
	"sync"
	"time"

	monitoringv1alpha1 "github.com/kdwils/k8s-service-health-monitor/api/v1alpha1"
)

// UptimeTracker tracks uptime statistics for health checks
type UptimeTracker struct {
	mu      sync.RWMutex
	windows map[string]*windowTracker // key: "namespace/name"
}

// windowTracker tracks data for multiple time windows
type windowTracker struct {
	windows map[string]*circularBuffer // key: window duration string (e.g., "1h")
}

// circularBuffer stores check results with timestamps
type circularBuffer struct {
	results []checkRecord
	maxSize int
	window  time.Duration
}

// checkRecord represents a single check result
type checkRecord struct {
	timestamp time.Time
	success   bool
}

// NewUptimeTracker creates a new uptime tracker
func NewUptimeTracker() *UptimeTracker {
	return &UptimeTracker{
		windows: make(map[string]*windowTracker),
	}
}

// RecordCheck records a check result
func (ut *UptimeTracker) RecordCheck(namespace, name string, windows []string, success bool) {
	ut.mu.Lock()
	defer ut.mu.Unlock()

	key := namespace + "/" + name

	// Initialize tracker if needed
	if _, exists := ut.windows[key]; !exists {
		ut.windows[key] = &windowTracker{
			windows: make(map[string]*circularBuffer),
		}
	}

	tracker := ut.windows[key]
	now := time.Now()

	// Record in each window
	for _, windowStr := range windows {
		duration, err := parseDuration(windowStr)
		if err != nil {
			continue
		}

		if _, exists := tracker.windows[windowStr]; !exists {
			tracker.windows[windowStr] = newCircularBuffer(duration)
		}

		buffer := tracker.windows[windowStr]
		buffer.add(checkRecord{
			timestamp: now,
			success:   success,
		})
	}
}

// GetUptimeStats calculates uptime statistics
func (ut *UptimeTracker) GetUptimeStats(namespace, name string, windows []string) []monitoringv1alpha1.UptimeStat {
	ut.mu.RLock()
	defer ut.mu.RUnlock()

	key := namespace + "/" + name
	tracker, exists := ut.windows[key]
	if !exists {
		return []monitoringv1alpha1.UptimeStat{}
	}

	stats := make([]monitoringv1alpha1.UptimeStat, 0, len(windows))
	now := time.Now()

	for _, windowStr := range windows {
		buffer, exists := tracker.windows[windowStr]
		if !exists {
			continue
		}

		total, successful := buffer.countWithin(now, buffer.window)
		percentage := 0.0
		if total > 0 {
			percentage = (float64(successful) / float64(total)) * 100.0
		}

		stats = append(stats, monitoringv1alpha1.UptimeStat{
			Window:           windowStr,
			Percentage:       percentage,
			TotalChecks:      total,
			SuccessfulChecks: successful,
		})
	}

	return stats
}

// RemoveHealthCheck removes tracking data for a health check
func (ut *UptimeTracker) RemoveHealthCheck(namespace, name string) {
	ut.mu.Lock()
	defer ut.mu.Unlock()

	key := namespace + "/" + name
	delete(ut.windows, key)
}

// newCircularBuffer creates a new circular buffer for a time window
func newCircularBuffer(window time.Duration) *circularBuffer {
	// Calculate max size based on window
	// Assuming checks every 30 seconds, with some buffer
	maxChecks := int(window.Seconds() / 30 * 1.5)
	if maxChecks < 10 {
		maxChecks = 10
	}
	if maxChecks > 100000 {
		maxChecks = 100000
	}

	return &circularBuffer{
		results: make([]checkRecord, 0, maxChecks),
		maxSize: maxChecks,
		window:  window,
	}
}

// add adds a check result to the buffer
func (cb *circularBuffer) add(record checkRecord) {
	// Remove old entries
	cutoff := record.timestamp.Add(-cb.window)
	validStart := 0
	for i, r := range cb.results {
		if r.timestamp.After(cutoff) {
			validStart = i
			break
		}
		if i == len(cb.results)-1 {
			// All entries are old
			validStart = len(cb.results)
		}
	}

	if validStart > 0 {
		cb.results = cb.results[validStart:]
	}

	// Add new record
	cb.results = append(cb.results, record)

	// Enforce max size (keep most recent)
	if len(cb.results) > cb.maxSize {
		cb.results = cb.results[len(cb.results)-cb.maxSize:]
	}
}

// countWithin counts checks within the time window
func (cb *circularBuffer) countWithin(now time.Time, window time.Duration) (total int64, successful int64) {
	cutoff := now.Add(-window)

	for _, record := range cb.results {
		if record.timestamp.After(cutoff) {
			total++
			if record.success {
				successful++
			}
		}
	}

	return total, successful
}

// parseDuration parses a duration string with support for days (d)
func parseDuration(s string) (time.Duration, error) {
	// Handle special case for days
	if len(s) > 0 && s[len(s)-1] == 'd' {
		days := s[:len(s)-1]
		duration, err := time.ParseDuration(days + "h")
		if err != nil {
			return 0, err
		}
		return duration * 24, nil
	}

	return time.ParseDuration(s)
}
