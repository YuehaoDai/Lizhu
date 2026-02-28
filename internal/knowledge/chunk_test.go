package knowledge

import (
	"context"
	"strings"
	"testing"
)

// ---- chunkText 测试 ----

func TestChunkText_Markdown_ShortParagraphs(t *testing.T) {
	content := "第一段内容\n\n第二段内容\n\n第三段内容"
	chunks := chunkText(content, ".md")
	if len(chunks) != 3 {
		t.Fatalf("want 3 chunks, got %d: %v", len(chunks), chunks)
	}
	if chunks[0] != "第一段内容" {
		t.Errorf("chunk[0] = %q", chunks[0])
	}
}

func TestChunkText_Markdown_EmptyParagraphsSkipped(t *testing.T) {
	content := "段落一\n\n\n\n段落二\n\n   \n\n段落三"
	chunks := chunkText(content, ".md")
	if len(chunks) != 3 {
		t.Fatalf("want 3 chunks (empty paragraphs skipped), got %d: %v", len(chunks), chunks)
	}
}

func TestChunkText_Markdown_LongParagraphSplitByLine(t *testing.T) {
	// 构造一个超过 512 字节的段落（用行分隔）
	longLine := strings.Repeat("x", 300)
	content := longLine + "\n" + longLine // 两行，合计 601 字节，超过 512

	chunks := chunkText(content, ".md")
	// 超长段落应按行再分
	if len(chunks) < 2 {
		t.Errorf("long paragraph should be split by line, got %d chunk(s)", len(chunks))
	}
}

func TestChunkText_TxtExtension(t *testing.T) {
	content := "第一段\n\n第二段"
	chunks := chunkText(content, ".txt")
	if len(chunks) != 2 {
		t.Fatalf("want 2 chunks for .txt, got %d", len(chunks))
	}
}

func TestChunkText_GoCode_SlidingWindow(t *testing.T) {
	// 构造超过 512 字节的 Go 代码，验证滑动窗口分块
	code := strings.Repeat("a", 1024)
	chunks := chunkText(code, ".go")

	// 期望至少 2 个块（1024 / (512/2) = 4 步，但每块最大 512 字节）
	if len(chunks) < 2 {
		t.Errorf("expected multiple chunks for 1024-byte code, got %d", len(chunks))
	}
	// 每块不超过 512 字节
	for i, c := range chunks {
		if len(c) > 512 {
			t.Errorf("chunk[%d] length %d exceeds 512", i, len(c))
		}
	}
}

func TestChunkText_EmptyContent(t *testing.T) {
	chunks := chunkText("", ".md")
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty content, got %d", len(chunks))
	}
}

func TestChunkText_GoCode_ShortContent(t *testing.T) {
	code := "fmt.Println(\"hello\")"
	chunks := chunkText(code, ".go")
	if len(chunks) != 1 {
		t.Fatalf("want 1 chunk for short code, got %d", len(chunks))
	}
	if chunks[0] != code {
		t.Errorf("chunk = %q, want %q", chunks[0], code)
	}
}

// ---- Ingester 无 Milvus 时的 stub 行为 ----

func TestIngester_Disabled_ReturnsNil(t *testing.T) {
	ing := New(Config{Enabled: false})
	result, err := ing.IngestFile(context.TODO(), "/tmp/nonexistent.md")
	if err != nil {
		t.Fatalf("disabled ingester should not error, got: %v", err)
	}
	if result != nil {
		t.Errorf("disabled ingester should return nil result, got: %+v", result)
	}
}

func TestIngester_DeleteByFilePath_Disabled_NoError(t *testing.T) {
	ing := New(Config{Enabled: false})
	if err := ing.DeleteByFilePath(context.TODO(), "/tmp/file.md"); err != nil {
		t.Errorf("disabled ingester DeleteByFilePath should not error: %v", err)
	}
}

// ---- Retriever 无 Milvus 时的 stub 行为 ----

func TestRetriever_Disabled_ReturnsEmpty(t *testing.T) {
	r := NewRetriever(Config{Enabled: false})
	results, err := r.Search(context.TODO(), "some query", 3)
	if err != nil {
		t.Fatalf("disabled retriever should not error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("disabled retriever should return empty results, got %d", len(results))
	}
}
