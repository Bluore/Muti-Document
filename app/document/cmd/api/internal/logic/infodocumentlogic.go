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

type InfoDocumentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// get document info
func NewInfoDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InfoDocumentLogic {
	return &InfoDocumentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *InfoDocumentLogic) InfoDocument(req *types.GetDocumentInfoReq) (resp *types.GetDocumentInfoResp, err error) {
	if req.DocumentId == "" {
		return nil, response.ErrValidation
	}

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

	// 只对外暴露元信息, 文档正文由协作连接单独获取
	return &types.GetDocumentInfoResp{
		DocumentInfo: types.Document{
			Id:       result.DocumentInfo.Id,
			OwnerId:  result.DocumentInfo.OwnerId,
			Summary:  result.DocumentInfo.Summary,
			CreateAt: result.DocumentInfo.CreateAt,
			UpdateAt: result.DocumentInfo.UpdateAt,
		},
	}, nil
}
