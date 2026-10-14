package logic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Bluore/mutidoc/app/document/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/pb"
	"github.com/Bluore/mutidoc/app/document/model"
	"github.com/Bluore/mutidoc/pkg/globalkey"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/ot"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDocumentContentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDocumentContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDocumentContentLogic {
	return &GetDocumentContentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取文档内容(整合数据库快照与 redis 上未落库的变更)
func (l *GetDocumentContentLogic) GetDocumentContent(in *pb.GetDocumentContentReq) (*pb.GetDocumentContentResp, error) {
	if in.DocumentId == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	documentModel, err := l.svcCtx.DocumentModel.FindOne(l.ctx, in.DocumentId)
	if err != nil && err != model.ErrNotFound {
		logx.WithContext(l.ctx).Errorf("FindOne fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db query error: %v", err)
	}
	if documentModel == nil {
		return nil, grpcerr.DOCUMENT_NOT_EXIST_ERROR
	}

	content, summary, version := l.mergeCache(documentModel)

	return &pb.GetDocumentContentResp{
		Content: content,
		Summary: summary,
		Version: version,
	}, nil
}

// documentCache 是文档在 redis 上的未落库变更, 与 edit 服务写入的格式保持一致。
// Version 为这批变更落库后的目标版本, Summary 为最新标题, Ops 为数据库内容到
// 当前内容的净变换(紧凑数组格式)。
type documentCache struct {
	Version int64           `json:"version"`
	Summary string          `json:"summary,omitempty"`
	Ops     json.RawMessage `json:"ops"`
}

// mergeCache 把 redis 上未落库的变更合并进数据库快照, 返回正文、标题与版本。
// 缓存缺失、过期或损坏时退化为数据库快照, 不影响查询。
func (l *GetDocumentContentLogic) mergeCache(detail *model.Documents) (content, summary string, version int64) {
	content = detail.Content
	summary = detail.Summary
	version = detail.Version

	key := fmt.Sprintf(globalkey.RedisKeyDocumentChange, detail.Id)
	raw, err := l.svcCtx.RedisClient.Get(l.ctx, key).Bytes()
	if err == redis.Nil {
		return
	}
	if err != nil {
		l.Errorw("load document cache fail",
			logx.Field("error", err),
			logx.Field("document_id", detail.Id),
		)
		return
	}

	var cache documentCache
	if err := json.Unmarshal(raw, &cache); err != nil {
		l.Errorw("unmarshal document cache fail",
			logx.Field("error", err),
			logx.Field("document_id", detail.Id),
		)
		return
	}

	// 缓存版本不大于数据库版本, 说明变更已落库, 直接用数据库快照
	if cache.Version <= detail.Version {
		return
	}

	merged := detail.Content
	if len(cache.Ops) > 0 {
		oper := &ot.Operator{}
		if err := json.Unmarshal(cache.Ops, oper); err != nil {
			l.Errorw("decode document cache operator fail",
				logx.Field("error", err),
				logx.Field("document_id", detail.Id),
			)
			return
		}
		if len(oper.Ops) > 0 {
			merged = string(oper.Apply([]byte(detail.Content)))
		}
	}

	content = merged
	version = cache.Version
	if cache.Summary != "" {
		summary = cache.Summary
	}
	return
}
