package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/pb"
	"github.com/Bluore/mutidoc/pkg/response"

	"github.com/zeromicro/go-zero/core/logx"
)

type StopDocumentSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStopDocumentSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StopDocumentSessionLogic {
	return &StopDocumentSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// StopDocumentSession 关闭并移除文档的内存编辑会话, 用于文档被删除等场景
func (l *StopDocumentSessionLogic) StopDocumentSession(in *pb.StopDocumentSessionReq) (*pb.StopDocumentSessionResp, error) {
	if in.DocumentId == "" {
		return nil, response.ErrValidation
	}

	documentClient := l.svcCtx.Document.Get(in.DocumentId)
	if documentClient != nil {
		documentClient.Close()
		l.svcCtx.Document.CheckAndDelete(in.DocumentId, documentClient)
	}

	logx.WithContext(l.ctx).Infow("stop document session",
		logx.Field("document_id", in.DocumentId),
		logx.Field("session_exist", documentClient != nil),
	)

	return &pb.StopDocumentSessionResp{}, nil
}
