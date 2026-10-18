// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package pbx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/eja/pbx/db"
	"github.com/eja/pbx/sys"
)

var llmHttpClient = &http.Client{Timeout: 120 * time.Second}

type LLMTool struct {
	Description string
	Parameters  any
	Callback    func(arguments string) (string, error)
}

type llmTool struct {
	Type     string          `json:"type"`
	Function llmToolFunction `json:"function"`
}

type llmToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type llmToolCall struct {
	ID           string              `json:"id"`
	Type         string              `json:"type"`
	Function     llmToolCallFunction `json:"function"`
	ExtraContent any                 `json:"extra_content,omitempty"`
}

type llmToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type llmRequestMessage struct {
	Role         string        `json:"role"`
	Content      string        `json:"content"`
	Name         string        `json:"name,omitempty"`
	ToolCallID   string        `json:"tool_call_id,omitempty"`
	ToolCalls    []llmToolCall `json:"tool_calls,omitempty"`
	ExtraContent any           `json:"extra_content,omitempty"`
}

type llmResponseMessage struct {
	Role      string        `json:"role"`
	Content   string        `json:"content"`
	ToolCalls []llmToolCall `json:"tool_calls,omitempty"`
}

type llmResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Choices []struct {
		Index        int                `json:"index"`
		Message      llmResponseMessage `json:"message"`
		Logprobs     any                `json:"logprobs"`
		FinishReason string             `json:"finish_reason"`
	} `json:"choices"`
}

