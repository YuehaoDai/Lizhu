package librarian

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---- buildSummarizePrompt 测试 ----

func TestBuildSummarizePrompt_ContainsFileName(t *testing.T) {
	prompt := buildSummarizePrompt("notes.md", "内容正文")
	if !strings.Contains(prompt, "notes.md") {
		t.Error("prompt should contain file name")
	}
	if !strings.Contains(prompt, "内容正文") {
		t.Error("prompt should contain file content")
	}
}

func TestBuildSummarizePrompt_ContainsInstruction(t *testing.T) {
	prompt := buildSummarizePrompt("a.txt", "x")
	if !strings.Contains(prompt, "请整理") {
		t.Error("prompt should contain instruction keyword")
	}
}

// ---- systemPromptTemplate 完整性检查 ----

func TestSystemPromptTemplate_NotEmpty(t *testing.T) {
	if strings.TrimSpace(systemPromptTemplate) == "" {
		t.Error("systemPromptTemplate should not be empty")
	}
}

func TestSystemPromptTemplate_ContainsOutputFormat(t *testing.T) {
	if !strings.Contains(systemPromptTemplate, "【文件】") {
		t.Error("systemPromptTemplate should specify output format with 【文件】")
	}
	if !strings.Contains(systemPromptTemplate, "【摘要】") {
		t.Error("systemPromptTemplate should specify 【摘要】")
	}
	if !strings.Contains(systemPromptTemplate, "【关键词】") {
		t.Error("systemPromptTemplate should specify 【关键词】")
	}
}

// ---- Summarize 内容截断测试 (mock OpenAI) ----

// mockOpenAIChatServer 模拟 OpenAI chat/completions 接口，返回固定摘要。
func mockOpenAIChatServer(t *testing.T, reply string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "chat/completions") {
			http.NotFound(w, r)
			return
		}
		resp := map[string]interface{}{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"choices": []map[string]interface{}{{"index": 0, "message": map[string]string{"role": "assistant", "content": reply}, "finish_reason": "stop"}},
			"usage":   map[string]int{"prompt_tokens": 10, "completion_tokens": 20, "total_tokens": 30},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestSummarize_ReturnsLLMReply(t *testing.T) {
	expectedSummary := "【文件】test.md\n【摘要】\n- 核心要点\n【关键词】Go、修行"
	srv := mockOpenAIChatServer(t, expectedSummary)
	defer srv.Close()

	a, err := New(context.Background(), Config{
		APIKey:  "fake-key",
		Model:   "gpt-4o-mini",
		BaseURL: srv.URL + "/v1",
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	got, err := a.Summarize(context.Background(), "/path/to/test.md", "这是笔记内容")
	if err != nil {
		t.Fatalf("Summarize error: %v", err)
	}
	if got != expectedSummary {
		t.Errorf("Summarize = %q, want %q", got, expectedSummary)
	}
}

func TestSummarize_ContentTruncation(t *testing.T) {
	// 验证超过 6000 字节的内容会被截断后正常调用 LLM（而非崩溃）
	var callContent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		// 从请求体中抓取 user message 内容
		if msgs, ok := body["messages"].([]interface{}); ok {
			for _, m := range msgs {
				if mm, ok := m.(map[string]interface{}); ok {
					if mm["role"] == "user" {
						callContent, _ = mm["content"].(string)
					}
				}
			}
		}
		resp := map[string]interface{}{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"choices": []map[string]interface{}{{"index": 0, "message": map[string]string{"role": "assistant", "content": "摘要"}, "finish_reason": "stop"}},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	a, err := New(context.Background(), Config{
		APIKey:  "key",
		Model:   "gpt-4o-mini",
		BaseURL: srv.URL + "/v1",
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	// 7000 字节的长内容
	longContent := strings.Repeat("修行笔记", 1750) // 4 * 1750 = 7000 字节
	_, err = a.Summarize(context.Background(), "/path/note.md", longContent)
	if err != nil {
		t.Fatalf("Summarize should not error even for long content: %v", err)
	}

	// 验证传给 LLM 的内容包含截断标志
	if !strings.Contains(callContent, "已截断") {
		t.Error("expected truncation marker in LLM prompt for oversized content")
	}
}
