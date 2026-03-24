package ports

import (
	"context"

	"github.com/DrummDaddy/task_service/internal/email"
)

type EmailSender interface {
	SendInvite(ctx context.Context, p email.InvitePayload) error
}
