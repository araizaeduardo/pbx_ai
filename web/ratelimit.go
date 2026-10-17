// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package web

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateBucket
	limit   int
	window  time.Duration
}

type rateBucket struct {
	count    int
	lastSeen time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		buckets: make(map[string]*rateBucket),
		limit:   limit,
		window:  window,
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok || now.Sub(b.lastSeen) > rl.window {
		rl.buckets[key] = &rateBucket{count: 1, lastSeen: now}
		return true
	}

	if b.count >= rl.limit {
		return false
	}

	b.count++
	return true
}

func (rl *rateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.window * 2)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for k, b := range rl.buckets {
			if now.Sub(b.lastSeen) > rl.window {
				delete(rl.buckets, k)
			}
		}
		rl.mu.Unlock()
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// chatRateLimiter limits /chat to 60 requests per minute per IP.
var chatRateLimiter = newRateLimiter(60, time.Minute)

// webhookRateLimiter limits webhooks to 120 requests per minute per IP.
var webhookRateLimiter = newRateLimiter(120, time.Minute)
