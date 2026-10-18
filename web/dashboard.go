// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package web

import (
	"encoding/json"
	"net/http"
	"runtime"
	"strconv"

	"github.com/eja/pbx/db"
	"github.com/eja/pbx/pbx"
	"github.com/eja/pbx/sys"
)

func checkDashboardAuth(w http.ResponseWriter, r *http.Request) bool {
	aiSettings := db.Settings()
	if sys.Number(aiSettings["userRestricted"]) > 0 {
		user, pass, ok := r.BasicAuth()
		if ok && user != "" && pass != "" {
			row, err := db.UserGetWithPassword(user, pass)
			if err != nil || row["uuid"] == "" {
				ok = false
			}
		} else {
			ok = false
		}
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="PBX AI Dashboard"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return false
		}
	}
	return true
}

func dashboardRedirectRouter(w http.ResponseWriter, r *http.Request) {
	if !checkDashboardAuth(w, r) {
		return
	}
	http.Redirect(w, r, "/pbx/dashboard.html", http.StatusFound)
}

func dashboardStatsHandler(w http.ResponseWriter, r *http.Request) {
	if !checkDashboardAuth(w, r) {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	mem := new(runtime.MemStats)
	runtime.ReadMemStats(mem)

	rt := sys.GetMetricsSnapshot()
	dbStats, err := db.GetDashboardStats()
	if err != nil {
		dbStats = db.DbDashboardStats{}
	}

	aiSettings := db.Settings()
	llmModel := aiSettings["llmModel"]
	llmUrl := sys.Options.LlmUrl
	if llmUrl == "" {
		llmUrl = aiSettings["llmUrl"]
	}

	// Cost estimation: approx $0.15 per 1M tokens (gpt-4o-mini baseline average)
	totalTokens := dbStats.TotalTokens
	if rt.TotalTokens > totalTokens {
		totalTokens = rt.TotalTokens
	}
	estimatedCostUSD := float64(totalTokens) * 0.0000003

	resp := map[string]any{
		"system": map[string]any{
			"version":        sys.Version,
			"uptime_seconds": rt.UptimeSeconds,
			"active_sessions": rt.ActiveSessions,
			"goroutines":     runtime.NumGoroutine(),
			"alloc_mb":       float64(mem.Alloc) / 1024 / 1024,
			"sys_mb":         float64(mem.Sys) / 1024 / 1024,
		},
		"metrics": map[string]any{
			"total_requests":     dbStats.TotalRequests,
			"rt_requests":        rt.TotalRequests,
			"total_tokens":       totalTokens,
			"estimated_cost_usd": estimatedCostUSD,
			"avg_latency_ms":     dbStats.AvgLatencyMs,
			"total_errors":       dbStats.TotalErrors,
			"channel_counts":     dbStats.ChannelCounts,
			"rt_channel_counts":  rt.ChannelRequests,
		},
		"providers": map[string]any{
			"llm": map[string]any{
				"configured": llmUrl != "",
				"model":      llmModel,
			},
			"meta": map[string]any{
				"configured": sys.Options.MetaUrl != "" || aiSettings["metaToken"] != "",
			},
			"telegram": map[string]any{
				"configured": sys.Options.TelegramToken != "" || aiSettings["telegramToken"] != "",
			},
			"asterisk": map[string]any{
				"enabled": sys.Options.Asterisk,
				"agi":     sys.Options.AsteriskAgi,
				"ari":     sys.Options.AsteriskAri,
			},
			"database": map[string]any{
				"type": sys.Options.DbType,
				"name": sys.Options.DbName,
			},
		},
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func dashboardLogsHandler(w http.ResponseWriter, r *http.Request) {
	if !checkDashboardAuth(w, r) {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	channel := r.URL.Query().Get("channel")
	search := r.URL.Query().Get("search")

	rows, err := db.GetRecentLogs(limit, channel, search)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch logs"}`, http.StatusInternalServerError)
		return
	}
	if rows == nil {
		rows = []map[string]string{}
	}

	_ = json.NewEncoder(w).Encode(rows)
}

func dashboardSessionsHandler(w http.ResponseWriter, r *http.Request) {
	if !checkDashboardAuth(w, r) {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	sessions := pbx.GetActiveSessions(true)
	if sessions == nil {
		sessions = []pbx.SessionSummary{}
	}
	_ = json.NewEncoder(w).Encode(sessions)
}
