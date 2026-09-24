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

type GetDocumentInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDocumentInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDocumentInfoLogic {
	return &GetDocumentInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取文档详情
func (l *GetDocumentInfoLogic) GetDocumentInfo(in *pb.GetDocumentInfoReq) (*pb.GetDocumentInfoResp, error) {
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

	// 只返回元信息, 文档正文由协作连接单独获取
	return &pb.GetDocumentInfoResp{
		DocumentInfo: &pb.Document{
			Id:       documentModel.Id,
			OwnerId:  documentModel.OwnerId,
			Summary:  documentModel.Summary,
			Version:  documentModel.Version,
			CreateAt: documentModel.CreateAt.Unix(),
			UpdateAt: documentModel.UpdateAt.Unix(),
			DeleteAt: documentModel.DeleteAt.Unix(),
		},
	}, nil
}
