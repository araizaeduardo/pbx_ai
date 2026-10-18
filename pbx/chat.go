// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package pbx

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/eja/pbx/db"
	"github.com/eja/pbx/i18n"
	"github.com/eja/pbx/sys"
)

// maxHistoryMessages limits how many messages are kept per user.
const maxHistoryMessages = 50

var (
	historyMu     sync.Mutex
	history       = make(map[string][]sys.TypeChatMessage)
	historyTime   = make(map[string]time.Time)
	historyThread = make(map[string]string)
)

// historyAppend adds a message to the user history (trimmed to maxHistoryMessages)
// and returns a copy of the resulting history.
func historyAppend(userId string, msg sys.TypeChatMessage) []sys.TypeChatMessage {
	historyMu.Lock()
	defer historyMu.Unlock()
	hist := append(history[userId], msg)
	if len(hist) > maxHistoryMessages {
		hist = append([]sys.TypeChatMessage(nil), hist[len(hist)-maxHistoryMessages:]...)
	}
	history[userId] = hist
	historyTime[userId] = time.Now()
	return append([]sys.TypeChatMessage(nil), hist...)
}

// historyGet returns a copy of the user history.
func historyGet(userId string) []sys.TypeChatMessage {
	historyMu.Lock()
	defer historyMu.Unlock()
	return append([]sys.TypeChatMessage(nil), history[userId]...)
}

// historyReset removes all the stored state for the user.
func historyReset(userId string) {
	historyMu.Lock()
	defer historyMu.Unlock()
	delete(history, userId)
	delete(historyTime, userId)
	delete(historyThread, userId)
}

var ChatProcess func(userId, system, message, language string, history []sys.TypeChatMessage, tools map[string]LLMTool) (string, error)

func Chat(platform, userId, message, language string) (string, error) {
	var response, system, assistant string
	var err error
	aiSettings := db.Settings()
	action := false

	timeZoneDiff := time.Duration(sys.Number(aiSettings["timezone"]))
	timeZoneNow := time.Now().Add(timeZoneDiff * time.Hour).Format("Monday 02 of January 2006, 03:04 pm")

	if language == "" {
		language = i18n.DefaultLanguage()
	}

	slog.Debug("chat request", "language", language, "user", userId, "message", message)

	system = "<system_prompt>"
	if rows, err := db.SystemPrompt(platform); err != nil {
		return "", err
	} else {
		for _, row := range rows {
			system += fmt.Sprintf("<%s>%s</%s>\n", row["label"], row["prompt"], row["label"])
		}
	}

	system += "<pbx_core>"
	system += fmt.Sprintf("Now is %s.\n", timeZoneNow)
	if platform == "pbx" {
		system += fmt.Sprintf("Users's phone number is %s.\n", userId)
	}
	system += fmt.Sprintf("The user usually speaks in %s, so please answer in that language or the language of the question if not instructed otherwise.\n", i18n.LanguageCodeToInternal(language))
	system += "Always add a new line containing the language code, 2 chars, between square brackets that you have used to answer the question at the end of your response, like this: \n\n[en]\n"
	system += "</pbx_core>\n</system_prompt>"

	startReq := time.Now()
	var details LLMDetails

	if strings.HasPrefix(message, "/") {
		action = true
		parameters := strings.Split(message, " ")
		var actionFunction, actionResponse = db.ChatAction(platform, parameters[0], language)
		if actionFunction != "" {
			if Plugins[actionFunction] != nil {
				response = Plugins[actionFunction](userId, language, message, actionResponse)
			} else {
				slog.Warn("chat plugin not found", "message", message)
			}
		} else if actionResponse != "" {
			response = actionResponse
		}
	} else {
		_ = db.LogEntryWrite(db.LogEntry{
			Source:  userId,
			Message: message,
			Channel: platform,
		})
	}

	if response == "" {
		hist := historyAppend(userId, sys.TypeChatMessage{Role: "user", Content: message})

		if ChatProcess != nil {
			assistant, err = ChatProcess(userId, system, message, language, hist, Tools)
			details.Content = assistant
			details.LatencyMs = time.Since(startReq).Milliseconds()
		} else {
			details, err = LLMWithDetails(hist, system, Tools)
			assistant = details.Content
		}
		if err != nil {
			sys.RecordRequest(platform, false, time.Since(startReq).Milliseconds(), 0, details.Model, err)
			_ = db.LogEntryWrite(db.LogEntry{
				Source:    userId,
				Message:   message,
				Channel:   platform,
				Model:     details.Model,
				LatencyMs: time.Since(startReq).Milliseconds(),
				Error:     err.Error(),
			})
			slog.Error("chat error", "error", err)
			return "", err
		}
		response = assistant
		sys.RecordRequest(platform, true, details.LatencyMs, details.TotalTokens, details.Model, nil)

		historyAppend(userId, sys.TypeChatMessage{Role: "assistant", Content: response})
	}

	slog.Debug("chat response", "language", language, "user", userId, "response", response)
	if !action {
		_ = db.LogEntryWrite(db.LogEntry{
			Source:    userId,
			Message:   response,
			Channel:   platform,
			Tokens:    details.TotalTokens,
			LatencyMs: details.LatencyMs,
			Model:     details.Model,
		})
	}

	return response, nil
}

type SessionSummary struct {
	UserID         string                `json:"user_id"`
	MessageCount   int                   `json:"message_count"`
	LastActive     string                `json:"last_active"`
	LastActiveUnix int64                 `json:"last_active_unix"`
	LastMessage    string                `json:"last_message"`
	History        []sys.TypeChatMessage `json:"history,omitempty"`
}

func GetActiveSessions(includeHistory bool) []SessionSummary {
	historyMu.Lock()
	defer historyMu.Unlock()
	sessions := []SessionSummary{}
	for uid, msgs := range history {
		if len(msgs) == 0 {
			continue
		}
		lastMsg := ""
		if len(msgs) > 0 {
			lastMsg = msgs[len(msgs)-1].Content
		}
		t := historyTime[uid]
		s := SessionSummary{
			UserID:         uid,
			MessageCount:   len(msgs),
			LastActive:     t.Format(time.RFC3339),
			LastActiveUnix: t.Unix(),
			LastMessage:    lastMsg,
		}
		if includeHistory {
			s.History = append([]sys.TypeChatMessage(nil), msgs...)
		}
		sessions = append(sessions, s)
	}
	return sessions
}

func GetActiveSessionCount() int {
	historyMu.Lock()
	defer historyMu.Unlock()
	return len(history)
}

func init() {
	sys.ActiveSessionsProvider = GetActiveSessionCount
}
