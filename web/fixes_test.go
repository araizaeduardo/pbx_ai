// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSSEChunksPreserveFormatting(t *testing.T) {
	inputs := []string{
		"",
		"hola",
		"  leading spaces",
		"# Title\n\n- item 1\n- item 2\n\n```go\nfmt.Println(\"x\")\n```\n",
		"tabs\tand\r\nwindows newlines  ",
	}
	for _, in := range inputs {
		got := strings.Join(sseChunkRegex.FindAllString(in, -1), "")
		if got != in {
			t.Errorf("chunks do not reproduce input:\nwant %q\ngot  %q", in, got)
		}
	}
}

func TestRateLimiterBlocked(t *testing.T) {
	rl := newRateLimiter(2, time.Minute)
	if rl.blocked("ip") {
		t.Fatal("unknown key should not be blocked")
	}
	rl.allow("ip")
	rl.allow("ip")
	if !rl.blocked("ip") {
		t.Fatal("key should be blocked after reaching limit")
	}
	if rl.blocked("other") {
		t.Fatal("other key should not be blocked")
	}
}

func TestDashboardAuthRequiresCredentials(t *testing.T) {
	for _, path := range []string{"/api/dashboard/stats", "/metrics"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.RemoteAddr = "192.0.2.1:1234"
		rec := httptest.NewRecorder()
		if checkDashboardAuth(rec, req) {
			t.Fatalf("%s: request without credentials must be rejected", path)
		}
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: want 401, got %d", path, rec.Code)
		}
		if rec.Header().Get("WWW-Authenticate") == "" {
			t.Fatalf("%s: missing WWW-Authenticate header", path)
		}
	}
}
