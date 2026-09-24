// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/document"
	"github.com/Bluore/mutidoc/pkg/ctxdata"
	"github.com/Bluore/mutidoc/pkg/response"

	"github.com/zeromicro/go-zero/core/logx"
)

// 分页默认值
const (
	defaultPageSize = 20
	maxPageSize     = 100
)

type ListDocumentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// list document
func NewListDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListDocumentLogic {
	return &ListDocumentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListDocumentLogic) ListDocument(req *types.GetDocumentListReq) (resp *types.GetDocumentListResp, err error) {
	// 只查询自己的文档, 归属者从jwt中获取
	ownerID := ctxdata.MustGetUserIdFromCtx(l.ctx)

	if req.Page < 1 {
		req.Page = 1
	}
	if req.Size < 1 || req.Size > maxPageSize {
		req.Size = defaultPageSize
	}

	result, err := l.svcCtx.DocumentRpc.GetDocumentList(l.ctx, &document.GetDocumentListReq{
		OwnerId: ownerID,
		Page:    int64(req.Page),
		Size:    int64(req.Size),
	})
	if err != nil {
		logx.WithContext(l.ctx).Errorf("list document fail: %v", err)
		return nil, response.ErrServer
	}

	// 只对外暴露元信息, 文档正文由协作连接单独获取
	documentInfo := make([]types.Document, 0, len(result.DocumentInfo))
	for _, item := range result.DocumentInfo {
		if item == nil {
			continue
		}
		documentInfo = append(documentInfo, types.Document{
			Id:       item.Id,
			OwnerId:  item.OwnerId,
			Summary:  item.Summary,
			CreateAt: item.CreateAt,
			UpdateAt: item.UpdateAt,
		})
	}
	logx.WithContext(l.ctx).Infow("list document success",
		logx.Field("owner_id", ownerID),
		logx.Field("total", result.Total),
	)

	return &types.GetDocumentListResp{
		DocumentInfo: documentInfo,
		DocumentLen:  int(result.Total),
	}, nil
}
