package tool

import (
	"context"
	"strings"
	"testing"

	"github.com/Bluore/mutidoc/app/document/cmd/rpc/document"
	"github.com/Bluore/mutidoc/pkg/ctxdata"

	"google.golang.org/grpc"
)

// fakeDocumentRpc 只覆写测试用到的方法, 其余方法保持 nil
type fakeDocumentRpc struct {
	document.DocumentZrpcClient
	hasPermission bool
	contentCalled bool
	content       string
}

func (f *fakeDocumentRpc) GetDocumentPermission(_ context.Context, _ *document.GetDocumentPermissionReq, _ ...grpc.CallOption) (*document.GetDocumentPermissionResp, error) {
	return &document.GetDocumentPermissionResp{HasPermission: f.hasPermission}, nil
}

func (f *fakeDocumentRpc) GetDocumentContent(_ context.Context, _ *document.GetDocumentContentReq, _ ...grpc.CallOption) (*document.GetDocumentContentResp, error) {
	f.contentCalled = true
	return &document.GetDocumentContentResp{Content: f.content, Summary: "s", Version: 1}, nil
}

func TestQueryDocumentRequiresUserInCtx(t *testing.T) {
	_, err := queryDocument(context.Background(), &fakeDocumentRpc{}, queryDocumentReq{DocumentID: "d1"})
	if err == nil {
		t.Fatal("expected error when user id is missing from ctx")
	}
}

func TestQueryDocumentDeniesWithoutPermission(t *testing.T) {
	rpc := &fakeDocumentRpc{hasPermission: false}
	ctx := context.WithValue(context.Background(), ctxdata.CtxKewJwtUserId, "u1")

	_, err := queryDocument(ctx, rpc, queryDocumentReq{DocumentID: "d1"})
	if err == nil {
		t.Fatal("expected error when user has no permission")
	}
	if rpc.contentCalled {
		t.Fatal("content should not be fetched without permission")
	}
}

func TestQueryDocumentReturnsContentWithPermission(t *testing.T) {
	rpc := &fakeDocumentRpc{hasPermission: true, content: "hello"}
	ctx := context.WithValue(context.Background(), ctxdata.CtxKewJwtUserId, "u1")

	out, err := queryDocument(ctx, rpc, queryDocumentReq{DocumentID: "d1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("output %q should contain document content", out)
	}
	if !rpc.contentCalled {
		t.Fatal("content should be fetched when permission is granted")
	}
}
