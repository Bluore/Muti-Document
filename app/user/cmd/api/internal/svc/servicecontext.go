// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/config"
	"github.com/Bluore/mutidoc/pkg/emailcontrol"
)

type ServiceContext struct {
	Config      config.Config
	EmailSender emailcontrol.EmailSender
}

func NewServiceContext(c config.Config) *ServiceContext {
	if c.EmailSender.Platform != "resend" {
		panic("email platform not supported")
	}
	emailClient := emailcontrol.NewResendClient(c.EmailSender.ApiKey)

	return &ServiceContext{
		Config:      c,
		EmailSender: emailClient,
	}
}
