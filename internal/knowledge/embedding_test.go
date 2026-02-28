package knowledge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockEmbeddingServer 启动一个伪 OpenAI embedding 服务，返回指定维度的零向量。
func mockEmbeddingServer(t *testing.T, dim int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req embeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		data := make([]struct {
			Embedding []float32 `json:"embedding"`
		}, len(req.Input))
		for i := range data {
			data[i].Embedding = make([]float32, dim)
		}
		resp := embeddingResponse{Data: data}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestEmbedTexts_Success(t *testing.T) {
	srv := mockEmbeddingServer(t, embeddingDim)
	defer srv.Close()

	texts := []string{"修行之道", "Go练气士", "AI应用"}
	vecs, err := embedTexts(context.Background(), "fake-key", srv.URL, defaultEmbeddingModel, texts)
	if err != nil {
		t.Fatalf("embedTexts error: %v", err)
	}
	if len(vecs) != len(texts) {
		t.Fatalf("expected %d vectors, got %d", len(texts), len(vecs))
	}
	for i, v := range vecs {
		if len(v) != embeddingDim {
			t.Errorf("vecs[%d] dim = %d, want %d", i, len(v), embeddingDim)
		}
	}
}

func TestEmbedTexts_EmptyInput(t *testing.T) {
	srv := mockEmbeddingServer(t, embeddingDim)
	defer srv.Close()

	vecs, err := embedTexts(context.Background(), "fake-key", srv.URL, defaultEmbeddingModel, nil)
	if err != nil {
		t.Fatalf("unexpected error for empty input: %v", err)
	}
	if len(vecs) != 0 {
		t.Errorf("expected 0 vectors for empty input, got %d", len(vecs))
	}
}

func TestEmbedTexts_APIError(t *testing.T) {
	// 服务返回 API 错误
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(embeddingResponse{
			Error: &struct {
				Message string `json:"message"`
			}{Message: "quota exceeded"},
		})
	}))
	defer srv.Close()

	_, err := embedTexts(context.Background(), "fake-key", srv.URL, defaultEmbeddingModel, []string{"test"})
	if err == nil {
		t.Fatal("expected error for API error response, got nil")
	}
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestEmbedTexts_BatchSplit(t *testing.T) {
	// 验证超过 96 条输入会被分批
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var req embeddingRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		data := make([]struct {
			Embedding []float32 `json:"embedding"`
		}, len(req.Input))
		for i := range data {
			data[i].Embedding = make([]float32, embeddingDim)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(embeddingResponse{Data: data})
	}))
	defer srv.Close()

	texts := make([]string, 100) // 100 条 > batchSize(96)
	for i := range texts {
		texts[i] = "文本"
	}

	vecs, err := embedTexts(context.Background(), "key", srv.URL, defaultEmbeddingModel, texts)
	if err != nil {
		t.Fatalf("embedTexts error: %v", err)
	}
	if len(vecs) != 100 {
		t.Errorf("expected 100 vectors, got %d", len(vecs))
	}
	if callCount < 2 {
		t.Errorf("expected at least 2 HTTP calls for batch split, got %d", callCount)
	}
}
