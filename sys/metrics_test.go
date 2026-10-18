// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package sys

import (
	"strings"
	"testing"
)

func TestMetricsRecordAndSnapshot(t *testing.T) {
	RecordRequest("web", true, 150, 42, "gpt-4o", nil)
	RecordRequest("asterisk", true, 300, 100, "gpt-4o", nil)
	RecordRequest("web", false, 50, 0, "gpt-4o", nil)

	snapshot := GetMetricsSnapshot()
	if snapshot.TotalRequests < 3 {
		t.Errorf("expected at least 3 total requests, got %d", snapshot.TotalRequests)
	}
	if snapshot.TotalErrors < 1 {
		t.Errorf("expected at least 1 total error, got %d", snapshot.TotalErrors)
	}
	if snapshot.TotalTokens < 142 {
		t.Errorf("expected at least 142 total tokens, got %d", snapshot.TotalTokens)
	}
	if snapshot.ChannelRequests["web"] < 2 {
		t.Errorf("expected at least 2 web requests, got %d", snapshot.ChannelRequests["web"])
	}
	if snapshot.ChannelRequests["asterisk"] < 1 {
		t.Errorf("expected at least 1 asterisk request, got %d", snapshot.ChannelRequests["asterisk"])
	}
}

func TestGeneratePrometheusMetrics(t *testing.T) {
	RecordRequest("meta", true, 200, 50, "gpt-4o-mini", nil)

	output := GeneratePrometheusMetrics()

	expectedSubstrings := []string{
		"pbx_uptime_seconds",
		"pbx_active_users",
		"pbx_requests_total{channel=\"meta\",status=\"success\"}",
		"pbx_tokens_total{channel=\"meta\",model=\"gpt-4o-mini\"}",
		"pbx_request_duration_ms_total{channel=\"meta\"}",
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(output, sub) {
			t.Errorf("Prometheus metrics missing expected line %q:\n%s", sub, output)
		}
	}
}
