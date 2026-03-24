package usecase

import (
	"context"
	"strings"

	"github.com/DrummDaddy/task_service/internal/core/ports"
	"github.com/DrummDaddy/task_service/internal/models"
	"github.com/DrummDaddy/task_service/internal/repo"
)

type TeamUseCase struct {
	teams ports.TeamRepository
	email ports.EmailSender
}

func NewTeamUsecase(teams ports.TeamRepository, email ports.EmailSender) *TeamUseCase {
	return &TeamUseCase{teams: teams, email: email}
}

func (uc *TeamUseCase) CreateTeam(ctx context.Context, userID uint64, name string) (uint64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, repo.ErrConflict
	}

	return uc.teams.CreateTeamWithOwner(ctx, name, userID)
}

func (uc *TeamUseCase) ListTeams(ctx context.Context, userID uint64) ([]models.Team, error) {
	return uc.teams.ListTeamsByUser(ctx, userID)
}

func (uc *TeamUseCase) Invite(ctx context.Context, actorID, teamID, invitedUserID uint64, role models.TeamMemberRole) (bool, error) {
	r, err := uc.teams.GetUserRole(ctx, teamID, actorID)
	if err != nil {
		return false, repo.ErrNotFound
	}
	if r != models.RoleOwner && r != models.RoleAdmin {
		return false, repo.ErrConflict
	}
	if role == "" {
		role = models.RoleMember
	}
	if role != models.RoleAdmin && role != models.RoleMember {
		return false, repo.ErrConflict
	}
	if err := uc.teams.AddMember(ctx, teamID, invitedUserID, role); err != nil {
		if err == repo.ErrConflict {
			return false, nil
		}
		return false, err
	}
	emailSent := true
	if uc.email != nil {
		if err := uc.email.SendInvite(ctx, ports.InvitePayload{
			TeamID:   teamID,
			UserID:   invitedUserID,
			InviteBy: actorID,
		}); err != nil {
			emailSent = false
		}
	}
	return emailSent, nil
}
