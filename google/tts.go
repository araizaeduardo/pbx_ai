// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package google

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/eja/pbx/db"
	"github.com/eja/pbx/sys"
)

type TTSRequest struct {
	Input struct {
		Text string `json:"text"`
	} `json:"input"`
	AudioConfig struct {
		AudioEncoding string `json:"audioEncoding"`
	} `json:"audioConfig"`
	Voice struct {
		SsmlGender   string `json:"ssmlGender"`
		LanguageCode string `json:"languageCode"`
	} `json:"voice"`
}

func splitTextForTTS(text string, maxLen int) []string {
	clean := strings.TrimSpace(text)
	if len(clean) <= maxLen {
		return []string{clean}
	}

	var chunks []string
	var current strings.Builder

	words := strings.Fields(clean)
	for _, word := range words {
		if len(word) > maxLen {
			if current.Len() > 0 {
				chunks = append(chunks, strings.TrimSpace(current.String()))
				current.Reset()
			}
			chunks = append(chunks, word)
			continue
		}

		if current.Len() > 0 && current.Len()+len(word)+1 > maxLen {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
		}

		if current.Len() > 0 {
			current.WriteString(" ")
		}
		current.WriteString(word)

		if current.Len() >= maxLen-50 {
			if strings.HasSuffix(word, ".") || strings.HasSuffix(word, "?") || strings.HasSuffix(word, "!") || strings.HasSuffix(word, ":") || strings.HasSuffix(word, ";") {
				chunks = append(chunks, strings.TrimSpace(current.String()))
				current.Reset()
			}
		}
	}

	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}
	return chunks
}

func TTS(filePath string, text string, languageCode string) error {
	aiSettings := db.Settings()
	apiKey := aiSettings["ttsToken"]
	if apiKey == "" {
		apiKey = sys.Options.AiToken
	}

	if apiKey == "" {
		lang := languageCode
		if len(lang) > 2 {
			lang = lang[:2]
		}
		if lang == "" {
			lang = "es"
		}

		chunks := splitTextForTTS(text, 140)
		var combinedAudio []byte

		for _, chunk := range chunks {
			chunk = strings.TrimSpace(chunk)
			if chunk == "" {
				continue
			}
			reqUrl := fmt.Sprintf("https://translate.google.com/translate_tts?ie=UTF-8&client=tw-ob&tl=%s&q=%s",
				url.QueryEscape(lang), url.QueryEscape(chunk))
			req, err := http.NewRequest("GET", reqUrl, nil)
			if err != nil {
				return err
			}
			req.Header.Set("User-Agent", "Mozilla/5.0")
			resp, err := httpClient.Do(req)
			if err != nil {
				return fmt.Errorf("google translate tts failed: %w", err)
			}
			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				return fmt.Errorf("google translate tts status %d for chunk: %s", resp.StatusCode, chunk)
			}
			audioData, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return err
			}
			combinedAudio = append(combinedAudio, audioData...)
		}

		if len(combinedAudio) == 0 {
			return fmt.Errorf("no audio content generated from google translate")
		}
		return os.WriteFile(filePath, combinedAudio, 0644)
	}
	gender := aiSettings["ttsGender"]
	if langGender := aiSettings["ttsGender_"+languageCode]; langGender != "" {
		gender = langGender
	}
	if gender == "" {
		gender = "FEMALE"
	}

	requestBody := TTSRequest{}
	requestBody.Input.Text = text
	requestBody.AudioConfig.AudioEncoding = "OGG_OPUS"
	requestBody.Voice.SsmlGender = gender
	requestBody.Voice.LanguageCode = languageCode

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %v", err)
	}

	url := "https://texttospeech.googleapis.com/v1/text:synthesize"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-Goog-Api-Key", apiKey)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	var respData struct {
		AudioContent string `json:"audioContent"`
	}
	err = json.Unmarshal(respBody, &respData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response body: %v", err)
	}

	if respData.AudioContent == "" {
		return fmt.Errorf("no audio content in response")
	}

	audioData, err := base64.StdEncoding.DecodeString(respData.AudioContent)
	if err != nil {
		return fmt.Errorf("failed to decode audio content: %v", err)
	}

	err = os.WriteFile(filePath, audioData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write audio file: %v", err)
	}

	return nil
}
