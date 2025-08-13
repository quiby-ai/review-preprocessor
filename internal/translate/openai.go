package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type OpenAIClient struct {
	Endpoint string
	Model    string
	APIKey   string
	Timeout  time.Duration
}

func NewOpenAIClient(endpoint, model, apiKey string, timeout time.Duration) *OpenAIClient {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	return &OpenAIClient{Endpoint: endpoint, Model: model, APIKey: apiKey, Timeout: timeout}
}

type openAIRequest struct {
	Model          string          `json:"model"`
	ResponseFormat map[string]any  `json:"response_format"`
	Messages       []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (c *OpenAIClient) TranslateBatch(ctx context.Context, items []Item, target string) (map[string]Result, error) {
	if len(items) == 0 {
		return map[string]Result{}, nil
	}
	payload := map[string]any{
		"items":  items,
		"target": target,
	}
	prompt, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	reqBody := openAIRequest{
		Model:          c.Model,
		ResponseFormat: map[string]any{"type": "json_object"},
		Messages: []openAIMessage{
			{Role: "system", Content: "You are a translator. Return strict JSON. If input is already in target language (English), set translated to empty string. Schema: {\"items\":[{\"id\":\"\",\"lang\":\"\",\"translated\":\"\"}]}"},
			{Role: "user", Content: string(prompt)},
		},
	}
	b, _ := json.Marshal(reqBody)
	httpClient := &http.Client{Timeout: c.Timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai status: %s", resp.Status)
	}
	var oaResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&oaResp); err != nil {
		return nil, err
	}
	if len(oaResp.Choices) == 0 {
		return nil, fmt.Errorf("openai: empty choices")
	}
	// Parse model JSON content
	var parsed struct {
		Items []Result `json:"items"`
	}
	if err := json.Unmarshal([]byte(oaResp.Choices[0].Message.Content), &parsed); err != nil {
		return nil, err
	}
	out := make(map[string]Result, len(parsed.Items))
	for _, r := range parsed.Items {
		out[r.ID] = r
	}
	return out, nil
}
