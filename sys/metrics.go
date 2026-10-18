// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package sys

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

var (
	startTime = time.Now()
	metricsMu sync.RWMutex

	// Counters
	requestsTotal   = make(map[string]int64) // key: channel:status
	requestDuration = make(map[string]int64) // key: channel -> total ms
	requestCount    = make(map[string]int64) // key: channel -> count
	tokensTotal     = make(map[string]int64) // key: channel:model -> tokens

	// Alert management
	alertMu           sync.Mutex
	consecutiveErrors int
	lastAlertTime     time.Time
)

// ActiveSessionsProvider allows pbx to report its in-memory active sessions count
var ActiveSessionsProvider func() int

type MetricsSnapshot struct {
	UptimeSeconds   int64            `json:"uptime_seconds"`
	ActiveSessions  int              `json:"active_sessions"`
	TotalRequests   int64            `json:"total_requests"`
	TotalErrors     int64            `json:"total_errors"`
	TotalTokens     int64            `json:"total_tokens"`
	AvgLatencyMs    int64            `json:"avg_latency_ms"`
	ChannelRequests map[string]int64 `json:"channel_requests"`
	ChannelTokens   map[string]int64 `json:"channel_tokens"`
}

// RecordRequest tracks a completed or failed request across any communication channel.
func RecordRequest(channel string, success bool, latencyMs int64, tokens int, model string, err error) {
	if channel == "" {
		channel = "unknown"
	}
	if model == "" {
		model = "default"
	}

	status := "success"
	if !success || err != nil {
		status = "error"
	}

	metricsMu.Lock()
	reqKey := channel + ":" + status
	requestsTotal[reqKey]++

	requestDuration[channel] += latencyMs
	requestCount[channel]++

	if tokens > 0 {
		tokenKey := channel + ":" + model
		tokensTotal[tokenKey] += int64(tokens)
	}
	metricsMu.Unlock()

	// Alert monitoring for consecutive failures
	alertMu.Lock()
	defer alertMu.Unlock()
	if !success || err != nil {
		consecutiveErrors++
		if consecutiveErrors >= 3 && time.Since(lastAlertTime) > 2*time.Minute {
			lastAlertTime = time.Now()
			errMsg := "unknown error"
			if err != nil {
				errMsg = err.Error()
			}
			slog.Error("CRITICAL: repeated provider failures detected", "channel", channel, "consecutive_errors", consecutiveErrors, "error", errMsg)

			if Options.AlertNtfy != "" {
				go func(count int, ch, msg string) {
					title := "PBX AI Alert: Repeated Provider Failures"
					body := fmt.Sprintf("Channel: %s\nConsecutive errors: %d\nLast error: %s", ch, count, msg)
					_ = Ntfy(Options.AlertNtfy, title, body, "")
				}(consecutiveErrors, channel, errMsg)
			}
		}
	} else {
		consecutiveErrors = 0
	}
}

// GetMetricsSnapshot returns a structured summary of in-memory metrics.
func GetMetricsSnapshot() MetricsSnapshot {
	metricsMu.RLock()
	defer metricsMu.RUnlock()

	var totalReq, totalErr, totalTok, totalDur, totalCount int64
	chanReqs := make(map[string]int64)
	chanTokens := make(map[string]int64)

	for k, v := range requestsTotal {
		parts := strings.Split(k, ":")
		ch := parts[0]
		st := parts[1]
		totalReq += v
		chanReqs[ch] += v
		if st == "error" {
			totalErr += v
		}
	}

	for k, v := range tokensTotal {
		parts := strings.Split(k, ":")
		ch := parts[0]
		totalTok += v
		chanTokens[ch] += v
	}

	for ch, d := range requestDuration {
		totalDur += d
		totalCount += requestCount[ch]
	}

	var avgLat int64
	if totalCount > 0 {
		avgLat = totalDur / totalCount
	}

	active := 0
	if ActiveSessionsProvider != nil {
		active = ActiveSessionsProvider()
	}

	return MetricsSnapshot{
		UptimeSeconds:   int64(time.Since(startTime).Seconds()),
		ActiveSessions:  active,
		TotalRequests:   totalReq,
		TotalErrors:     totalErr,
		TotalTokens:     totalTok,
		AvgLatencyMs:    avgLat,
		ChannelRequests: chanReqs,
		ChannelTokens:   chanTokens,
	}
}

// GeneratePrometheusMetrics renders metrics in Prometheus text exposition format.
func GeneratePrometheusMetrics() string {
	metricsMu.RLock()
	defer metricsMu.RUnlock()

	var sb strings.Builder

	sb.WriteString("# HELP pbx_uptime_seconds Process uptime in seconds\n")
	sb.WriteString("# TYPE pbx_uptime_seconds gauge\n")
	sb.WriteString(fmt.Sprintf("pbx_uptime_seconds %d\n\n", int64(time.Since(startTime).Seconds())))

	active := 0
	if ActiveSessionsProvider != nil {
		active = ActiveSessionsProvider()
	}
	sb.WriteString("# HELP pbx_active_users Number of currently active chat sessions in memory\n")
	sb.WriteString("# TYPE pbx_active_users gauge\n")
	sb.WriteString(fmt.Sprintf("pbx_active_users %d\n\n", active))

	sb.WriteString("# HELP pbx_requests_total Total number of chat and call requests processed\n")
	sb.WriteString("# TYPE pbx_requests_total counter\n")
	for k, v := range requestsTotal {
		parts := strings.Split(k, ":")
		sb.WriteString(fmt.Sprintf("pbx_requests_total{channel=%q,status=%q} %d\n", parts[0], parts[1], v))
	}
	sb.WriteString("\n")

	sb.WriteString("# HELP pbx_tokens_total Total number of LLM tokens consumed\n")
	sb.WriteString("# TYPE pbx_tokens_total counter\n")
	for k, v := range tokensTotal {
		parts := strings.Split(k, ":")
		sb.WriteString(fmt.Sprintf("pbx_tokens_total{channel=%q,model=%q} %d\n", parts[0], parts[1], v))
	}
	sb.WriteString("\n")

	sb.WriteString("# HELP pbx_request_duration_ms_total Total duration of requests in milliseconds\n")
	sb.WriteString("# TYPE pbx_request_duration_ms_total counter\n")
	for ch, v := range requestDuration {
		sb.WriteString(fmt.Sprintf("pbx_request_duration_ms_total{channel=%q} %d\n", ch, v))
	}
	sb.WriteString("\n")

	sb.WriteString("# HELP pbx_request_duration_ms_count Count of measured requests\n")
	sb.WriteString("# TYPE pbx_request_duration_ms_count counter\n")
	for ch, v := range requestCount {
		sb.WriteString(fmt.Sprintf("pbx_request_duration_ms_count{channel=%q} %d\n", ch, v))
	}
	sb.WriteString("\n")

	alertMu.Lock()
	errs := consecutiveErrors
	alertMu.Unlock()
	sb.WriteString("# HELP pbx_consecutive_errors Current consecutive provider errors\n")
	sb.WriteString("# TYPE pbx_consecutive_errors gauge\n")
	sb.WriteString(fmt.Sprintf("pbx_consecutive_errors %d\n", errs))

	return sb.String()
}
