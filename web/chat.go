// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package web

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/eja/pbx/db"
	"github.com/eja/pbx/pbx"
	"github.com/eja/pbx/sys"
)

func generateChatID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func chatRouter(w http.ResponseWriter, r *http.Request) {
	const tag = "[chat]"
	chatID := ""
	language := ""
	aiSettings := db.Settings()

	cors := sys.Options.ChatCors
	if cors == "" {
		cors = aiSettings["chatCors"]
	}
	if cors == "" {
		cors = "*"
	}
	w.Header().Set("Access-Control-Allow-Origin", cors)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	ip := clientIP(r)
	if !chatRateLimiter.allow(ip) {
		w.Header().Set("Retry-After", "60")
		http.Error(w, "Too many requests", http.StatusTooManyRequests)
		return
	}

	if sys.Number(aiSettings["userRestricted"]) > 0 {
		user, pass, ok := r.BasicAuth()
		if ok && user != "" && pass != "" {
			row, err := db.UserGetWithPassword(user, pass)
			if err != nil || row["uuid"] == "" {
				ok = false
			} else {
				if row["language"] != "" {
					language = row["language"]
				}
			}
		} else {
			ok = false
		}
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted Area"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		chatID = user

	}

	if r.Method == http.MethodGet {
		query := r.URL.Query()
		if query.Get("uuid") == "" {
			if chatID == "" {
				chatID = generateChatID()
			}
			query.Set("uuid", chatID)
			if sys.Options.ChatAudio {
				query.Set("audio", "on")
			}
			r.URL.RawQuery = query.Encode()
			http.Redirect(w, r, r.URL.String(), http.StatusFound)
			return
		}

		htmlContent, err := embeddedFiles.ReadFile("assets/chat.html")
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(htmlContent)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if chatID == "" {
		chatID = r.URL.Query().Get("uuid")
		if chatID == "" {
			http.Error(w, "Missing uuid parameter", http.StatusBadRequest)
			return
		}
	}

	contentType := r.Header.Get("Content-Type")
	var userInput string
	var isAudio bool

	if strings.Contains(contentType, "application/json") {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB max
		var msg struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		userInput = msg.Text

	} else if strings.Contains(contentType, "multipart/form-data") {
		isAudio = true
		if err := r.ParseMultipartForm(50 << 20); err != nil { // 50 MB max
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("audio")
		if err != nil {
			http.Error(w, "Error retrieving audio", http.StatusBadRequest)
			return
		}
		defer file.Close()

		tmpIn, err := os.CreateTemp(sys.Options.MediaPath, "web-asr-*.audio")
		if err != nil {
			http.Error(w, "Failed to create temp file", http.StatusInternalServerError)
			return
		}
		defer os.Remove(tmpIn.Name())

		if _, err := io.Copy(tmpIn, file); err != nil {
			http.Error(w, "Failed to save audio", http.StatusInternalServerError)
			return
		}
		tmpIn.Close()

		userInput, err = pbx.ASR(tmpIn.Name(), language)
		if err != nil {
			slog.Error("Chat, ASR internal error", "error", err)
			http.Error(w, "STT error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		defer os.Remove(tmpIn.Name() + ".google")
		defer os.Remove(tmpIn.Name() + ".whisper")

	} else {
		http.Error(w, "Unsupported Content-Type", http.StatusBadRequest)
		return
	}

	llmResponse, err := pbx.Text(chatID, language, userInput)
	if err != nil {
		slog.Error("Chat, internal error", "error", err)
		http.Error(w, "LLM Processing error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if isAudio {
		tmpOut, err := os.CreateTemp(sys.Options.MediaPath, "web-tts-*.audio")
		if err != nil {
			http.Error(w, "Failed to create temp file", http.StatusInternalServerError)
			return
		}
		tmpOut.Close()
		defer os.Remove(tmpOut.Name())

		if err := pbx.TTS(llmResponse, language, tmpOut.Name()); err != nil {
			slog.Error("Chat, TTS internal error", "error", err)
			http.Error(w, "TTS error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		audioData, err := os.ReadFile(tmpOut.Name())
		if err != nil {
			http.Error(w, "Failed to read TTS audio", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "audio/wav")
		w.Write(audioData)
	} else {
		if r.URL.Query().Get("stream") == "true" || strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			if flusher, ok := w.(http.Flusher); ok {
				words := strings.Fields(llmResponse)
				for i, word := range words {
					isDone := i == len(words)-1
					data, _ := json.Marshal(map[string]any{
						"delta": word + " ",
						"done":  isDone,
					})
					fmt.Fprintf(w, "data: %s\n\n", string(data))
					flusher.Flush()
					time.Sleep(20 * time.Millisecond)
				}
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"text": llmResponse})
	}
}
