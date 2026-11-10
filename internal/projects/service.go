package projects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	ids "github.com/YacineMK/SiteTrack/kit/idgen"
	kitredis "github.com/YacineMK/SiteTrack/kit/redis"
)

var ErrInvalidInput = errors.New("invalid input")

type Service interface {
	Create(ctx context.Context, userID uuid.UUID, req CreateProjectRequest) (ProjectResponse, error)
	Get(ctx context.Context, displayID string, isAdmin bool) (ProjectResponse, error)
	List(ctx context.Context, p ListParams) (ListProjectsResponse, error)
	Update(ctx context.Context, displayID string, isAdmin bool, req UpdateProjectRequest) (ProjectResponse, error)
	Delete(ctx context.Context, displayID string) error
	ListMembers(ctx context.Context, projectID uuid.UUID) (ListMembersResponse, error)
	SetMemberAdmin(ctx context.Context, projectID, userID uuid.UUID, req UpdateMemberRequest) error
	RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error
	Invite(ctx context.Context, projectID uuid.UUID, projectDisplayID string, inviterID uuid.UUID, req InviteRequest) (InvitationResponse, error)
	WithdrawInvitation(ctx context.Context, projectID, notificationID uuid.UUID) error
	AcceptInvitation(ctx context.Context, userID, notificationID uuid.UUID) error
	RejectInvitation(ctx context.Context, userID, notificationID uuid.UUID) error
}

type service struct {
	repo       Repository
	ids        *ids.Generator
	utils      Utils
	redis      *kitredis.Redis
	inviteTTL  time.Duration
	appBaseURL string
}

func NewService(repo Repository, generator *ids.Generator, redis *kitredis.Redis, inviteTTL time.Duration, appBaseURL string) Service {
	return &service{
		repo:       repo,
		ids:        generator,
		utils:      NewUtils(),
		redis:      redis,
		inviteTTL:  inviteTTL,
		appBaseURL: strings.TrimRight(appBaseURL, "/"),
	}
}

func (s *service) Create(ctx context.Context, userID uuid.UUID, req CreateProjectRequest) (ProjectResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if err := req.Validate(); err != nil {
		return ProjectResponse{}, ErrInvalidInput
	}

	start, end, err := s.utils.ParseDates(req.PlannedStartDate, req.PlannedEndDate)
	if err != nil {
		return ProjectResponse{}, err
	}

	generated, err := s.ids.NewProject(ctx)
	if err != nil {
		return ProjectResponse{}, err
	}

	project, err := s.repo.Create(ctx, Project{
		ID:               generated.UUID,
		DisplayID:        generated.DisplayID,
		Name:             req.Name,
		Description:      req.Description,
		PlannedStartDate: start,
		PlannedEndDate:   end,
	}, userID)
	if err != nil {
		return ProjectResponse{}, err
	}

	return s.utils.ToResponse(project), nil
}

func (s *service) Get(ctx context.Context, displayID string, isAdmin bool) (ProjectResponse, error) {
	project, err := s.repo.GetByDisplayID(ctx, displayID)
	if err != nil {
		return ProjectResponse{}, err
	}
	project.IsAdmin = isAdmin
	return s.utils.ToResponse(project), nil
}

func (s *service) List(ctx context.Context, p ListParams) (ListProjectsResponse, error) {
	switch p.Sort {
	case "", "name", "start_date", "progress", "late":
	default:
		return ListProjectsResponse{}, ErrInvalidInput
	}
	p.Query = strings.TrimSpace(p.Query)
	p.Limit = s.utils.NormalizeLimit(p.Limit)
	if p.Offset < 0 {
		p.Offset = 0
	}

	items, total, err := s.repo.List(ctx, p)
	if err != nil {
		return ListProjectsResponse{}, err
	}

	return s.utils.ListResponse(items, total, p.Limit), nil
}

func (s *service) Update(ctx context.Context, displayID string, isAdmin bool, req UpdateProjectRequest) (ProjectResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if err := req.Validate(); err != nil {
		return ProjectResponse{}, ErrInvalidInput
	}

	start, end, err := s.utils.ParseDates(req.PlannedStartDate, req.PlannedEndDate)
	if err != nil {
		return ProjectResponse{}, err
	}

	project, err := s.repo.Update(ctx, displayID, req.Name, req.Description, start, end)
	if err != nil {
		return ProjectResponse{}, err
	}
	project.IsAdmin = isAdmin

	return s.utils.ToResponse(project), nil
}