type LLMDetails struct {
	Content          string `json:"content"`
	Model            string `json:"model"`
	TotalTokens      int    `json:"total_tokens"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	LatencyMs        int64  `json:"latency_ms"`
}

func LLM(messages []sys.TypeChatMessage, system string, tools map[string]LLMTool) (string, error) {
	details, err := LLMWithDetails(messages, system, tools)
	return details.Content, err
}

func LLMWithDetails(messages []sys.TypeChatMessage, system string, tools map[string]LLMTool) (LLMDetails, error) {
	start := time.Now()
	var details LLMDetails
	var totalTokens, promptTokens, compTokens int

	aiSettings := db.Settings()
	url := sys.Options.LlmUrl
	if url == "" {
		url = aiSettings["llmUrl"]
	}
	if url == "" {
		return details, fmt.Errorf("llm url cannot be empty")
	}
	model := aiSettings["llmModel"]
	token := aiSettings["llmToken"]
	if token == "" {
		token = sys.Options.AiToken
	}
	mcpURL := aiSettings["mcpUrl"]
	if mcpURL == "" {
		mcpURL = sys.Options.McpUrl
	}
	mcpToken := aiSettings["mcpToken"]
	if mcpToken == "" {
		mcpToken = sys.Options.McpToken
	}

	finalTools := make(map[string]LLMTool)
	maps.Copy(finalTools, tools)

	if mcpURL != "" {
		mcpPrompts, err := fetchMCPPrompts(mcpURL, mcpToken)
		if err != nil {
			slog.Warn("LLM, failed to fetch MCP prompts", "error", err)
		} else {
			for promptName := range mcpPrompts {
				promptText, err := getMCPPrompt(mcpURL, mcpToken, promptName, nil)
				if err == nil && promptText != "" {
					system = strings.ReplaceAll(system, "</system_prompt>", fmt.Sprintf("<pbx_mcp_prompt>%s</pbx_mcp_prompt></system_prompt>", promptText))
				}
			}
		}

		mcpTools, err := fetchMCPTools(mcpURL, mcpToken)
		if err != nil {
			slog.Warn("LLM, failed to fetch MCP tools", "error", err)
		} else {
			for name, t := range mcpTools {
				finalTools["mcp_"+name] = t
			}
		}
	}

	var reqMessages []llmRequestMessage
	reqMessages = append(reqMessages, llmRequestMessage{
		Role:    "system",
		Content: system,
	})
	for _, m := range messages {
		reqMessages = append(reqMessages, llmRequestMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	var oaTools []llmTool
	for name, t := range finalTools {
		oaTools = append(oaTools, llmTool{
			Type: "function",
			Function: llmToolFunction{
				Name:        name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}

	const maxIterations = 5
	for range maxIterations {
		reqData := map[string]any{
			"stream":   false,
			"model":    model,
			"messages": reqMessages,
		}
		if len(oaTools) > 0 {
			reqData["tools"] = oaTools
		}

		payload, err := json.Marshal(reqData)
		if err != nil {
			return details, fmt.Errorf("error marshaling JSON: %v", err)
		}

		const maxHttpRetries = 3
		var resp *http.Response
		var lastErr error
		for attempt := range maxHttpRetries {
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
			if err != nil {
				return details, fmt.Errorf("error creating request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err = llmHttpClient.Do(req)
			if err != nil {
				lastErr = err
				backoff := time.Duration(1<<attempt) * 500 * time.Millisecond
				slog.Warn("LLM network error, retrying", "attempt", attempt+1, "error", err, "backoff", backoff)
				time.Sleep(backoff)
				continue
			}

			if resp.StatusCode == http.StatusTooManyRequests || (resp.StatusCode >= 500 && resp.StatusCode <= 599) {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				lastErr = fmt.Errorf("LLM HTTP %d: %s", resp.StatusCode, string(body))

				backoff := time.Duration(1<<attempt) * 500 * time.Millisecond
				if ra := resp.Header.Get("Retry-After"); ra != "" {
					if secs, e := strconv.Atoi(ra); e == nil && secs > 0 {
						backoff = time.Duration(secs) * time.Second
					}
				}
				slog.Warn("LLM temporary error, retrying with backoff", "status", resp.StatusCode, "attempt", attempt+1, "backoff", backoff)
				time.Sleep(backoff)
				continue
			}

			lastErr = nil
			break
		}

		if lastErr != nil {
			return details, fmt.Errorf("LLM request failed after retries: %w", lastErr)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return details, fmt.Errorf("LLM HTTP %d: %s", resp.StatusCode, string(body))
		}

		var response llmResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		resp.Body.Close()
		if err != nil {
			return details, fmt.Errorf("error decoding JSON response: %v", err)
		}

		if len(response.Choices) == 0 {
			return details, fmt.Errorf("no choices returned in response")
		}

		promptTokens += response.Usage.PromptTokens
		compTokens += response.Usage.CompletionTokens
		totalTokens += response.Usage.TotalTokens

		msg := response.Choices[0].Message

		if len(msg.ToolCalls) == 0 {
			details.Content = msg.Content
			details.Model = response.Model
			if details.Model == "" {
				details.Model = model
			}
			details.PromptTokens = promptTokens
			details.CompletionTokens = compTokens
			details.TotalTokens = totalTokens
			details.LatencyMs = time.Since(start).Milliseconds()
			return details, nil
		}

		reqMessages = append(reqMessages, llmRequestMessage{
			Role:      "assistant",
			Content:   msg.Content,
			ToolCalls: msg.ToolCalls,
		})

		for _, tc := range msg.ToolCalls {
			funcName := tc.Function.Name
			args := tc.Function.Arguments

			var result string
			if tool, ok := finalTools[funcName]; ok {
				res, err := tool.Callback(args)
				if err != nil {
					result = fmt.Sprintf("Error executing function: %v", err)
				} else {
					result = res
				}
			} else {
				result = fmt.Sprintf("Error: function '%s' not found", funcName)
			}

			reqMessages = append(reqMessages, llmRequestMessage{
				Role:         "tool",
				ToolCallID:   tc.ID,
				Name:         funcName,
				Content:      result,
				ExtraContent: tc.ExtraContent,
			})
		}
	}

	return details, fmt.Errorf("exceeded max tool call iterations")
}
