package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/pb"
	"github.com/Bluore/mutidoc/app/document/model"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/pkg/errors"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteDocumentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteDocumentLogic {
	return &DeleteDocumentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 删除文档
func (l *DeleteDocumentLogic) DeleteDocument(in *pb.DeleteDocumentReq) (*pb.DeleteDocumentResp, error) {
	if in.DocumentId == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	documentModel, err := l.svcCtx.DocumentModel.FindOne(l.ctx, in.DocumentId)
	if err != nil && err != model.ErrNotFound {
		logx.WithContext(l.ctx).Errorf("FindOne fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db query error: %v", err)
	}
	if documentModel == nil {
		return nil, grpcerr.DOCUMENT_NOT_EXIST_ERROR
	}

	// 软删除, 同时刷新版本号
	if err = l.svcCtx.DocumentModel.DeleteSoft(l.ctx, nil, documentModel); err != nil {
		logx.WithContext(l.ctx).Errorf("DeleteSoft fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db delete error: %v", err)
	}
	logx.WithContext(l.ctx).Infow("delete document success", logx.Field("document_id", in.DocumentId))

	return &pb.DeleteDocumentResp{}, nil
}
