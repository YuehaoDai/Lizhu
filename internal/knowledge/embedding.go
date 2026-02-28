// Package knowledge — OpenAI Embedding 工具函数（直接调用 HTTP API，无额外依赖）。
package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultEmbeddingModel = "text-embedding-3-small"
	embeddingDim          = 1536
	embeddingEndpoint     = "https://api.openai.com/v1/embeddings"
)

type embeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// embedTexts 批量向量化文本，返回每个文本对应的 float32 向量。
func embedTexts(ctx context.Context, apiKey, baseURL, model string, texts []string) ([][]float32, error) {
	if model == "" {
		model = defaultEmbeddingModel
	}
	endpoint := embeddingEndpoint
	if baseURL != "" {
		endpoint = baseURL + "/embeddings"
	}

	// 每批最多 96 条（避免单次请求过大）
	const batchSize = 96
	var allVecs [][]float32
	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}
		vecs, err := embedBatch(ctx, apiKey, endpoint, model, texts[i:end])
		if err != nil {
			return nil, err
		}
		allVecs = append(allVecs, vecs...)
	}
	return allVecs, nil
}

func embedBatch(ctx context.Context, apiKey, endpoint, model string, texts []string) ([][]float32, error) {
	body, _ := json.Marshal(embeddingRequest{Model: model, Input: texts})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	cli := &http.Client{Timeout: 60 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var er embeddingResponse
	if err := json.Unmarshal(raw, &er); err != nil {
		return nil, fmt.Errorf("parse embedding response: %w", err)
	}
	if er.Error != nil {
		return nil, fmt.Errorf("embedding api error: %s", er.Error.Message)
	}
	vecs := make([][]float32, len(er.Data))
	for i, d := range er.Data {
		vecs[i] = d.Embedding
	}
	return vecs, nil
}
