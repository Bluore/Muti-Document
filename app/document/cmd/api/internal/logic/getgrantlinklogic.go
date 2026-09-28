// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/httperr"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/document"
	"github.com/Bluore/mutidoc/app/link/cmd/rpc/link"
	"github.com/Bluore/mutidoc/pkg/ctxdata"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/response"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetGrantLinkLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// get grant link
func NewGetGrantLinkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGrantLinkLogic {
	return &GetGrantLinkLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetGrantLinkLogic) GetGrantLink(req *types.GetGrantLinkReq) (resp *types.GetGrantLinkResp, err error) {
	// 归属者从jwt中获取
	ownerID := ctxdata.MustGetUserIdFromCtx(l.ctx)

	// 校验文档归属, 只允许归属者生成授权链接
	result, err := l.svcCtx.DocumentRpc.GetDocumentInfo(l.ctx, &document.GetDocumentInfoReq{
		DocumentId: req.DocumentId,
	})
	if err != nil {
		if grpcerr.DOCUMENT_NOT_EXIST_ERROR.EqualError(err) {
			return nil, httperr.ErrDocumentNotExist
		}
		logx.WithContext(l.ctx).Errorf("get document info fail: %v", err)
		return nil, response.ErrServer
	}
	if result.DocumentInfo == nil {
		return nil, httperr.ErrDocumentNotExist
	}
	if result.DocumentInfo.OwnerId != ownerID {
		logx.WithContext(l.ctx).Errorw("create grant link permission denied",
			logx.Field("document_id", req.DocumentId),
			logx.Field("owner_id", result.DocumentInfo.OwnerId),
			logx.Field("operator_id", ownerID),
		)
		return nil, httperr.ErrDocumentNoPermission
	}

	// 授权链接由 link-rpc 生成, 已存在则复用
	grant, err := l.svcCtx.LinkRpc.CreateGrantLink(l.ctx, &link.CreateGrantLinkReq{
		DocumentId: req.DocumentId,
		UserId:     ownerID,
	})
	if err != nil {
		logx.WithContext(l.ctx).Errorf("create grant link fail: %v", err)
		return nil, response.ErrServer
	}
	logx.WithContext(l.ctx).Infow("get grant link success",
		logx.Field("document_id", req.DocumentId),
		logx.Field("owner_id", ownerID),
	)

	return &types.GetGrantLinkResp{
		Link: grant.GrantLink,
	}, nil
}
