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

type GetDocumentContentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// get document content
func NewGetDocumentContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDocumentContentLogic {
	return &GetDocumentContentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetDocumentContentLogic) GetDocumentContent(req *types.GetDocumentContentReq) (resp *types.GetDocumentContentResp, err error) {
	// 操作者从jwt中获取
	userID := ctxdata.MustGetUserIdFromCtx(l.ctx)
	if req.DocumentId == "" {
		return nil, response.ErrValidation
	}

	// 先校验权限, 无权限直接拒绝
	permission, err := l.svcCtx.DocumentRpc.GetDocumentPermission(l.ctx, &document.GetDocumentPermissionReq{
		UserId:     userID,
		DocumentId: req.DocumentId,
	})
	if err != nil {
		if grpcerr.DOCUMENT_NOT_EXIST_ERROR.EqualError(err) {
			return nil, httperr.ErrDocumentNotExist
		}
		logx.WithContext(l.ctx).Errorf("get document permission fail: %v", err)
		return nil, response.ErrServer
	}
	if !permission.GetHasPermission() {
		logx.WithContext(l.ctx).Errorw("get document content permission denied",
			logx.Field("document_id", req.DocumentId),
			logx.Field("operator_id", userID),
		)
		return nil, httperr.ErrDocumentNoPermission
	}

	// 正文由 document-rpc 整合数据库快照与未落库变更后返回
	result, err := l.svcCtx.DocumentRpc.GetDocumentContent(l.ctx, &document.GetDocumentContentReq{
		DocumentId: req.DocumentId,
	})
	if err != nil {
		if grpcerr.DOCUMENT_NOT_EXIST_ERROR.EqualError(err) {
			return nil, httperr.ErrDocumentNotExist
		}
		logx.WithContext(l.ctx).Errorf("get document content fail: %v", err)
		return nil, response.ErrServer
	}

	return &types.GetDocumentContentResp{
		Content: result.Content,
		Summary: result.Summary,
		Version: result.Version,
	}, nil
}
