package ids

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestProjectDisplayID(t *testing.T) {
	tests := []struct {
		seq  uint64
		want string
	}{
		{seq: 1, want: "PRJ-0001"},
		{seq: 42, want: "PRJ-0042"},
		{seq: 9999, want: "PRJ-9999"},
		{seq: 10000, want: "PRJ-10000"},
	}

	for _, tt := range tests {
		if got := ProjectDisplayID(tt.seq); got != tt.want {
			t.Fatalf("ProjectDisplayID(%d) = %q, want %q", tt.seq, got, tt.want)
		}
	}
}

func TestTaskDisplayID(t *testing.T) {
	tests := []struct {
		project string
		seq     uint64
		want    string
	}{
		{project: "PRJ-0042", seq: 1, want: "PRJ-0042-T-0001"},
		{project: "PRJ-0042", seq: 9999, want: "PRJ-0042-T-9999"},
		{project: "PRJ-10000", seq: 10000, want: "PRJ-10000-T-10000"},
	}

	for _, tt := range tests {
		if got := TaskDisplayID(tt.project, tt.seq); got != tt.want {
			t.Fatalf("TaskDisplayID(%q, %d) = %q, want %q", tt.project, tt.seq, got, tt.want)
		}
	}
}

func TestGenerator(t *testing.T) {
	project := uuid.New()
	gen := NewGenerator(
		func(ctx context.Context) (int64, error) { return 42, nil },
		func(ctx context.Context, projectID uuid.UUID) (int64, error) {
			if projectID != project {
				t.Fatalf("task sequence got project %s, want %s", projectID, project)
			}
			return 7, nil
		},
	)

	p, err := gen.NewProject(context.Background())
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if p.UUID.Version() != 7 || p.DisplayID != "PRJ-0042" {
		t.Fatalf("project = %+v", p)
	}

	task, err := gen.NewTask(context.Background(), project, "PRJ-0042")
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	if task.UUID.Version() != 7 || task.DisplayID != "PRJ-0042-T-0007" {
		t.Fatalf("task = %+v", task)
	}
}

func TestGeneratorPropagatesSequenceError(t *testing.T) {
	expected := errors.New("sequence failed")
	gen := NewGenerator(
		func(ctx context.Context) (int64, error) { return 0, expected },
		func(ctx context.Context, projectID uuid.UUID) (int64, error) { return 0, expected },
	)

	if _, err := gen.NewProject(context.Background()); !errors.Is(err, expected) {
		t.Fatalf("NewProject() error = %v, want %v", err, expected)
	}
	if _, err := gen.NewTask(context.Background(), uuid.New(), "PRJ-0001"); !errors.Is(err, expected) {
		t.Fatalf("NewTask() error = %v, want %v", err, expected)
	}
}
