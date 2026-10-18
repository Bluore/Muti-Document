// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/httperr"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/document"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/response"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSnapshotDocumentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// get snapshot document content
func NewGetSnapshotDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSnapshotDocumentLogic {
	return &GetSnapshotDocumentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 公开接口, 无需鉴权, 直接按快照id读取内容
func (l *GetSnapshotDocumentLogic) GetSnapshotDocument(req *types.GetSnapshotDocumentReq) (resp *types.GetSnapshotDocumentResp, err error) {
	if req.SnapshotId == "" {
		return nil, response.ErrValidation
	}

	snapshot, err := l.svcCtx.DocumentRpc.GetSnapshotDocument(l.ctx, &document.GetSnapshotDocumentReq{
		SnapshotId: req.SnapshotId,
	})
	if err != nil {
		if grpcerr.DOCUMENT_SNAPSHOT_NOT_EXIST_ERROR.EqualError(err) {
			return nil, httperr.ErrSnapshotNotExist
		}
		logx.WithContext(l.ctx).Errorf("get snapshot document fail: %v", err)
		return nil, response.ErrServer
	}

	return &types.GetSnapshotDocumentResp{
		Id:       snapshot.Id,
		OwnerId:  snapshot.OwnerId,
		Summary:  snapshot.Summary,
		Content:  snapshot.Content,
		Version:  snapshot.Version,
		CreateAt: snapshot.CreateAt,
		UpdateAt: snapshot.UpdateAt,
	}, nil
}
