package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateDocumentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateDocumentLogic {
	return &UpdateDocumentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 更新文档(全量更新)
func (l *UpdateDocumentLogic) UpdateDocument(in *pb.UpdateDocumentReq) (*pb.UpdateDocumentResp, error) {
	// todo: add your logic here and delete this line

	return &pb.UpdateDocumentResp{}, nil
}
