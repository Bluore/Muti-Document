package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/pb"
	"github.com/Bluore/mutidoc/app/document/model"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/idgenerate"
	"github.com/pkg/errors"

	"github.com/zeromicro/go-zero/core/logx"
)

// 文档摘要默认值, 与documents表默认值保持一致
const defaultDocumentSummary = "new document"

type CreateDocumentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDocumentLogic {
	return &CreateDocumentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 创建文档
func (l *CreateDocumentLogic) CreateDocument(in *pb.CreateDocumentReq) (*pb.CreateDocumentResp, error) {
	if in.OwnerId == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	// 正文内容由协作连接维护, 创建时为空
	documentModel := &model.Documents{
		Id:      idgenerate.NewID(l.svcCtx.Config.ServerID),
		OwnerId: in.OwnerId,
		Summary: defaultDocumentSummary,
		Content: "",
		Version: 1,
	}

	_, err := l.svcCtx.DocumentModel.Insert(l.ctx, nil, documentModel)
	if err != nil {
		logx.WithContext(l.ctx).Errorf("insert document fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db insert error: %v", err)
	}
	logx.WithContext(l.ctx).Infow("create document success",
		logx.Field("document_id", documentModel.Id),
		logx.Field("owner_id", documentModel.OwnerId),
	)

	return &pb.CreateDocumentResp{
		DocumentId: documentModel.Id,
	}, nil
}
