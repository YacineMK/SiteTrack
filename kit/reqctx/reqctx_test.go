package reqctx

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestAuthRoundtrip(t *testing.T) {
	id := uuid.New()
	ctx := WithAuth(context.Background(), id, "super_admin")

	gotID, ok := UserID(ctx)
	if !ok || gotID != id {
		t.Fatalf("UserID = %v, %v; want %v, true", gotID, ok, id)
	}
	gotRole, ok := AppRole(ctx)
	if !ok || gotRole != "super_admin" {
		t.Fatalf("AppRole = %q, %v; want %q, true", gotRole, ok, "super_admin")
	}
}

func TestProjectRoundtrip(t *testing.T) {
	pid := uuid.New()
	ctx := WithProject(context.Background(), pid, true)

	gotID, ok := ProjectID(ctx)
	if !ok || gotID != pid {
		t.Fatalf("ProjectID = %v, %v; want %v, true", gotID, ok, pid)
	}
	admin, ok := IsProjectAdmin(ctx)
	if !ok || !admin {
		t.Fatalf("IsProjectAdmin = %v, %v; want true, true", admin, ok)
	}
}

func TestEmptyContext(t *testing.T) {
	ctx := context.Background()

	if id, ok := UserID(ctx); ok || id != (uuid.UUID{}) {
		t.Fatalf("UserID on empty ctx = %v, %v; want zero, false", id, ok)
	}
	if role, ok := AppRole(ctx); ok || role != "" {
		t.Fatalf("AppRole on empty ctx = %q, %v; want \"\", false", role, ok)
	}
	if id, ok := ProjectID(ctx); ok || id != (uuid.UUID{}) {
		t.Fatalf("ProjectID on empty ctx = %v, %v; want zero, false", id, ok)
	}
	if admin, ok := IsProjectAdmin(ctx); ok || admin {
		t.Fatalf("IsProjectAdmin on empty ctx = %v, %v; want false, false", admin, ok)
	}
}

func TestOverwrite(t *testing.T) {
	first := uuid.New()
	second := uuid.New()
	ctx := WithAuth(context.Background(), first, "user")
	ctx = WithAuth(ctx, second, "super_admin")

	if got, _ := UserID(ctx); got != second {
		t.Fatalf("UserID after overwrite = %v; want %v", got, second)
	}
	if got, _ := AppRole(ctx); got != "super_admin" {
		t.Fatalf("AppRole after overwrite = %q; want super_admin", got)
	}
}

func TestWrongTypeInKey(t *testing.T) {
	// If someone pokes the wrong type into our key, cast returns zero + false.
	ctx := context.WithValue(context.Background(), userIDKey, "not-a-uuid")
	if id, ok := UserID(ctx); ok || id != (uuid.UUID{}) {
		t.Fatalf("UserID with wrong type = %v, %v; want zero, false", id, ok)
	}
}

type recordingCtx struct {
	stdCtx
	calls int
}

func (r *recordingCtx) UserID(ctx context.Context) (uuid.UUID, bool) {
	r.calls++
	return r.stdCtx.UserID(ctx)
}

func TestInterfaceSwappable(t *testing.T) {
	// Verifies the Default can be swapped — key for test fakes.
	rec := &recordingCtx{}
	orig := Default
	Default = rec
	t.Cleanup(func() { Default = orig })

	id := uuid.New()
	ctx := WithAuth(context.Background(), id, "user")
	UserID(ctx)
	UserID(ctx)
	if rec.calls != 2 {
		t.Fatalf("recorder calls = %d; want 2", rec.calls)
	}
}
