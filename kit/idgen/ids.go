package ids

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ID struct {
	UUID      uuid.UUID
	DisplayID string
}

type ProjectSequence func(ctx context.Context) (int64, error)
type TaskSequence func(ctx context.Context, projectID uuid.UUID) (int64, error)

type Generator interface {
	NewProject(ctx context.Context) (ID, error)
	NewTask(ctx context.Context, projectID uuid.UUID, projectDisplayID string) (ID, error)
}

type generator struct {
	project ProjectSequence
	task    TaskSequence
}

func NewGenerator(project ProjectSequence, task TaskSequence) Generator {
	return &generator{project: project, task: task}
}

func (g *generator) NewProject(ctx context.Context) (ID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return ID{}, err
	}

	seq, err := g.project(ctx)
	if err != nil {
		return ID{}, err
	}

	return ID{UUID: id, DisplayID: ProjectDisplayID(uint64(seq))}, nil
}

func (g *generator) NewTask(ctx context.Context, projectID uuid.UUID, projectDisplayID string) (ID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return ID{}, err
	}

	seq, err := g.task(ctx, projectID)
	if err != nil {
		return ID{}, err
	}

	return ID{UUID: id, DisplayID: TaskDisplayID(projectDisplayID, uint64(seq))}, nil
}

func ProjectDisplayID(seq uint64) string {
	return fmt.Sprintf("PRJ-%04d", seq)
}

func TaskDisplayID(projectDisplayID string, seq uint64) string {
	return fmt.Sprintf("%s-T-%04d", projectDisplayID, seq)
}
