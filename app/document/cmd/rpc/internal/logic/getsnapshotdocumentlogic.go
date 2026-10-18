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

type GetSnapshotDocumentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetSnapshotDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSnapshotDocumentLogic {
	return &GetSnapshotDocumentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取快照文档内容(无需鉴权)
func (l *GetSnapshotDocumentLogic) GetSnapshotDocument(in *pb.GetSnapshotDocumentReq) (*pb.GetSnapshotDocumentResp, error) {
	if in.SnapshotId == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	snapshot, err := l.svcCtx.PublicDocumentsModel.FindOne(l.ctx, in.SnapshotId)
	if err != nil && err != model.ErrNotFound {
		logx.WithContext(l.ctx).Errorf("FindOne public document fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db query error: %v", err)
	}
	if snapshot == nil {
		return nil, grpcerr.DOCUMENT_SNAPSHOT_NOT_EXIST_ERROR
	}

	return &pb.GetSnapshotDocumentResp{
		Id:       snapshot.Id,
		OwnerId:  snapshot.OwnerId,
		Summary:  snapshot.Summary,
		Content:  snapshot.Content,
		Version:  snapshot.Version,
		CreateAt: snapshot.CreateAt.Unix(),
		UpdateAt: snapshot.UpdateAt.Unix(),
	}, nil
}
