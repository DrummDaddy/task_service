package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/DrummDaddy/task_service/internal/core/ports"
	"github.com/DrummDaddy/task_service/internal/models"
	"github.com/DrummDaddy/task_service/internal/repo"
	"github.com/stretchr/testify/require"
)

type mockEmailSender struct {
	sendFn func(ctx context.Context, p ports.InvitePayload) error
	called bool
	p      ports.InvitePayload
}

func (m *mockEmailSender) SendInvite(ctx context.Context, p ports.InvitePayload) error {
	m.called = true
	m.p = p
	if m.sendFn != nil {
		return m.sendFn(ctx, p)
	}
	return nil
}

func TestTeamUsecase_CreateTeam(t *testing.T) {
	teams := &mockTeamRepo{
		createTeamFn: func(ctx context.Context, name string, createdBy uint64) (uint64, error) {
			require.Equal(t, "team1", name)
			require.Equal(t, uint64(1), createdBy)
			return 99, nil
		},
	}
	uc := NewTeamUsecase(teams, nil)
	id, err := uc.CreateTeam(context.Background(), 1, " team1 ")
	require.NoError(t, err)
	require.Equal(t, uint64(99), id)
}

func TestTeamUsecase_Invite_Success_EmailOK(t *testing.T) {
	teams := &mockTeamRepo{
		getRoleFn: func(ctx context.Context, teamID, userID uint64) (models.TeamMemberRole, error) {
			return models.RoleOwner, nil
		},
		addMemberFn: func(ctx context.Context, teamID, userID uint64, role models.TeamMemberRole) error { return nil },
	}
	email := &mockEmailSender{}
	uc := NewTeamUsecase(teams, email)
	sent, err := uc.Invite(context.Background(), 1, 10, 7, models.RoleMember)
	require.NoError(t, err)
	require.True(t, sent)
	require.True(t, email.called)
	require.Equal(t, ports.InvitePayload{TeamID: 10, UserID: 7, InviteBy: 1}, email.p)
}

func TestTeamUsecase_Invite_EmailFailed(t *testing.T) {
	teams := &mockTeamRepo{
		getRoleFn: func(ctx context.Context, teamID, userID uint64) (models.TeamMemberRole, error) {
			return models.RoleAdmin, nil
		},
		addMemberFn: func(ctx context.Context, teamID, userID uint64, role models.TeamMemberRole) error { return nil },
	}
	email := &mockEmailSender{
		sendFn: func(ctx context.Context, p ports.InvitePayload) error { return errors.New("email down") },
	}
	uc := NewTeamUsecase(teams, email)
	sent, err := uc.Invite(context.Background(), 1, 10, 7, models.RoleMember)
	require.NoError(t, err)
	require.False(t, sent, "emailSent should be false when email sender fails")
}

func TestTeamUsecase_Invite_InsufficientRole(t *testing.T) {
	teams := &mockTeamRepo{
		getRoleFn: func(ctx context.Context, teamID, userID uint64) (models.TeamMemberRole, error) {
			return models.RoleMember, nil
		},
	}
	uc := NewTeamUsecase(teams, nil)
	_, err := uc.Invite(context.Background(), 1, 10, 7, models.RoleMember)
	require.ErrorIs(t, err, repo.ErrConflict)
}

func TestTeamUsecase_Invite_InvalidRole(t *testing.T) {
	teams := &mockTeamRepo{
		getRoleFn: func(ctx context.Context, teamID, userID uint64) (models.TeamMemberRole, error) {
			return models.RoleOwner, nil
		},
	}
	uc := NewTeamUsecase(teams, nil)
	_, err := uc.Invite(context.Background(), 1, 10, 7, "super-admin")
	require.ErrorIs(t, err, repo.ErrConflict)
}

func TestTeamUsecase_Invite_AlreadyMember(t *testing.T) {
	teams := &mockTeamRepo{
		getRoleFn: func(ctx context.Context, teamID, userID uint64) (models.TeamMemberRole, error) {
			return models.RoleOwner, nil
		},
		addMemberFn: func(ctx context.Context, teamID, userID uint64, role models.TeamMemberRole) error {
			return repo.ErrConflict
		},
	}
	uc := NewTeamUsecase(teams, nil)
	sent, err := uc.Invite(context.Background(), 1, 10, 7, models.RoleMember)
	require.NoError(t, err, "usecase returns nil err for already member")
	require.False(t, sent)
}
