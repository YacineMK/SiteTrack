// Package reqctx stores and reads request-scoped values (user id, app role,
// resolved project id, project-admin flag) set by authentication middleware
// and read by handlers.
package reqctx

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const (
	userIDKey       contextKey = "user_id"
	appRoleKey      contextKey = "app_role"
	projectIDKey    contextKey = "project_id"
	projectAdminKey contextKey = "project_admin"
)

// Context is the full read+write surface. Default is the production impl;
// tests can swap in their own.
type Context interface {
	WithAuth(ctx context.Context, userID uuid.UUID, appRole string) context.Context
	WithProject(ctx context.Context, projectID uuid.UUID, isAdmin bool) context.Context

	UserID(ctx context.Context) (uuid.UUID, bool)
	AppRole(ctx context.Context) (string, bool)
	ProjectID(ctx context.Context) (uuid.UUID, bool)
	IsProjectAdmin(ctx context.Context) (bool, bool)
}

type stdCtx struct{}

func (stdCtx) WithAuth(ctx context.Context, userID uuid.UUID, appRole string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	return context.WithValue(ctx, appRoleKey, appRole)
}

func (stdCtx) WithProject(ctx context.Context, projectID uuid.UUID, isAdmin bool) context.Context {
	ctx = context.WithValue(ctx, projectIDKey, projectID)
	return context.WithValue(ctx, projectAdminKey, isAdmin)
}

func (stdCtx) UserID(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(userIDKey).(uuid.UUID)
	return v, ok
}

func (stdCtx) AppRole(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(appRoleKey).(string)
	return v, ok
}

func (stdCtx) ProjectID(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(projectIDKey).(uuid.UUID)
	return v, ok
}

func (stdCtx) IsProjectAdmin(ctx context.Context) (bool, bool) {
	v, ok := ctx.Value(projectAdminKey).(bool)
	return v, ok
}

// Default is the production Context used by package-level helpers.
var Default Context = stdCtx{}

func WithAuth(ctx context.Context, userID uuid.UUID, appRole string) context.Context {
	return Default.WithAuth(ctx, userID, appRole)
}

func WithProject(ctx context.Context, projectID uuid.UUID, isAdmin bool) context.Context {
	return Default.WithProject(ctx, projectID, isAdmin)
}

func UserID(ctx context.Context) (uuid.UUID, bool)        { return Default.UserID(ctx) }
func AppRole(ctx context.Context) (string, bool)          { return Default.AppRole(ctx) }
func ProjectID(ctx context.Context) (uuid.UUID, bool)     { return Default.ProjectID(ctx) }
func IsProjectAdmin(ctx context.Context) (bool, bool)     { return Default.IsProjectAdmin(ctx) }
