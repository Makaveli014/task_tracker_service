package service

import (
	"context"
	"errors"
	"fmt"

	"TestTask_Bazis/internal/models"
	"TestTask_Bazis/internal/repository"
)

var (
	ErrNotAuthorized = errors.New("not authorized")
	ErrAlreadyMember = errors.New("user is already a member")
	ErrTeamNotFound  = errors.New("team not found")
)

type TeamService struct {
	teamRepo *repository.TeamRepository
	userRepo *repository.UserRepository
	emailSvc *EmailService
}

func NewTeamService(teamRepo *repository.TeamRepository, userRepo *repository.UserRepository, emailSvc *EmailService) *TeamService {
	return &TeamService{
		teamRepo: teamRepo,
		userRepo: userRepo,
		emailSvc: emailSvc,
	}
}

func (s *TeamService) Create(ctx context.Context, req *models.CreateTeamRequest, userID uint64) (*models.Team, error) {
	team, err := s.teamRepo.Create(ctx, req.Name, req.Description, userID)
	if err != nil {
		return nil, fmt.Errorf("create team: %w", err)
	}
	return team, nil
}

func (s *TeamService) ListByUser(ctx context.Context, userID uint64) ([]models.Team, error) {
	teams, err := s.teamRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	return teams, nil
}

func (s *TeamService) Invite(ctx context.Context, teamID uint64, inviterID uint64, req *models.InviteRequest) error {
	role, err := s.teamRepo.GetMemberRole(ctx, teamID, inviterID)
	if err != nil {
		return fmt.Errorf("get member role: %w", err)
	}
	if role != "owner" && role != "admin" {
		return ErrNotAuthorized
	}

	team, err := s.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("get team: %w", err)
	}
	if team == nil {
		return ErrTeamNotFound
	}

	isMember, err := s.teamRepo.IsMember(ctx, teamID, req.UserID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if isMember {
		return ErrAlreadyMember
	}

	inviteRole := req.Role
	if inviteRole == "" {
		inviteRole = "member"
	}

	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return errors.New("user not found")
	}

	if err := s.teamRepo.AddMember(ctx, teamID, req.UserID, inviteRole); err != nil {
		return fmt.Errorf("add member: %w", err)
	}

	// Circuit breaker: отправить email-приглашение (best effort)
	if s.emailSvc != nil {
		_ = s.emailSvc.SendInvite(ctx, user.Email, team.Name)
	}

	return nil
}
