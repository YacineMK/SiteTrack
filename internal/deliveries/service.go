package deliveries

import (
	"context"
	"errors"
	"sort"

	"github.com/google/uuid"

	"github.com/YacineMK/SiteTrack/internal/projects"
)

var ErrInvalidInput = errors.New("invalid input")

type Service interface {
	CreateFromCompany(ctx context.Context, req CreateCompanyDeliveryRequest) (DeliveryResponse, error)
	CreateFromProject(ctx context.Context, projectDisplayID string, req CreateProjectDeliveryRequest) (DeliveryResponse, error)
	ListCompany(ctx context.Context, status string, limit, offset int32) (ListDeliveriesResponse, error)
	ListProject(ctx context.Context, projectDisplayID, status string, limit, offset int32) (ListDeliveriesResponse, error)
	Get(ctx context.Context, id string) (DeliveryResponse, error)
	Record(ctx context.Context, id string) (DeliveryResponse, error)
	Reject(ctx context.Context, id string) (DeliveryResponse, error)
	Cancel(ctx context.Context, id string) (DeliveryResponse, error)
}

type service struct {
	repo     Repository
	projects projects.Repository
	utils    Utils
}

func NewService(repo Repository, projectRepo projects.Repository) Service {
	return &service{repo: repo, projects: projectRepo, utils: NewUtils()}
}

func (s *service) CreateFromCompany(ctx context.Context, req CreateCompanyDeliveryRequest) (DeliveryResponse, error) {
	if err := req.Validate(); err != nil {
		return DeliveryResponse{}, ErrInvalidInput
	}

	receiver, err := s.projects.GetByDisplayID(ctx, req.ReceiverProject)
	if err != nil {
		return DeliveryResponse{}, err
	}

	lines, err := s.parseLines(req.Lines)
	if err != nil {
		return DeliveryResponse{}, err
	}

	delivery, err := s.repo.Create(ctx, Delivery{
		SenderType:   PartyCompany,
		ReceiverType: PartyProject,
		ReceiverID:   &receiver.ID,
		Lines:        lines,
	})
	if err != nil {
		return DeliveryResponse{}, err
	}

	return s.utils.ToResponse(delivery), nil
}

func (s *service) CreateFromProject(ctx context.Context, projectDisplayID string, req CreateProjectDeliveryRequest) (DeliveryResponse, error) {
	if err := req.Validate(); err != nil {
		return DeliveryResponse{}, ErrInvalidInput
	}

	sender, err := s.projects.GetByDisplayID(ctx, projectDisplayID)
	if err != nil {
		return DeliveryResponse{}, err
	}

	lines, err := s.parseLines(req.Lines)
	if err != nil {
		return DeliveryResponse{}, err
	}

	delivery, err := s.repo.Create(ctx, Delivery{
		SenderType:   PartyProject,
		SenderID:     &sender.ID,
		ReceiverType: PartyCompany,
		Lines:        lines,
	})
	if err != nil {
		return DeliveryResponse{}, err
	}

	return s.utils.ToResponse(delivery), nil
}

func (s *service) ListCompany(ctx context.Context, status string, limit, offset int32) (ListDeliveriesResponse, error) {
	if !validStatus(status) {
		return ListDeliveriesResponse{}, ErrInvalidInput
	}

	limit, offset = s.utils.NormalizePagination(limit, offset)

	deliveries, total, err := s.repo.ListCompany(ctx, status, limit, offset)
	if err != nil {
		return ListDeliveriesResponse{}, err
	}

	return s.utils.ListResponse(deliveries, total, limit, offset), nil
}

func (s *service) ListProject(ctx context.Context, projectDisplayID, status string, limit, offset int32) (ListDeliveriesResponse, error) {
	if !validStatus(status) {
		return ListDeliveriesResponse{}, ErrInvalidInput
	}

	project, err := s.projects.GetByDisplayID(ctx, projectDisplayID)
	if err != nil {
		return ListDeliveriesResponse{}, err
	}

	limit, offset = s.utils.NormalizePagination(limit, offset)

	deliveries, total, err := s.repo.ListProject(ctx, project.ID, status, limit, offset)
	if err != nil {
		return ListDeliveriesResponse{}, err
	}

	return s.utils.ListResponse(deliveries, total, limit, offset), nil
}

func (s *service) Get(ctx context.Context, id string) (DeliveryResponse, error) {
	deliveryID, err := s.utils.ParseID(id)
	if err != nil {
		return DeliveryResponse{}, err
	}

	delivery, err := s.repo.Get(ctx, deliveryID)
	if err != nil {
		return DeliveryResponse{}, err
	}

	return s.utils.ToResponse(delivery), nil
}

func (s *service) Record(ctx context.Context, id string) (DeliveryResponse, error) {
	return s.transition(ctx, id, StatusDelivered)
}

func (s *service) Reject(ctx context.Context, id string) (DeliveryResponse, error) {
	return s.transition(ctx, id, StatusRejected)
}

func (s *service) Cancel(ctx context.Context, id string) (DeliveryResponse, error) {
	return s.transition(ctx, id, StatusCancelled)
}

func (s *service) transition(ctx context.Context, id, status string) (DeliveryResponse, error) {
	deliveryID, err := s.utils.ParseID(id)
	if err != nil {
		return DeliveryResponse{}, err
	}

	delivery, err := s.repo.Transition(ctx, deliveryID, status)
	if err != nil {
		return DeliveryResponse{}, err
	}

	return s.utils.ToResponse(delivery), nil
}

func (s *service) parseLines(lines []DeliveryLineRequest) ([]Line, error) {
	seen := make(map[uuid.UUID]bool, len(lines))
	result := make([]Line, 0, len(lines))
	for _, line := range lines {
		materialID, err := s.utils.ParseID(line.MaterialID)
		if err != nil || seen[materialID] {
			return nil, ErrInvalidInput
		}
		seen[materialID] = true
		result = append(result, Line{MaterialID: materialID, Quantity: line.Quantity})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].MaterialID.String() < result[j].MaterialID.String()
	})
	return result, nil
}

func validStatus(status string) bool {
	switch status {
	case "", StatusInTransit, StatusDelivered, StatusRejected, StatusCancelled:
		return true
	default:
		return false
	}
}
