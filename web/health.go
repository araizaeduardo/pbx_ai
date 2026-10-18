// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package web

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/eja/pbx/db"
	"github.com/eja/pbx/pbx"
	"github.com/eja/pbx/sys"
)

type HealthResponse struct {
	Status         string            `json:"status"`
	Version        string            `json:"version"`
	UptimeSeconds  int64             `json:"uptime_seconds"`
	Timestamp      string            `json:"timestamp"`
	Database       string            `json:"database"`
	Channels       map[string]bool   `json:"channels"`
	ActiveSessions int               `json:"active_sessions"`
}

func healthRouter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	dbStatus := "ok"
	statusCode := http.StatusOK

	// Ping database
	if _, err := db.DefaultLanguage(); err != nil {
		dbStatus = "error: " + err.Error()
		statusCode = http.StatusServiceUnavailable
	}

	metrics := sys.GetMetricsSnapshot()
	status := "ok"
	if statusCode != http.StatusOK {
		status = "degraded"
	}

	resp := HealthResponse{
		Status:        status,
		Version:       sys.Version,
		UptimeSeconds: metrics.UptimeSeconds,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Database:      dbStatus,
		Channels: map[string]bool{
			"web":        sys.Options.Chat,
			"chat_audio": sys.Options.ChatAudio,
			"asterisk":   sys.Options.Asterisk,
			"meta":       sys.Options.MetaUrl != "",
			"telegram":   sys.Options.TelegramToken != "",
		},
		ActiveSessions: pbx.GetActiveSessionCount(),
	}

	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(resp)
}
