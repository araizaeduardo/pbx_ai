// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package elevenlabs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/eja/pbx/db"
	"github.com/eja/pbx/sys"
)

var httpClient = &http.Client{Timeout: 60 * time.Second}

const (
	defaultVoice = "21m00Tcm4TlvDq8ikWAM" // Rachel
	defaultModel = "eleven_multilingual_v2"
	baseUrl      = "https://api.elevenlabs.io/v1/text-to-speech"
)

type ttsPayload struct {
	Text    string `json:"text"`
	ModelID string `json:"model_id"`
}

func TTS(filePath, text, languageCode string) error {
	aiSettings := db.Settings()

	token := aiSettings["ttsToken"]
	if token == "" {
		token = sys.Options.AiToken
	}
	if token == "" {
		return fmt.Errorf("elevenlabs: api token is empty")
	}

	voice := aiSettings["ttsVoice"]
	if langVoice := aiSettings["ttsVoice_"+languageCode]; langVoice != "" {
		voice = langVoice
	}
	if voice == "" {
		voice = defaultVoice
	}

	model := aiSettings["ttsModel"]
	if langModel := aiSettings["ttsModel_"+languageCode]; langModel != "" {
		model = langModel
	}
	if model == "" {
		model = defaultModel
	}

	url := fmt.Sprintf("%s/%s", baseUrl, voice)

	body, err := json.Marshal(ttsPayload{
		Text:    text,
		ModelID: model,
	})
	if err != nil {
		return fmt.Errorf("elevenlabs: encoding payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("elevenlabs: creating request: %w", err)
	}
	req.Header.Set("xi-api-key", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("elevenlabs: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("elevenlabs: api error %d: %s", resp.StatusCode, string(respBody))
	}

	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("elevenlabs: creating audio file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("elevenlabs: saving audio file: %w", err)
	}

	return nil
}
