package emailcontrol

import (
	"context"

	"github.com/resend/resend-go/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type resendClient struct {
	client *resend.Client
}

func NewResendClient(apikey string) EmailSender {
	return &resendClient{client: resend.NewClient(apikey)}
}

func (c *resendClient) SendEmail(ctx context.Context, email *EmailContent) error {
	params := &resend.SendEmailRequest{
		To:      email.To,
		From:    email.From,
		Subject: email.Subject,
		Text:    email.Text,
		Bcc:     email.BCC,
		Cc:      email.CC,
	}
	result, err := c.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return err
	}
	logx.Infof("resend send email result: %+v", result)

	return nil
}
