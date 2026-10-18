// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package grantLink

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/rpc/document"
	"github.com/Bluore/mutidoc/app/link/cmd/api/internal/httperr"
	"github.com/Bluore/mutidoc/app/link/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/link/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/app/link/cmd/rpc/link"
	"github.com/Bluore/mutidoc/pkg/ctxdata"
	"github.com/Bluore/mutidoc/pkg/globalkey"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/response"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetGrantLinkLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// grant document permission and open link
func NewGetGrantLinkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGrantLinkLogic {
	return &GetGrantLinkLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetGrantLinkLogic) GetGrantLink(req *types.GetGrantDocumentReq) (resp *types.GetGrantDocumentResp, err error) {
	// 被授权用户从 jwt 中获取
	userID := ctxdata.MustGetUserIdFromCtx(l.ctx)
	if req.GrantID == "" {
		return nil, response.ErrValidation
	}

	// rpc 只解析授权链接, 返回对应文档与所有者
	grant, err := l.svcCtx.LinkRpc.GetGrantLink(l.ctx, &link.GetGrantDocumentReq{
		GrantId: req.GrantID,
	})
	if err != nil {
		if grpcerr.LINK_NOT_EXIST_ERROR.EqualError(err) {
			return nil, httperr.ErrLinkNotExist
		}
		if grpcerr.LINK_EXPIRED_ERROR.EqualError(err) {
			return nil, httperr.ErrLinkExpired
		}
		logx.WithContext(l.ctx).Errorf("get grant link fail: %v", err)
		return nil, response.ErrServer
	}

	// 已授权直接返回, 未授权则先为用户授予文档权限
	permission, err := l.svcCtx.DocumentRpc.GetDocumentPermission(l.ctx, &document.GetDocumentPermissionReq{
		UserId:     userID,
		DocumentId: grant.Document,
	})
	if err != nil {
		if grpcerr.DOCUMENT_NOT_EXIST_ERROR.EqualError(err) {
			return nil, httperr.ErrLinkNotExist
		}
		logx.WithContext(l.ctx).Errorf("get document permission fail: %v", err)
		return nil, response.ErrServer
	}
	if !permission.HasPermission {
		if _, err = l.svcCtx.DocumentRpc.GrantaDocumentPermission(l.ctx, &document.GrantDocumentPermissionReq{
			UserId:     userID,
			DocumentId: grant.Document,
		}); err != nil {
			if grpcerr.DOCUMENT_NOT_EXIST_ERROR.EqualError(err) {
				return nil, httperr.ErrLinkNotExist
			}
			logx.WithContext(l.ctx).Errorf("grant document permission fail: %v", err)
			return nil, httperr.ErrLinkNoPermission
		}
		logx.WithContext(l.ctx).Infow("grant document permission",
			logx.Field("user_id", userID),
			logx.Field("document_id", grant.Document),
			logx.Field("owner_id", grant.OwnerId),
		)
	}

	// 授权成功, 返回文档编辑页地址, 由前端自行跳转
	return &types.GetGrantDocumentResp{
		Link: l.svcCtx.Config.BaseUrl + globalkey.UrlDocument(grant.Document),
	}, nil
}
