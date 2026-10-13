// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/httperr"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/edit"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/document"
	"github.com/Bluore/mutidoc/pkg/ctxdata"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/response"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteDocumentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// delete document
func NewDeleteDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteDocumentLogic {
	return &DeleteDocumentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteDocumentLogic) DeleteDocument(req *types.DeleteDocumentReq) (resp *types.DeleteDocumentResp, err error) {
	// 操作者从jwt中获取
	ownerID := ctxdata.MustGetUserIdFromCtx(l.ctx)
	if req.DocumentId == "" {
		return nil, response.ErrValidation
	}

	// 校验文档归属, 只允许删除自己的文档
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
		logx.WithContext(l.ctx).Errorw("delete document permission denied",
			logx.Field("document_id", req.DocumentId),
			logx.Field("owner_id", result.DocumentInfo.OwnerId),
			logx.Field("operator_id", ownerID),
		)
		return nil, httperr.ErrDocumentNoPermission
	}

	_, err = l.svcCtx.DocumentRpc.DeleteDocument(l.ctx, &document.DeleteDocumentReq{
		DocumentId: req.DocumentId,
	})
	if err != nil {
		if grpcerr.DOCUMENT_NOT_EXIST_ERROR.EqualError(err) {
			return nil, httperr.ErrDocumentNotExist
		}
		logx.WithContext(l.ctx).Errorf("delete document fail: %v", err)
		return nil, response.ErrServer
	}

	// 删除成功后结束该文档的内存编辑会话; edit 不可用时不阻断删除结果
	if _, err = l.svcCtx.EditRpc.StopDocumentSession(l.ctx, &edit.StopDocumentSessionReq{
		DocumentId: req.DocumentId,
	}); err != nil {
		logx.WithContext(l.ctx).Errorw("stop edit session fail",
			logx.Field("error", err),
			logx.Field("document_id", req.DocumentId),
		)
	}

	logx.WithContext(l.ctx).Infow("delete document success",
		logx.Field("document_id", req.DocumentId),
	)

	return &types.DeleteDocumentResp{}, nil
}
