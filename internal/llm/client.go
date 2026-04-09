package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Config struct {
	Enabled  bool   `toml:"enabled"`
	Endpoint string `toml:"endpoint"`
	Model    string `toml:"model"`
	APIKey   string `toml:"api_key"`
	TimeoutS int    `toml:"timeout_s"`
}

type Client struct {
	cfg    Config
	http   *http.Client
}

func New(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: time.Duration(cfg.TimeoutS) * time.Second},
	}
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
}

// Analyze sendet einen Prompt an das LLM und gibt die Antwort zurück.
func (c *Client) Analyze(ctx context.Context, prompt string) (string, error) {
	if !c.cfg.Enabled {
		return "", nil
	}

	body, _ := json.Marshal(chatRequest{
		Model:    c.cfg.Model,
		Messages: []message{{Role: "user", Content: prompt}},
		Stream:   false,
	})

	req, err := http.NewRequestWithContext(ctx, "POST",
		c.cfg.Endpoint+"/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("LLM-Anfrage fehlgeschlagen: %w", err)
	}
	defer resp.Body.Close()

	// Raw Body lesen für bessere Fehlerdiagnose
	var raw bytes.Buffer
	raw.ReadFrom(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM HTTP %d: %s", resp.StatusCode, raw.String())
	}

	var result chatResponse
	if err := json.Unmarshal(raw.Bytes(), &result); err != nil {
		return "", fmt.Errorf("LLM-Antwort ungültig: %w (body: %s)", err, raw.String())
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("LLM: leere Antwort (body: %s)", raw.String())
	}
	return result.Choices[0].Message.Content, nil
}
