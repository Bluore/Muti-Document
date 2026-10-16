package tool

import (
	"context"
	"strings"
	"testing"

	"github.com/Bluore/mutidoc/app/document/cmd/rpc/document"

	"google.golang.org/grpc"
)

// fakeSearchDocumentRpc 只覆写检索相关方法, 其余方法保持 nil
type fakeSearchDocumentRpc struct {
	document.DocumentZrpcClient
	hasPermission bool
	permissionErr error

	searchCalled bool
	lastReq      *document.SearchDocumentReq
	result       string
	searchErr    error
}

func (f *fakeSearchDocumentRpc) GetDocumentPermission(_ context.Context, _ *document.GetDocumentPermissionReq, _ ...grpc.CallOption) (*document.GetDocumentPermissionResp, error) {
	if f.permissionErr != nil {
		return nil, f.permissionErr
	}
	return &document.GetDocumentPermissionResp{HasPermission: f.hasPermission}, nil
}

func (f *fakeSearchDocumentRpc) SearchDocument(_ context.Context, in *document.SearchDocumentReq, _ ...grpc.CallOption) (*document.SearchDocumentResp, error) {
	f.searchCalled = true
	f.lastReq = in
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	return &document.SearchDocumentResp{Result: f.result}, nil
}

func TestNewSearchDocumentToolBuilds(t *testing.T) {
	if _, err := NewSearchDocumentTool(&fakeSearchDocumentRpc{}); err != nil {
		t.Fatalf("build tool: %v", err)
	}
}

func TestSearchDocumentRequiresUserInCtx(t *testing.T) {
	if _, err := searchDocument(context.Background(), &fakeSearchDocumentRpc{}, searchDocumentReq{Query: "q"}); err == nil {
		t.Fatal("expected error when user id is missing from ctx")
	}
}

func TestSearchDocumentRequiresQuery(t *testing.T) {
	rpc := &fakeSearchDocumentRpc{}
	if _, err := searchDocument(userCtx(), rpc, searchDocumentReq{Query: "   "}); err == nil {
		t.Fatal("expected error when query is empty")
	}
	if rpc.searchCalled {
		t.Fatal("search should not be called without a query")
	}
}

// 未指定文档时按归属过滤, 只检索当前用户自己的文档
func TestSearchDocumentScopesToOwner(t *testing.T) {
	rpc := &fakeSearchDocumentRpc{result: `[{"doc_id":"d1","chunk_index":0,"text":"向量检索原文","score":0.87}]`}

	out, err := searchDocument(userCtx(), rpc, searchDocumentReq{Query: " 检索 "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rpc.lastReq.GetOwnerId() != "u1" {
		t.Fatalf("expected owner filter u1, got %q", rpc.lastReq.GetOwnerId())
	}
	if rpc.lastReq.GetDocumentId() != "" {
		t.Fatalf("expected no document filter, got %q", rpc.lastReq.GetDocumentId())
	}
	if rpc.lastReq.GetText() != "检索" {
		t.Fatalf("expected trimmed query, got %q", rpc.lastReq.GetText())
	}
	if rpc.lastReq.GetTopK() != searchDocumentDefaultTopK {
		t.Fatalf("expected default top k %d, got %d", searchDocumentDefaultTopK, rpc.lastReq.GetTopK())
	}
	for _, want := range []string{"d1", "向量检索原文"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q missing %q", out, want)
		}
	}
}

// 指定文档时先校验权限, 无权限不检索
func TestSearchDocumentDeniesWithoutPermission(t *testing.T) {
	rpc := &fakeSearchDocumentRpc{hasPermission: false}

	if _, err := searchDocument(userCtx(), rpc, searchDocumentReq{Query: "q", DocumentID: "d1"}); err == nil {
		t.Fatal("expected error when user has no permission")
	}
	if rpc.searchCalled {
		t.Fatal("search should not be called without permission")
	}
}

// 有权限时不再按归属过滤, 兼容别人共享给该用户的文档
func TestSearchDocumentWithPermissionSkipsOwnerFilter(t *testing.T) {
	rpc := &fakeSearchDocumentRpc{hasPermission: true, result: "[]"}

	if _, err := searchDocument(userCtx(), rpc, searchDocumentReq{Query: "q", DocumentID: "d1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rpc.lastReq.GetDocumentId() != "d1" {
		t.Fatalf("expected document filter d1, got %q", rpc.lastReq.GetDocumentId())
	}
	if rpc.lastReq.GetOwnerId() != "" {
		t.Fatalf("expected no owner filter, got %q", rpc.lastReq.GetOwnerId())
	}
}

func TestSearchDocumentClampsTopK(t *testing.T) {
	cases := []struct {
		in   int64
		want int64
	}{
		{0, searchDocumentDefaultTopK},
		{-1, searchDocumentDefaultTopK},
		{5, 5},
		{searchDocumentMaxTopK + 1, searchDocumentMaxTopK},
	}
	for _, c := range cases {
		rpc := &fakeSearchDocumentRpc{result: "[]"}
		if _, err := searchDocument(userCtx(), rpc, searchDocumentReq{Query: "q", TopK: c.in}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rpc.lastReq.GetTopK() != c.want {
			t.Fatalf("top_k %d: expected %d, got %d", c.in, c.want, rpc.lastReq.GetTopK())
		}
	}
}

func TestSearchDocumentEmptyResult(t *testing.T) {
	rpc := &fakeSearchDocumentRpc{result: "[]"}

	out, err := searchDocument(userCtx(), rpc, searchDocumentReq{Query: "q"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "没有检索到") {
		t.Fatalf("output %q should mention empty result", out)
	}
}

// 检索结果不是合法 JSON 时按检索失败处理
func TestSearchDocumentInvalidResult(t *testing.T) {
	rpc := &fakeSearchDocumentRpc{result: "not json"}

	if _, err := searchDocument(userCtx(), rpc, searchDocumentReq{Query: "q"}); err == nil {
		t.Fatal("expected error when result is not valid json")
	}
}
