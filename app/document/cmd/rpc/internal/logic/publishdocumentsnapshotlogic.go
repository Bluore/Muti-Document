package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/pb"
	"github.com/Bluore/mutidoc/app/document/model"
	"github.com/Bluore/mutidoc/app/link/cmd/rpc/link"
	"github.com/Bluore/mutidoc/pkg/globalkey"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/idgenerate"
	"github.com/pkg/errors"

	"github.com/zeromicro/go-zero/core/logx"
)

type PublishDocumentSnapshotLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPublishDocumentSnapshotLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishDocumentSnapshotLogic {
	return &PublishDocumentSnapshotLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 发布文档快照, 返回快照短链
func (l *PublishDocumentSnapshotLogic) PublishDocumentSnapshot(in *pb.PublishDocumentSnapshotReq) (*pb.PublishDocumentSnapshotResp, error) {
	if in.DocumentId == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	detail, err := l.svcCtx.DocumentModel.FindOne(l.ctx, in.DocumentId)
	if err != nil && err != model.ErrNotFound {
		logx.WithContext(l.ctx).Errorf("FindOne fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db query error: %v", err)
	}
	if detail == nil {
		return nil, grpcerr.DOCUMENT_NOT_EXIST_ERROR
	}

	// 快照取整合后的正文与标题, 与协作文档当前内容保持一致
	content, summary, version := mergeDocumentCache(l.ctx, l.svcCtx, detail)

	snapshot := &model.PublicDocuments{
		Id:      idgenerate.NewID(l.svcCtx.Config.ServerID),
		OwnerId: detail.OwnerId,
		Summary: summary,
		Content: content,
		Version: version,
	}
	if _, err = l.svcCtx.PublicDocumentsModel.Insert(l.ctx, nil, snapshot); err != nil {
		logx.WithContext(l.ctx).Errorf("insert public document fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db insert error: %v", err)
	}

	// 短链指向前端公开文档页
	shortLink, err := l.svcCtx.LinkRpc.CreateShortLink(l.ctx, &link.CreateShortLinkReq{
		TargetUrl: globalkey.UrlPublicDocument(snapshot.Id),
	})
	if err != nil {
		logx.WithContext(l.ctx).Errorw("create short link fail",
			logx.Field("error", err),
			logx.Field("snapshot_id", snapshot.Id),
			logx.Field("document_id", in.DocumentId),
		)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "create short link error: %v", err)
	}

	logx.WithContext(l.ctx).Infow("publish document snapshot success",
		logx.Field("snapshot_id", snapshot.Id),
		logx.Field("document_id", in.DocumentId),
		logx.Field("owner_id", snapshot.OwnerId),
	)

	return &pb.PublishDocumentSnapshotResp{
		SnapshotId: snapshot.Id,
		ShortLink:  shortLink.ShortLink,
	}, nil
}