func (s *service) Delete(ctx context.Context, displayID string) error {
	rows, err := s.repo.Delete(ctx, displayID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *service) ListMembers(ctx context.Context, projectID uuid.UUID) (ListMembersResponse, error) {
	members, err := s.repo.ListMembers(ctx, projectID)
	if err != nil {
		return ListMembersResponse{}, err
	}
	return s.utils.MembersResponse(members), nil
}

func (s *service) SetMemberAdmin(ctx context.Context, projectID, userID uuid.UUID, req UpdateMemberRequest) error {
	if err := req.Validate(); err != nil {
		return ErrInvalidInput
	}
	return s.repo.SetMemberAdmin(ctx, projectID, userID, *req.IsAdmin)
}

func (s *service) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	return s.repo.RemoveMember(ctx, projectID, userID)
}

type inviteData struct {
	ProjectID        string    `json:"project_id"`
	ProjectDisplayID string    `json:"project_display_id"`
	Role             string    `json:"role"`
	ExpiresAt        time.Time `json:"expires_at"`
}

func (s *service) Invite(ctx context.Context, projectID uuid.UUID, projectDisplayID string, inviterID uuid.UUID, req InviteRequest) (InvitationResponse, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Role = strings.TrimSpace(req.Role)
	if err := req.Validate(); err != nil {
		return InvitationResponse{}, ErrInvalidInput
	}

	invitee, err := s.repo.FindUserByEmail(ctx, req.Email)
	if err != nil {
		return InvitationResponse{}, err
	}

	member, _, err := s.repo.Membership(ctx, projectID, invitee.ID)
	if err != nil {
		return InvitationResponse{}, err
	}
	if member {
		return InvitationResponse{}, ErrAlreadyMember
	}

	pendingKey := fmt.Sprintf("invite:pending:%s:%s", projectID, invitee.ID)
	created, err := s.redis.Client.SetNX(ctx, pendingKey, "1", s.inviteTTL).Result()
	if err != nil {
		return InvitationResponse{}, err
	}
	if !created {
		return InvitationResponse{}, ErrInviteExists
	}

	notificationID, err := uuid.NewV7()
	if err != nil {
		return InvitationResponse{}, err
	}
	appOutboxID, err := uuid.NewV7()
	if err != nil {
		return InvitationResponse{}, err
	}
	emailOutboxID, err := uuid.NewV7()
	if err != nil {
		return InvitationResponse{}, err
	}

	expiresAt := time.Now().Add(s.inviteTTL)
	link := s.appBaseURL + "/invitations/" + notificationID.String()

	data, err := json.Marshal(inviteData{
		ProjectID:        projectID.String(),
		ProjectDisplayID: projectDisplayID,
		Role:             req.Role,
		ExpiresAt:        expiresAt,
	})
	if err != nil {
		return InvitationResponse{}, err
	}

	appPayload, err := json.Marshal(map[string]string{
		"outbox_id":       appOutboxID.String(),
		"notification_id": notificationID.String(),
	})
	if err != nil {
		return InvitationResponse{}, err
	}

	emailPayload, err := json.Marshal(map[string]string{
		"outbox_id": emailOutboxID.String(),
		"user_id":   invitee.ID.String(),
		"subject":   "Project invitation",
		"text":      fmt.Sprintf("You have been invited to project %s as %s. Accept or reject: %s", projectDisplayID, req.Role, link),
		"html":      fmt.Sprintf("<p>You have been invited to project <strong>%s</strong> as <strong>%s</strong>.</p><p><a href=\"%s\">Accept or reject</a></p>", projectDisplayID, req.Role, link),
	})
	if err != nil {
		return InvitationResponse{}, err
	}

	if err := s.repo.CreateInvitationNotification(ctx, InvitationNotification{
		ID:     notificationID,
		UserID: invitee.ID,
		Type:   "app_invite",
		Title:  "Project invitation",
		Body:   fmt.Sprintf("You have been invited to project %s as %s", projectDisplayID, req.Role),
		Data:   data,
	}, []OutboxEvent{
		{ID: appOutboxID, Kind: "app_notification", Payload: appPayload},
		{ID: emailOutboxID, Kind: "email", Payload: emailPayload},
	}); err != nil {
		s.redis.Client.Del(ctx, pendingKey)
		return InvitationResponse{}, err
	}

	if err := s.redis.Client.Set(ctx, "invite:"+notificationID.String(), "1", s.inviteTTL).Err(); err != nil {
		return InvitationResponse{}, err
	}

	return InvitationResponse{
		ID:               notificationID,
		ProjectDisplayID: projectDisplayID,
		InviteeID:        invitee.ID,
		Name:             invitee.Name,
		Email:            invitee.Email,
		Status:           "pending",
		ExpiresAt:        expiresAt,
		CreatedAt:        time.Now(),
	}, nil
}

