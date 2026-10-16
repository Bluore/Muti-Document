package model

import (
	"strings"
	"testing"
)

func TestSplitDocumentChunksEmpty(t *testing.T) {
	if chunks := SplitDocumentChunks("  \n\t ", 10, 2); chunks != nil {
		t.Fatalf("expected nil, got %v", chunks)
	}
}

func TestSplitDocumentChunksSingle(t *testing.T) {
	text := "短文本不应该被切分"
	chunks := SplitDocumentChunks(text, 100, 10)
	if len(chunks) != 1 || chunks[0] != text {
		t.Fatalf("expected one chunk %q, got %v", text, chunks)
	}
}

func TestSplitDocumentChunksFallbackConf(t *testing.T) {
	text := strings.Repeat("a", DefaultChunkSize)
	chunks := SplitDocumentChunks(text, 0, 0)
	if len(chunks) != 1 || chunks[0] != text {
		t.Fatalf("expected single chunk of default size, got %d chunks", len(chunks))
	}
}

// 相邻分块要共享边界内容, 且顺序拼接(去掉重叠)后能还原原文
func TestSplitDocumentChunksOverlap(t *testing.T) {
	var builder strings.Builder
	for i := 0; i < 100; i++ {
		builder.WriteRune(rune('a' + i%26))
	}
	text := builder.String()

	const size, overlap = 30, 10
	chunks := SplitDocumentChunks(text, size, overlap)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	var rebuilt []rune
	for i, chunk := range chunks {
		runes := []rune(chunk)
		if len(runes) > size {
			t.Fatalf("chunk %d has %d runes, want <= %d", i, len(runes), size)
		}

		if i == 0 {
			rebuilt = append(rebuilt, runes...)
			continue
		}

		prev := []rune(chunks[i-1])
		tail := prev[len(prev)-overlap:]
		if !strings.HasPrefix(chunk, string(tail)) {
			t.Fatalf("chunk %d does not start with the tail of chunk %d:\nchunk %d = %q\nchunk %d = %q",
				i, i-1, i-1, chunks[i-1], i, chunk)
		}
		rebuilt = append(rebuilt, runes[overlap:]...)
	}

	if string(rebuilt) != text {
		t.Fatalf("rebuilt text mismatch:\nwant %q\n got %q", text, string(rebuilt))
	}
}

func TestSplitDocumentChunksOverlapNotLargerThanSize(t *testing.T) {
	text := strings.Repeat("ab", 50)
	chunks := SplitDocumentChunks(text, 10, 99)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	for i, chunk := range chunks {
		if runes := []rune(chunk); len(runes) > 10 {
			t.Fatalf("chunk %d has %d runes, want <= 10", i, len(runes))
		}
	}
}

func TestDocumentChunkPointIDStable(t *testing.T) {
	first := documentChunkPointID("doc-1", 2)
	if first != documentChunkPointID("doc-1", 2) {
		t.Fatal("point id must be stable for the same document and index")
	}
	if first == documentChunkPointID("doc-1", 3) {
		t.Fatal("point id must differ across chunk indexes")
	}
	if first == documentChunkPointID("doc-2", 2) {
		t.Fatal("point id must differ across documents")
	}
}
