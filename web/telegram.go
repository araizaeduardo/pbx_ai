// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package web

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/eja/pbx/db"
	"github.com/eja/pbx/i18n"
	"github.com/eja/pbx/pbx"
	"github.com/eja/pbx/sys"
	"github.com/eja/pbx/telegram"
)

type typeTelegramMessage struct {
	Message struct {
		From struct {
			Id           int    `json:"id"`
			LanguageCode string `json:"language_code"`
		} `json:"from"`
		Chat struct {
			Id int `json:"id"`
		} `json:"chat"`
		Text    string `json:"text,omitempty"`
		Caption string `json:"caption,omitempty"`
		Voice   struct {
			FileId string `json:"file_id"`
		} `json:"voice"`
		Photo []struct {
			FileId string `json:"file_id"`
		} `json:"photo,omitempty"`
		Document *struct {
			FileId   string `json:"file_id"`
			FileName string `json:"file_name"`
			MimeType string `json:"mime_type"`
		} `json:"document,omitempty"`
		Context struct {
			Forwarded bool `json:"forwarded"`
		} `json:"context"`
	} `json:"message"`
}

func telegramRouter(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !webhookRateLimiter.allow(ip) {
		http.Error(w, "Too many requests", http.StatusTooManyRequests)
		return
	}

	if r.Method == http.MethodPost {
		const platform = "telegram"
		aiSettings := db.Settings()

		secret := aiSettings["telegramSecret"]
		if secret != "" {
			tokenHeader := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
			if subtle.ConstantTimeCompare([]byte(tokenHeader), []byte(secret)) != 1 {
				slog.Warn("Telegram, invalid secret token")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		var telegramMessage typeTelegramMessage

		err := json.NewDecoder(r.Body).Decode(&telegramMessage)
		if err != nil {
			errMessage := "Error decoding request body"
			http.Error(w, errMessage, http.StatusBadRequest)
			slog.Warn("Telegram, json decoding problem", "error", errMessage)
			return
		}

		slog.Debug("Telegram, incoming message", "message", telegramMessage)
		userId := fmt.Sprintf("TG.%d", telegramMessage.Message.From.Id)
		chatId := fmt.Sprintf("%d", telegramMessage.Message.Chat.Id)
		chatLanguage := telegramMessage.Message.From.LanguageCode

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
			if sys.Number(user["welcome"]) < 1 {
				_, actionResponse := db.ChatAction("chat", "/welcome", user["language"])
				telegram.SendText(chatId, actionResponse)
				db.UserUpdate(userId, "welcome", "1")
			}

			if text := telegramMessage.Message.Text; text != "" {
				response, err := pbx.Text(userId, user["language"], text)
				if err != nil {
					response = i18n.Translate(user["language"], "error")
					slog.Warn("Telegram, process text problem", "user", userId, "chat", chatId, "error", err)
				}
				if err := telegram.SendText(chatId, response); err != nil {
					slog.Warn("Telegram, send text problem", "user", userId, "chat", chatId, "error", err)
				}
			}

			if voice := telegramMessage.Message.Voice; voice.FileId != "" {
				if sys.Number(user["audio"]) > 0 {
					response, err := pbx.Audio(
						platform,
						userId,
						user["language"],
						chatId,
						voice.FileId,
						sys.Number(user["audio"]) > 1,
					)
					if err != nil {
						slog.Warn("Telegram, process audio problem", "user", userId, "chat", chatId, "error", err)
						if err := telegram.SendText(chatId, i18n.Translate(chatLanguage, "error")); err != nil {
							slog.Warn("Telegram, process text problem", "user", userId, "chat", chatId, "error", err)
						}
					}
					if response != "" {
						if err := telegram.SendText(chatId, response); err != nil {
							slog.Warn("Telegram, process text problem", "user", userId, "chat", chatId, "error", err)
						}
					}
				} else {
					if err := telegram.SendText(chatId, i18n.Translate(user["language"], "audio_disabled")); err != nil {
						slog.Warn("Telegram, process text problem", "user", userId, "chat", chatId, "error", err)
					}
				}
			}

			if len(telegramMessage.Message.Photo) > 0 {
				photo := telegramMessage.Message.Photo[len(telegramMessage.Message.Photo)-1]
				prompt := telegramMessage.Message.Caption
				if prompt == "" {
					prompt = "Image received."
				}
				filePath := fmt.Sprintf("%s/%s.jpg", sys.Options.MediaPath, photo.FileId)
				_ = telegram.MediaGet(photo.FileId, filePath)
				textInput := fmt.Sprintf("[Image attached: %s] %s", photo.FileId, prompt)
				response, err := pbx.Text(userId, user["language"], textInput)
				if err != nil {
					response = i18n.Translate(user["language"], "error")
				}
				_ = telegram.SendText(chatId, response)
			} else if telegramMessage.Message.Document != nil && telegramMessage.Message.Document.FileId != "" {
				doc := telegramMessage.Message.Document
				prompt := telegramMessage.Message.Caption
				if prompt == "" {
					prompt = "Document received."
				}
				filePath := fmt.Sprintf("%s/%s_%s", sys.Options.MediaPath, doc.FileId, doc.FileName)
				_ = telegram.MediaGet(doc.FileId, filePath)
				textInput := fmt.Sprintf("[Document attached: %s] %s", doc.FileName, prompt)
				response, err := pbx.Text(userId, user["language"], textInput)
				if err != nil {
					response = i18n.Translate(user["language"], "error")
				}
				_ = telegram.SendText(chatId, response)
			}
		} else {
			if err := telegram.SendText(chatId, i18n.Translate(chatLanguage, "user_unknown")); err != nil {
				slog.Warn("Telegram, process text problem", "user", userId, "chat", chatId, "error", err)
			}
		}
	}
}
