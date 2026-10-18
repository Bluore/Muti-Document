// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/httperr"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/document"
	"github.com/Bluore/mutidoc/pkg/ctxdata"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/response"

	"github.com/zeromicro/go-zero/core/logx"
)

type PublishDocumentSnapshotLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// publish document snapshot
func NewPublishDocumentSnapshotLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishDocumentSnapshotLogic {
	return &PublishDocumentSnapshotLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PublishDocumentSnapshotLogic) PublishDocumentSnapshot(req *types.PublishDocumentSnapshotReq) (resp *types.PublishDocumentSnapshotResp, err error) {
	// 操作者从jwt中获取
	ownerID := ctxdata.MustGetUserIdFromCtx(l.ctx)
	if req.DocumentId == "" {
		return nil, response.ErrValidation
	}

	// 校验文档归属, 只允许归属者发布快照
	info, err := l.svcCtx.DocumentRpc.GetDocumentInfo(l.ctx, &document.GetDocumentInfoReq{
		DocumentId: req.DocumentId,
	})
	if err != nil {
		if grpcerr.DOCUMENT_NOT_EXIST_ERROR.EqualError(err) {
			return nil, httperr.ErrDocumentNotExist
		}
		logx.WithContext(l.ctx).Errorf("get document info fail: %v", err)
		return nil, response.ErrServer
	}
	if info.DocumentInfo == nil {
		return nil, httperr.ErrDocumentNotExist
	}
	if info.DocumentInfo.OwnerId != ownerID {
		logx.WithContext(l.ctx).Errorw("publish document snapshot permission denied",
			logx.Field("document_id", req.DocumentId),
			logx.Field("owner_id", info.DocumentInfo.OwnerId),
			logx.Field("operator_id", ownerID),
		)
		return nil, httperr.ErrDocumentNoPermission
	}

	// 快照与短链均由 document-rpc 创建
	snapshot, err := l.svcCtx.DocumentRpc.PublishDocumentSnapshot(l.ctx, &document.PublishDocumentSnapshotReq{
		DocumentId: req.DocumentId,
	})
	if err != nil {
		if grpcerr.DOCUMENT_NOT_EXIST_ERROR.EqualError(err) {
			return nil, httperr.ErrDocumentNotExist
		}
		logx.WithContext(l.ctx).Errorf("publish document snapshot fail: %v", err)
		return nil, response.ErrServer
	}
	logx.WithContext(l.ctx).Infow("publish document snapshot success",
		logx.Field("document_id", req.DocumentId),
		logx.Field("snapshot_id", snapshot.SnapshotId),
		logx.Field("owner_id", ownerID),
	)

	return &types.PublishDocumentSnapshotResp{
		SnapshotId: snapshot.SnapshotId,
		ShortLink:  snapshot.ShortLink,
	}, nil
}
