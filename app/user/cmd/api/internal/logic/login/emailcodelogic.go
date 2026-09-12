// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package login

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/pkg/emailcontrol"
	"github.com/Bluore/mutidoc/pkg/response"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	ErrSendEmailFailed = response.NewError(200, 1015001, "发送邮件失败，请稍后再试")
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
	// todo 添加rebot 检查

	// todo 验证邮箱格式

	// 发送邮件
	verifyCode := rand.Intn(999999)
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
		return nil, ErrSendEmailFailed
	}

	return
}
