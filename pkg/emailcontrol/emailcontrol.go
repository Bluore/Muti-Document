package emailcontrol

import (
	"context"
	"time"
)

type EmailSender interface {
	SendEmail(ctx context.Context, email *EmailContent) error
}

type EmailContent struct {
	To      []string
	From    string
	Text    string
	Subject string

	BCC         []string
	CC          []string
	ScheduledAt time.Time
}