func (s *service) WithdrawInvitation(ctx context.Context, projectID, notificationID uuid.UUID) error {
	invitation, err := s.invitation(ctx, notificationID)
	if err != nil {
		return err
	}
	data, err := s.inviteData(invitation)
	if err != nil {
		return err
	}
	if data.ProjectID != projectID.String() {
		return ErrInviteNotFound
	}

	rows, err := s.repo.DeleteInvitationNotification(ctx, notificationID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrInviteNotFound
	}

	s.redis.Client.Del(ctx, "invite:"+notificationID.String())
	s.redis.Client.Del(ctx, fmt.Sprintf("invite:pending:%s:%s", data.ProjectID, invitation.UserID))
	return nil
}

func (s *service) AcceptInvitation(ctx context.Context, userID, notificationID uuid.UUID) error {
	invitation, data, err := s.pendingInvitation(ctx, userID, notificationID)
	if err != nil {
		return err
	}

	projectID, err := uuid.Parse(data.ProjectID)
	if err != nil {
		return ErrInviteNotFound
	}

	if err := s.repo.AcceptInvitation(ctx, notificationID, projectID, userID, data.Role == "admin"); err != nil {
		return err
	}

	s.redis.Client.Del(ctx, "invite:"+invitation.ID.String())
	s.redis.Client.Del(ctx, fmt.Sprintf("invite:pending:%s:%s", data.ProjectID, userID))
	return nil
}

func (s *service) RejectInvitation(ctx context.Context, userID, notificationID uuid.UUID) error {
	invitation, data, err := s.pendingInvitation(ctx, userID, notificationID)
	if err != nil {
		return err
	}

	rows, err := s.repo.RejectInvitation(ctx, notificationID, userID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrInviteNotFound
	}

	s.redis.Client.Del(ctx, "invite:"+invitation.ID.String())
	s.redis.Client.Del(ctx, fmt.Sprintf("invite:pending:%s:%s", data.ProjectID, userID))
	return nil
}

func (s *service) invitation(ctx context.Context, notificationID uuid.UUID) (Invitation, error) {
	invitation, err := s.repo.GetInvitationNotification(ctx, notificationID)
	if err != nil {
		return Invitation{}, err
	}
	if invitation.Type != "app_invite" {
		return Invitation{}, ErrInviteNotFound
	}
	return invitation, nil
}

func (s *service) inviteData(invitation Invitation) (inviteData, error) {
	var data inviteData
	if err := json.Unmarshal(invitation.Data, &data); err != nil {
		return inviteData{}, ErrInviteNotFound
	}
	return data, nil
}

func (s *service) pendingInvitation(ctx context.Context, userID, notificationID uuid.UUID) (Invitation, inviteData, error) {
	invitation, err := s.invitation(ctx, notificationID)
	if err != nil {
		return Invitation{}, inviteData{}, err
	}
	if invitation.UserID != userID || invitation.ReadAt != nil {
		return Invitation{}, inviteData{}, ErrInviteNotFound
	}

	data, err := s.inviteData(invitation)
	if err != nil {
		return Invitation{}, inviteData{}, err
	}
	if time.Now().After(data.ExpiresAt) {
		return Invitation{}, inviteData{}, ErrInviteExpired
	}

	exists, err := s.redis.Client.Exists(ctx, "invite:"+notificationID.String()).Result()
	if err != nil {
		return Invitation{}, inviteData{}, err
	}
	if exists == 0 {
		return Invitation{}, inviteData{}, ErrInviteExpired
	}
	return invitation, data, nil
}

func parseDates(start, end string) (time.Time, time.Time, error) {
	startDate, err := time.Parse(dateLayout, start)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalidInput
	}

	endDate, err := time.Parse(dateLayout, end)
	if err != nil || endDate.Before(startDate) {
		return time.Time{}, time.Time{}, ErrInvalidInput
	}

	return startDate, endDate, nil
}
