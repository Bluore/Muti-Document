package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/pb"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/pkg/errors"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListSessionLogic {
	return &ListSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 会话列表
func (l *ListSessionLogic) ListSession(in *pb.ListSessionReq) (*pb.ListSessionResp, error) {
	if in.UserId == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	// 以数据库为准返回用户全部会话, 不受进程内存影响, 按创建时间倒序
	rows, err := l.svcCtx.Session.Model.FindAll(
		l.ctx,
		l.svcCtx.Session.Model.SelectBuilder().Where("user_id = ?", in.UserId),
		"create_at DESC",
	)
	if err != nil {
		logx.WithContext(l.ctx).Errorf("list session fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "list session error: %v", err)
	}

	sessions := make([]*pb.Session, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		sessions = append(sessions, &pb.Session{
			SessionId:    row.Id,
			SessionTitle: row.Title,
			UserId:       row.UserId,
		})
	}

	logx.WithContext(l.ctx).Infow("list session success",
		logx.Field("user_id", in.UserId),
		logx.Field("count", len(sessions)),
	)

	return &pb.ListSessionResp{Sessions: sessions}, nil
}
