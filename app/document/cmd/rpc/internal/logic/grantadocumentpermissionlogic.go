package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/pb"
	"github.com/Bluore/mutidoc/app/document/model"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/idgenerate"
	"github.com/pkg/errors"

	"github.com/zeromicro/go-zero/core/logx"
)

type GrantaDocumentPermissionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGrantaDocumentPermissionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GrantaDocumentPermissionLogic {
	return &GrantaDocumentPermissionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 授权用户文档权限
func (l *GrantaDocumentPermissionLogic) GrantaDocumentPermission(in *pb.GrantDocumentPermissionReq) (*pb.GrantDocumentPermissionResp, error) {
	if in.UserId == "" || in.DocumentId == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	document, err := l.svcCtx.DocumentModel.FindOne(l.ctx, in.DocumentId)
	if err != nil && err != model.ErrNotFound {
		logx.WithContext(l.ctx).Errorf("FindOne document fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db query error: %v", err)
	}
	if document == nil {
		return nil, grpcerr.DOCUMENT_NOT_EXIST_ERROR
	}

	// 拥有者本身已具备权限, 被授权用户不能是其本人
	if document.OwnerId == in.UserId {
		logx.WithContext(l.ctx).Errorf("grant to owner is not allowed, document_id: %s, user_id: %s", in.DocumentId, in.UserId)
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	if _, err := l.svcCtx.DocumentPermissionModel.Insert(l.ctx, nil, &model.GrantDocumentsPermission{
		PermissionId: idgenerate.NewID(l.svcCtx.Config.ServerID),
		UserId:       in.UserId,
		DocumentId:   in.DocumentId,
		Version:      1,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("insert document permission fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db insert error: %v", err)
	}
	logx.WithContext(l.ctx).Infow("grant document permission success",
		logx.Field("document_id", in.DocumentId),
		logx.Field("user_id", in.UserId),
	)

	return &pb.GrantDocumentPermissionResp{}, nil
}
