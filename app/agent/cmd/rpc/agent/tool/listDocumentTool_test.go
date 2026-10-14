package tool

import (
	"context"
	"strings"
	"testing"

	"github.com/Bluore/mutidoc/app/document/cmd/rpc/document"
	"github.com/Bluore/mutidoc/pkg/ctxdata"

	"google.golang.org/grpc"
)

// fakeListDocumentRpc 按页返回预置文档, 其余方法保持 nil
type fakeListDocumentRpc struct {
	document.DocumentZrpcClient
	pages map[int64][]*document.Document
	total int64
	calls int
}

func (f *fakeListDocumentRpc) GetDocumentList(_ context.Context, in *document.GetDocumentListReq, _ ...grpc.CallOption) (*document.GetDocumentListResp, error) {
	f.calls++
	return &document.GetDocumentListResp{DocumentInfo: f.pages[in.Page], Total: f.total}, nil
}

func TestNewListDocumentToolBuilds(t *testing.T) {
	if _, err := NewListDocumentTool(&fakeListDocumentRpc{}); err != nil {
		t.Fatalf("build tool: %v", err)
	}
}

func TestListDocumentRequiresUserInCtx(t *testing.T) {
	if _, err := listDocuments(context.Background(), &fakeListDocumentRpc{}); err == nil {
		t.Fatal("expected error when user id is missing from ctx")
	}
}

func TestListDocumentEmpty(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxdata.CtxKewJwtUserId, "u1")

	out, err := listDocuments(ctx, &fakeListDocumentRpc{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "还没有创建") {
		t.Fatalf("output %q should mention empty list", out)
	}
}

func TestListDocumentPaginates(t *testing.T) {
	rpc := &fakeListDocumentRpc{
		pages: map[int64][]*document.Document{
			1: {{Id: "d1", Summary: "第一篇"}, {Id: "d2", Summary: "第二篇"}},
			2: {{Id: "d3", Summary: "第三篇"}},
		},
		total: 3,
	}
	ctx := context.WithValue(context.Background(), ctxdata.CtxKewJwtUserId, "u1")

	out, err := listDocuments(ctx, rpc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"d1", "d2", "d3", "第三篇"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q missing %q", out, want)
		}
	}
	if rpc.calls != 2 {
		t.Fatalf("expected 2 page calls, got %d", rpc.calls)
	}
}
