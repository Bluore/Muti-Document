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

type GetDocumentPermissionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDocumentPermissionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDocumentPermissionLogic {
	return &GetDocumentPermissionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取用户是否有授权
func (l *GetDocumentPermissionLogic) GetDocumentPermission(in *pb.GetDocumentPermissionReq) (*pb.GetDocumentPermissionResp, error) {
	if in.UserId == "" || in.DocumentId == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	permission, err := l.svcCtx.DocumentPermissionModel.FindOneByUserIdAndDocumentId(l.ctx, in.UserId, in.DocumentId)
	if err != nil && err != model.ErrNotFound {
		logx.WithContext(l.ctx).Errorf("FindOneByUserIdAndDocumentId fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db query error: %v", err)
	}
	if permission != nil {
		return &pb.GetDocumentPermissionResp{HasPermission: true}, nil
	}

	// 没有授权记录时, 文档拥有者视为有权限
	document, err := l.svcCtx.DocumentModel.FindOne(l.ctx, in.DocumentId)
	if err != nil && err != model.ErrNotFound {
		logx.WithContext(l.ctx).Errorf("FindOne document fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db query error: %v", err)
	}
	if document == nil {
		return nil, grpcerr.DOCUMENT_NOT_EXIST_ERROR
	}

	return &pb.GetDocumentPermissionResp{
		HasPermission: document.OwnerId == in.UserId,
	}, nil
}
