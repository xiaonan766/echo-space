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

const maxDashScopeErrorBody = 4096

type DashScopeClient struct {
	apiKey    string
	baseURL   string
	model     string
	dimension int
	resLevel  int
	http      *http.Client
}

type dashScopeRequest struct {
	Model      string              `json:"model"`
	Input      dashScopeInput      `json:"input"`
	Parameters dashScopeParameters `json:"parameters"`
}

type dashScopeInput struct {
	Contents []map[string]string `json:"contents"`
}

type dashScopeParameters struct {
	Dimension  int    `json:"dimension"`
	OutputType string `json:"output_type"`
	ResLevel   int    `json:"res_level"`
}

type dashScopeResponse struct {
	Output struct {
		Embeddings []struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
			Type      string    `json:"type"`
		} `json:"embeddings"`
	} `json:"output"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewDashScopeClient(apiKey, baseURL, model string, dimension int, httpClient *http.Client) (*DashScopeClient, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("DASHSCOPE_API_KEY is not configured")
	}
	if strings.TrimSpace(baseURL) == "" || strings.TrimSpace(model) == "" || dimension <= 0 {
		return nil, errors.New("dashscope embedding configuration is invalid")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &DashScopeClient{apiKey: apiKey, baseURL: baseURL, model: model, dimension: dimension, resLevel: 1, http: httpClient}, nil
}

func (c *DashScopeClient) Model() string { return c.model }

func (c *DashScopeClient) EmbedText(ctx context.Context, text string) ([]float32, error) {
	vectors, err := c.embed(ctx, []map[string]string{{"text": text}})
	if err != nil {
		return nil, err
	}
	return vectors[0], nil
}

func (c *DashScopeClient) EmbedImages(ctx context.Context, images [][]byte) ([][]float32, error) {
	if len(images) == 0 || len(images) > 4 {
		return nil, errors.New("each embedding batch must contain 1 to 4 images")
	}
	contents := make([]map[string]string, 0, len(images))
	for _, imageData := range images {
		contents = append(contents, map[string]string{"image": JPEGDataURI(imageData)})
	}
	return c.embed(ctx, contents)
}

func (c *DashScopeClient) embed(ctx context.Context, contents []map[string]string) ([][]float32, error) {
	payload, err := json.Marshal(dashScopeRequest{
		Model:      c.model,
		Input:      dashScopeInput{Contents: contents},
		Parameters: dashScopeParameters{Dimension: c.dimension, OutputType: "dense", ResLevel: c.resLevel},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal dashscope request: %w", err)
	}

	var response dashScopeResponse
	for attempt := 0; attempt < 3; attempt++ {
		response, err = c.do(ctx, payload)
		if err == nil {
			break
		}
		var retryable *retryableError
		if !errors.As(err, &retryable) || attempt == 2 {
			return nil, err
		}
		if err := waitRetry(ctx, time.Duration(1<<attempt)*300*time.Millisecond); err != nil {
			return nil, err
		}
	}

	if len(response.Output.Embeddings) != len(contents) {
		return nil, fmt.Errorf("dashscope returned %d embeddings, want %d", len(response.Output.Embeddings), len(contents))
	}
	vectors := make([][]float32, len(contents))
	for _, item := range response.Output.Embeddings {
		if item.Index < 0 || item.Index >= len(contents) || len(item.Embedding) != c.dimension {
			return nil, errors.New("dashscope returned an invalid embedding")
		}
		vectors[item.Index] = item.Embedding
	}
	for _, vector := range vectors {
		if len(vector) != c.dimension {
			return nil, errors.New("dashscope embedding order is incomplete")
		}
	}
	return vectors, nil
}

type retryableError struct{ status int }

func (e *retryableError) Error() string {
	return fmt.Sprintf("temporary dashscope error: status=%d", e.status)
}

func (c *DashScopeClient) do(ctx context.Context, payload []byte) (dashScopeResponse, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return dashScopeResponse{}, fmt.Errorf("create dashscope request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.apiKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return dashScopeResponse{}, fmt.Errorf("request dashscope embedding: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxDashScopeErrorBody+8*1024*1024))
	if err != nil {
		return dashScopeResponse{}, errors.New("read dashscope response failed")
	}
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 {
		return dashScopeResponse{}, &retryableError{status: response.StatusCode}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return dashScopeResponse{}, fmt.Errorf("dashscope rejected embedding request: status=%d", response.StatusCode)
	}
	var result dashScopeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return dashScopeResponse{}, errors.New("parse dashscope response failed")
	}
	if result.Code != "" {
		return dashScopeResponse{}, fmt.Errorf("dashscope embedding failed: code=%s", result.Code)
	}
	return result, nil
}

func waitRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
