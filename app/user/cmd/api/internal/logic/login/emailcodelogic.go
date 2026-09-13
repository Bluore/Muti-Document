// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package login

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/httperr"
	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/pkg/emailcontrol"
	"github.com/Bluore/mutidoc/pkg/globalkey"
	"github.com/Bluore/mutidoc/pkg/response"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

type EmailCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// get email code
func NewEmailCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EmailCodeLogic {
	return &EmailCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EmailCodeLogic) EmailCode(req *types.EmailCodeReq) (resp *types.EmailCodeResp, err error) {
	// email格式化
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// todo 添加robot 检查

	// 验证邮箱格式
	ok := globalkey.RegexpEmail.MatchString(req.Email)
	if !ok {
		return nil, httperr.ErrEmailFormatUnsupported
	}

	// 频率限制
	result, err := l.svcCtx.RedisClient.Get(l.ctx, fmt.Sprintf(globalkey.RedisKeyUserVerifyLimit, req.Email)).Result()
	if err != nil && err != redis.Nil {
		logx.WithContext(l.ctx).Errorf("redis fail: %v", err)
		return nil, response.ErrServer
	}
	if result != "" {
		return nil, httperr.ErrEmailLimitExceed
	}

	// 更新redis
	verifyCode := rand.Intn(999999)
	_, err = l.svcCtx.RedisClient.Set(
		l.ctx,
		fmt.Sprintf(globalkey.RedisKeyUserVerifyKey, req.Email),
		fmt.Sprintf("%06d", verifyCode),
		time.Minute*10,
	).Result()
	if err != nil {
		logx.WithContext(l.ctx).Errorf("redis fail: %v", err)
		return nil, response.ErrServer
	}
	_, err = l.svcCtx.RedisClient.Set(
		l.ctx,
		fmt.Sprintf(globalkey.RedisKeyUserVerifyLimit, req.Email),
		1,
		time.Minute*1,
	).Result()
	if err != nil {
		logx.WithContext(l.ctx).Errorf("redis fail: %v", err)
		return nil, response.ErrServer
	}

	// 发送邮件
	emailText := `
你好，
欢迎注册mutidoc，你的一次性验证码是 %06d 。
请在10分钟内完成验证
`
	email := &emailcontrol.EmailContent{
		To:      []string{req.Email},
		From:    l.svcCtx.Config.EmailSender.SenderEmail,
		Text:    fmt.Sprintf(emailText, verifyCode),
		Subject: "一次性验证码",
	}
	err = l.svcCtx.EmailSender.SendEmail(l.ctx, email)
	if err != nil {
		return nil, httperr.ErrSendEmailFailed
	}

	return &types.EmailCodeResp{}, nil
}
