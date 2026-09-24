package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/pb"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/pkg/errors"

	"github.com/zeromicro/go-zero/core/logx"
)

// 分页默认值
const (
	defaultPageSize int64 = 20
	maxPageSize     int64 = 100
)

type GetDocumentListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDocumentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDocumentListLogic {
	return &GetDocumentListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取文档列表
func (l *GetDocumentListLogic) GetDocumentList(in *pb.GetDocumentListReq) (*pb.GetDocumentListResp, error) {
	if in.OwnerId == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	page, size := in.Page, in.Size
	if page < 1 {
		page = 1
	}
	if size < 1 || size > maxPageSize {
		size = defaultPageSize
	}

	documents, total, err := l.svcCtx.DocumentModel.FindPageListByPageWithTotal(
		l.ctx,
		l.svcCtx.DocumentModel.SelectBuilder().Where("owner_id = ?", in.OwnerId),
		page,
		size,
		"create_at DESC",
	)
	if err != nil {
		logx.WithContext(l.ctx).Errorf("FindPageListByPageWithTotal fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db query error: %v", err)
	}

	// 只返回元信息, 文档正文由协作连接单独获取
	documentInfo := make([]*pb.Document, 0, len(documents))
	for _, documentModel := range documents {
		if documentModel == nil {
			continue
		}
		documentInfo = append(documentInfo, &pb.Document{
			Id:       documentModel.Id,
			OwnerId:  documentModel.OwnerId,
			Summary:  documentModel.Summary,
			Version:  documentModel.Version,
			CreateAt: documentModel.CreateAt.Unix(),
			UpdateAt: documentModel.UpdateAt.Unix(),
			DeleteAt: documentModel.DeleteAt.Unix(),
		})
	}
	logx.WithContext(l.ctx).Infow("list document success",
		logx.Field("owner_id", in.OwnerId),
		logx.Field("total", total),
	)

	return &pb.GetDocumentListResp{
		DocumentInfo: documentInfo,
		Total:        total,
	}, nil
}
