package tool

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/tool/toolcallresp"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/edit"
	"github.com/Bluore/mutidoc/pkg/ctxdata"
	"github.com/Bluore/mutidoc/pkg/grpcerr"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeEditRpc 只覆写测试用到的方法, 其余方法保持 nil
type fakeEditRpc struct {
	edit.Edit
	createCalled bool
	createReq    *edit.CreateBotReq
	getResp      *edit.GetResp
	getErr       error
	replaceErr   error
	insertErr    error
}

func (f *fakeEditRpc) CreateBot(_ context.Context, in *edit.CreateBotReq, _ ...grpc.CallOption) (*edit.CreateBotResp, error) {
	f.createCalled = true
	f.createReq = in
	return &edit.CreateBotResp{}, nil
}

func (f *fakeEditRpc) Get(_ context.Context, _ *edit.GetReq, _ ...grpc.CallOption) (*edit.GetResp, error) {
	return f.getResp, f.getErr
}

func (f *fakeEditRpc) Replace(_ context.Context, _ *edit.ReplaceReq, _ ...grpc.CallOption) (*edit.ReplaceResp, error) {
	return &edit.ReplaceResp{}, f.replaceErr
}

func (f *fakeEditRpc) Insert(_ context.Context, _ *edit.InsertReq, _ ...grpc.CallOption) (*edit.InsertResp, error) {
	return &edit.InsertResp{}, f.insertErr
}

func userCtx() context.Context {
	return context.WithValue(context.Background(), ctxdata.CtxKewJwtUserId, "u1")
}

func TestCreateDocumentConnectionRequiresParams(t *testing.T) {
	if _, err := createDocumentConnection(userCtx(), &fakeEditRpc{}, createDocumentConnectionReq{}); err == nil {
		t.Fatal("expected error when document_id is missing")
	}
	if _, err := createDocumentConnection(context.Background(), &fakeEditRpc{}, createDocumentConnectionReq{DocumentID: "d1"}); err == nil {
		t.Fatal("expected error when user is missing from ctx")
	}
}

func TestCreateDocumentConnectionDerivesBotID(t *testing.T) {
	rpc := &fakeEditRpc{}
	out, err := createDocumentConnection(userCtx(), rpc, createDocumentConnectionReq{DocumentID: "d1", BotName: "助手"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rpc.createCalled {
		t.Fatal("CreateBot should be called")
	}
	if rpc.createReq.GetDocumentId() != "d1" || rpc.createReq.GetUserId() != "u1" {
		t.Fatalf("create req = %+v", rpc.createReq)
	}
	if rpc.createReq.GetBotId() != botIDFor("u1", "d1") {
		t.Fatalf("bot id = %q, want %q", rpc.createReq.GetBotId(), botIDFor("u1", "d1"))
	}
	if !strings.Contains(out, "d1") {
		t.Fatalf("output %q should mention document id", out)
	}
}

func TestCreateDocumentConnectionDefaultsBotName(t *testing.T) {
	rpc := &fakeEditRpc{}
	if _, err := createDocumentConnection(userCtx(), rpc, createDocumentConnectionReq{DocumentID: "d1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rpc.createReq.GetBotName() == "" {
		t.Fatal("bot name should default to a non-empty value")
	}
}

func TestGetDocumentConnectionReturnsContent(t *testing.T) {
	rpc := &fakeEditRpc{getResp: &edit.GetResp{Content: "hello world"}}
	out, err := getDocumentConnection(userCtx(), rpc, getDocumentConnectionReq{DocumentID: "d1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "hello world") {
		t.Fatalf("output %q should contain content", out)
	}
}

func TestGetDocumentConnectionMapsBotNotExist(t *testing.T) {
	rpc := &fakeEditRpc{getErr: status.Error(codes.Code(grpcerr.BOT_NOT_EXIST_ERROR), "bot not exist")}
	_, err := getDocumentConnection(userCtx(), rpc, getDocumentConnectionReq{DocumentID: "d1"})
	if !errors.Is(err, toolcallresp.ErrBotConnectionNotExist) {
		t.Fatalf("err = %v, want ErrBotConnectionNotExist", err)
	}
}

func TestReplaceDocumentConnectionRequiresOriginal(t *testing.T) {
	if _, err := replaceDocumentConnection(userCtx(), &fakeEditRpc{}, replaceDocumentConnectionReq{DocumentID: "d1"}); !errors.Is(err, toolcallresp.ErrEditTextRequired) {
		t.Fatalf("err = %v, want ErrEditTextRequired", err)
	}
}

func TestReplaceDocumentConnectionMapsTextNotFound(t *testing.T) {
	rpc := &fakeEditRpc{replaceErr: status.Error(codes.Code(grpcerr.BOT_TEXT_NOT_FOUND_ERROR), "not found")}
	_, err := replaceDocumentConnection(userCtx(), rpc, replaceDocumentConnectionReq{DocumentID: "d1", Original: "a", Replacement: "b"})
	if !errors.Is(err, toolcallresp.ErrBotTextNotFound) {
		t.Fatalf("err = %v, want ErrBotTextNotFound", err)
	}
}

func TestInsertDocumentConnectionRequiresInsertion(t *testing.T) {
	if _, err := insertDocumentConnection(userCtx(), &fakeEditRpc{}, insertDocumentConnectionReq{DocumentID: "d1"}); !errors.Is(err, toolcallresp.ErrEditTextRequired) {
		t.Fatalf("err = %v, want ErrEditTextRequired", err)
	}
}

func TestInsertDocumentConnectionOK(t *testing.T) {
	out, err := insertDocumentConnection(userCtx(), &fakeEditRpc{}, insertDocumentConnectionReq{DocumentID: "d1", Anchor: "hello", Insertion: " world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "插入") {
		t.Fatalf("output %q should confirm insertion", out)
	}
}
