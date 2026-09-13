package httperr

import "github.com/Bluore/mutidoc/pkg/response"

var (
	ErrSendEmailFailed        = response.NewError(200, 1015001, "发送邮件失败，请稍后再试")
	ErrEmailFormatUnsupported = response.NewError(200, 1014002, "邮箱格式错误或邮箱不支持")
	ErrEmailLimitExceed       = response.NewError(200, 1014003, "邮件发送太快了，请稍后再试")
	ErrEmailNotSendVerify     = response.NewError(200, 1014004, "验证码已过期或没有发送验证码，请先发送验证码")
	ErrEmailVerifyFail        = response.NewError(200, 1014005, "验证码错误")
	ErrEmailExist             = response.NewError(200, 1014006, "邮箱已存在")
	ErrNicknameExist          = response.NewError(200, 1014006, "昵称已使用")
	ErrPasswordWrong          = response.NewError(200, 1014007, "用户名或密码错误")
	ErrUserNotExist           = response.NewError(200, 1014008, "用户不存在")
)
