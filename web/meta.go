// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package web

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/eja/pbx/db"
	"github.com/eja/pbx/i18n"
	"github.com/eja/pbx/meta"
	"github.com/eja/pbx/pbx"
	"github.com/eja/pbx/sys"
)

type typeMetaMessage struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				Messages []struct {
					From string `json:"from"`
					Text *struct {
						Body string `json:"body"`
					} `json:"text,omitempty"`
					Audio *struct {
						ID string `json:"id"`
					} `json:"audio,omitempty"`
					Image *struct {
						ID      string `json:"id"`
						Caption string `json:"caption,omitempty"`
					} `json:"image,omitempty"`
					Document *struct {
						ID       string `json:"id"`
						Filename string `json:"filename,omitempty"`
						Caption  string `json:"caption,omitempty"`
					} `json:"document,omitempty"`
					ID string `json:"id"`
				} `json:"messages"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

func metaRouter(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !webhookRateLimiter.allow(ip) {
		http.Error(w, "Too many requests", http.StatusTooManyRequests)
		return
	}

	if r.Method == http.MethodGet {
		hubMode := r.URL.Query().Get("hub.mode")
		verifyToken := r.URL.Query().Get("hub.verify_token")
		metaToken := sys.Options.MetaToken
		if metaToken == "" {
			metaToken = db.Settings()["metaToken"]
		}
		if hubMode == "subscribe" && metaToken != "" && subtle.ConstantTimeCompare([]byte(verifyToken), []byte(metaToken)) == 1 {
			w.Write([]byte(r.URL.Query().Get("hub.challenge")))
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Token verification error"))
		}
	} else {
		const platform = "meta"
		var metaMessage typeMetaMessage

		bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}

		metaAuth := sys.Options.MetaAuth
		if metaAuth == "" {
			metaAuth = db.Settings()["metaAuth"]
		}
		if metaAuth != "" {
			sig := r.Header.Get("X-Hub-Signature-256")
			mac := hmac.New(sha256.New, []byte(metaAuth))
			mac.Write(bodyBytes)
			expectedSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
			if subtle.ConstantTimeCompare([]byte(sig), []byte(expectedSig)) != 1 {
				slog.Warn("Meta, signature verification failed")
				http.Error(w, "Invalid signature", http.StatusUnauthorized)
				return
			}
		}

		err = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&metaMessage)
		if err != nil {
			errMessage := "Error decoding request body"
			http.Error(w, errMessage, http.StatusBadRequest)
			slog.Warn("Meta error", "error", errMessage)
			return
		}

		slog.Debug("Meta, incoming message", "message", metaMessage)
		if len(metaMessage.Entry) > 0 {
			changes := metaMessage.Entry[0].Changes
			if len(changes) > 0 {
				value := changes[0].Value

				if len(value.Messages) > 0 {
					message := value.Messages[0]
					userId := message.From
					chatId := message.ID
					aiSettings := db.Settings()

					user, err := db.UserGet(userId)
					if user == nil && sys.Number(aiSettings["userRestricted"]) < 1 {
						user = make(map[string]string)
						user["welcome"] = "1"
						user["language"] = aiSettings["language"]
						user["audio"] = "2"
					}
					if aiSettings["asrProvider"] == "" {
						user["audio"] = "0"
					}

					if err == nil && user != nil {
						if err := meta.SendStatus(chatId, "read"); err != nil {
							slog.Warn("Meta, send status error", "user", userId, "chat", chatId, "error", err)
						}

						if sys.Number(user["welcome"]) < 1 {
							_, actionResponse := db.ChatAction("chat", "/welcome", user["language"])
							meta.SendText(userId, actionResponse)
							db.UserUpdate(userId, "welcome", "1")
						}

						if message.Text != nil && message.Text.Body != "" {
							response, err := pbx.Text(userId, user["language"], message.Text.Body)
							if err != nil {
								response = i18n.Translate(user["language"], "error")
								slog.Warn("Meta, process text error", "user", userId, "chat", chatId, "error", err)
							}
							if err := meta.SendText(userId, response); err != nil {
								slog.Warn("Meta, send text error", "user", userId, "error", err)
							}
						} else if message.Audio != nil {
							if sys.Number(user["audio"]) > 0 {
								response, err := pbx.Audio(
									platform,
									userId,
									user["language"],
									chatId,
									message.Audio.ID,
									sys.Number(user["audio"]) > 1,
								)
								if err != nil {
									slog.Warn("Meta, process audio error", "user", userId, "chat", chatId, "error", err)
									if err := meta.SendText(userId, i18n.Translate(user["language"], "error")); err != nil {
										slog.Warn("Meta, send text error", "user", userId, "chat", chatId, "error", err)
									}
								}
								if response != "" {
									if err := meta.SendText(userId, response); err != nil {
										slog.Warn("Meta, send text error", "user", userId, "chat", chatId, "error", err)
									}
								}
							} else {
								if err := meta.SendText(userId, i18n.Translate(user["language"], "audio_disabled")); err != nil {
									slog.Warn("Meta, send text error", "user", userId, "chat", chatId, "error", err)
								}
							}
						} else if message.Image != nil {
							prompt := message.Image.Caption
							if prompt == "" {
								prompt = "Image received."
							}
							filePath := fmt.Sprintf("%s/%s.jpg", sys.Options.MediaPath, message.Image.ID)
							_ = meta.MediaGet(message.Image.ID, filePath)
							textInput := fmt.Sprintf("[Image attached: %s] %s", message.Image.ID, prompt)
							response, err := pbx.Text(userId, user["language"], textInput)
							if err != nil {
								response = i18n.Translate(user["language"], "error")
							}
							_ = meta.SendText(userId, response)
						} else if message.Document != nil {
							prompt := message.Document.Caption
							if prompt == "" {
								prompt = "Document received."
							}
							filePath := fmt.Sprintf("%s/%s_%s", sys.Options.MediaPath, message.Document.ID, message.Document.Filename)
							_ = meta.MediaGet(message.Document.ID, filePath)
							textInput := fmt.Sprintf("[Document attached: %s] %s", message.Document.Filename, prompt)
							response, err := pbx.Text(userId, user["language"], textInput)
							if err != nil {
								response = i18n.Translate(user["language"], "error")
							}
							_ = meta.SendText(userId, response)
						}
					} else {
						if err := meta.SendText(userId, i18n.Translate("", "user_unknown")); err != nil {
							slog.Warn("Meta, send text error", "user", userId, "chat", chatId, "error", err)
						}
					}
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	}
}
