// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package web

import (
	"net/http"

	"github.com/eja/pbx/sys"
)

func metricsRouter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	body := sys.GeneratePrometheusMetrics()
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}
