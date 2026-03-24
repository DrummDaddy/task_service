package ports

import (
	"context"
)

type InvitePayload struct {
	TeamID   uint64 `json:"team_id"`
	UserID   uint64 `json:"user_id"`
	InviteBy uint64 `json:"invite_by"`
}

type EmailSender interface {
	SendInvite(ctx context.Context, p InvitePayload) error
}
