package projects

import (
	"errors"
	"net/http"
	"time"

	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type Utils interface {
	ToResponse(project Project) ProjectResponse
	ListResponse(projects []Project, total int64, limit int32) ListProjectsResponse
	MembersResponse(members []Member) ListMembersResponse
	WriteServiceError(w http.ResponseWriter, err error)
	ParseDates(start, end string) (time.Time, time.Time, error)
	NormalizeLimit(limit int32) int32
}

type utils struct{}

func NewUtils() Utils {
	return utils{}
}

func (utils) ToResponse(project Project) ProjectResponse {
	return ProjectResponse{
		DisplayID:        project.DisplayID,
		Name:             project.Name,
		Description:      project.Description,
		PlannedStartDate: project.PlannedStartDate.Format(dateLayout),
		PlannedEndDate:   project.PlannedEndDate.Format(dateLayout),
		IsAdmin:          project.IsAdmin,
		MemberCount:      project.MemberCount,
		TaskCounts: TaskCounts{
			NotStarted: project.NotStarted,
			InProgress: project.InProgress,
			Done:       project.Done,
		},
		Progress:     project.Progress,
		LateCount:    project.LateCount,
		BlockedCount: project.BlockedCount,
		CreatedAt:    project.CreatedAt,
		UpdatedAt:    project.UpdatedAt,
	}
}

func (u utils) ListResponse(projects []Project, total int64, limit int32) ListProjectsResponse {
	items := make([]ProjectResponse, 0, len(projects))
	for _, project := range projects {
		items = append(items, u.ToResponse(project))
	}
	return ListProjectsResponse{
		Items: items,
		Total: total,
		Limit: limit,
	}
}

func (utils) MembersResponse(members []Member) ListMembersResponse {
	items := make([]MemberResponse, 0, len(members))
	for _, member := range members {
		items = append(items, MemberResponse{
			UserID:    member.UserID,
			Name:      member.Name,
			Email:     member.Email,
			IsAdmin:   member.IsAdmin,
			CreatedAt: member.CreatedAt,
		})
	}
	return ListMembersResponse{Items: items, Total: int64(len(items))}
}

func (utils) WriteServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		jsonutil.Error(w, http.StatusBadRequest, "invalid input")
	case errors.Is(err, ErrNotFound):
		jsonutil.Error(w, http.StatusNotFound, "project not found")
	case errors.Is(err, ErrMemberNotFound):
		jsonutil.Error(w, http.StatusNotFound, "project member not found")
	case errors.Is(err, ErrLastAdmin):
		jsonutil.Error(w, http.StatusConflict, "project must keep at least one admin")
	case errors.Is(err, ErrInviteExists):
		jsonutil.Error(w, http.StatusConflict, "invitation already exists")
	case errors.Is(err, ErrInviteNotFound):
		jsonutil.Error(w, http.StatusNotFound, "invitation not found")
	case errors.Is(err, ErrInviteExpired):
		jsonutil.Error(w, http.StatusGone, "invitation expired")
	case errors.Is(err, ErrInviteeNotFound):
		jsonutil.Error(w, http.StatusNotFound, "user not found")
	case errors.Is(err, ErrAlreadyMember):
		jsonutil.Error(w, http.StatusConflict, "user is already a member")
	case errors.Is(err, ErrProjectInUse):
		jsonutil.Error(w, http.StatusConflict, "project has deliveries")
	default:
		jsonutil.Error(w, http.StatusInternalServerError, "internal server error")
	}
}

func (utils) NormalizeLimit(limit int32) int32 {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func (utils) ParseDates(start, end string) (time.Time, time.Time, error) {
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
