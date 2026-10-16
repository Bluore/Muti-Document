// Package embedding 提供 OpenAI 兼容 /embeddings 接口的文本向量化客户端。
// DeepSeek 没有 embedding 接口, 因此这里只依赖一个可配置的兼容端点。
package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// defaultTimeout 单次请求的默认超时
const defaultTimeout = 30 * time.Second

// maxResponseBytes 限制读取的响应体大小, 避免异常响应打爆内存
const maxResponseBytes = 8 << 20

// Config 描述一个 OpenAI 兼容的 embedding 服务。
// BaseURL 需包含版本前缀(如 https://api.openai.com/v1)。
type Config struct {
	BaseURL string
	APIKey  string
	Model   string
	// Dimensions 为期望的输出维度, 0 表示用模型默认维度
	Dimensions int
	Timeout    time.Duration
}

// Client 把文本批量向量化
type Client struct {
	cfg  Config
	http *http.Client
}

func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: timeout},
	}
}

type embedRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions,omitempty"`
}

type embedResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// Embed 批量向量化, 返回的向量与入参顺序一致
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	if c.cfg.BaseURL == "" || c.cfg.Model == "" {
		return nil, errors.New("embedding: base url or model not configured")
	}

	body, err := json.Marshal(embedRequest{
		Model:      c.cfg.Model,
		Input:      texts,
		Dimensions: c.cfg.Dimensions,
	})
	if err != nil {
		return nil, fmt.Errorf("embedding: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(c.cfg.BaseURL, "/")+"/embeddings",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("embedding: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding: request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("embedding: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding: status %d: %s", resp.StatusCode, truncate(raw, 512))
	}

	var parsed embedResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("embedding: decode response: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("embedding: %s: %s", parsed.Error.Type, parsed.Error.Message)
	}
	if len(parsed.Data) != len(texts) {
		return nil, fmt.Errorf("embedding: got %d vectors for %d texts", len(parsed.Data), len(texts))
	}

	vectors := make([][]float32, len(texts))
	for _, item := range parsed.Data {
		if item.Index < 0 || item.Index >= len(vectors) {
			return nil, fmt.Errorf("embedding: index %d out of range", item.Index)
		}
		if c.cfg.Dimensions > 0 && len(item.Embedding) != c.cfg.Dimensions {
			return nil, fmt.Errorf("embedding: dimension %d, want %d", len(item.Embedding), c.cfg.Dimensions)
		}
		vectors[item.Index] = item.Embedding
	}
	for i, vec := range vectors {
		if len(vec) == 0 {
			return nil, fmt.Errorf("embedding: empty vector at index %d", i)
		}
	}
	return vectors, nil
}

// EmbedOne 向量化单条文本
func (c *Client) EmbedOne(ctx context.Context, text string) ([]float32, error) {
	vectors, err := c.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, errors.New("embedding: empty result")
	}
	return vectors[0], nil
}

func truncate(raw []byte, max int) string {
	if len(raw) <= max {
		return string(raw)
	}
	return string(raw[:max]) + "..."
}
