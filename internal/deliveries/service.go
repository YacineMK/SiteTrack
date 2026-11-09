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
	CreateFromCompany(ctx context.Context, actorID uuid.UUID, req CreateCompanyDeliveryRequest) (DeliveryResponse, error)
	CreateFromProject(ctx context.Context, projectDisplayID string, actorID uuid.UUID, req CreateProjectDeliveryRequest) (DeliveryResponse, error)
	ListCompany(ctx context.Context, status string, limit, offset int32) (ListDeliveriesResponse, error)
	ListProject(ctx context.Context, projectDisplayID, status string, limit, offset int32) (ListDeliveriesResponse, error)
	Get(ctx context.Context, actorID uuid.UUID, role, id string) (DeliveryResponse, error)
	Record(ctx context.Context, actorID uuid.UUID, role, id string) (DeliveryResponse, error)
	Reject(ctx context.Context, actorID uuid.UUID, role, id string) (DeliveryResponse, error)
	Cancel(ctx context.Context, actorID uuid.UUID, role, id string) (DeliveryResponse, error)
}

type service struct {
	repo     Repository
	projects projects.Repository
	utils    Utils
}

func NewService(repo Repository, projectRepo projects.Repository) Service {
	return &service{repo: repo, projects: projectRepo, utils: NewUtils()}
}

func (s *service) CreateFromCompany(ctx context.Context, actorID uuid.UUID, req CreateCompanyDeliveryRequest) (DeliveryResponse, error) {
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
		SentBy:       &actorID,
		Lines:        lines,
	})
	if err != nil {
		return DeliveryResponse{}, err
	}

	return s.utils.ToResponse(delivery), nil
}

func (s *service) CreateFromProject(ctx context.Context, projectDisplayID string, actorID uuid.UUID, req CreateProjectDeliveryRequest) (DeliveryResponse, error) {
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
		SentBy:       &actorID,
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

func (s *service) Get(ctx context.Context, actorID uuid.UUID, role, id string) (DeliveryResponse, error) {
	deliveryID, err := s.utils.ParseID(id)
	if err != nil {
		return DeliveryResponse{}, err
	}

	delivery, err := s.repo.Get(ctx, deliveryID)
	if err != nil {
		return DeliveryResponse{}, err
	}

	sender, receiver, err := s.sides(ctx, delivery, actorID, role)
	if err != nil {
		return DeliveryResponse{}, err
	}
	if !sender && !receiver {
		return DeliveryResponse{}, ErrNotFound
	}

	return s.utils.ToResponse(delivery), nil
}

func (s *service) Record(ctx context.Context, actorID uuid.UUID, role, id string) (DeliveryResponse, error) {
	return s.transition(ctx, actorID, role, id, StatusDelivered, true)
}

func (s *service) Reject(ctx context.Context, actorID uuid.UUID, role, id string) (DeliveryResponse, error) {
	return s.transition(ctx, actorID, role, id, StatusRejected, true)
}

func (s *service) Cancel(ctx context.Context, actorID uuid.UUID, role, id string) (DeliveryResponse, error) {
	return s.transition(ctx, actorID, role, id, StatusCancelled, false)
}

func (s *service) transition(ctx context.Context, actorID uuid.UUID, role, id, status string, receiverAction bool) (DeliveryResponse, error) {
	deliveryID, err := s.utils.ParseID(id)
	if err != nil {
		return DeliveryResponse{}, err
	}

	delivery, err := s.repo.Get(ctx, deliveryID)
	if err != nil {
		return DeliveryResponse{}, err
	}

	sender, receiver, err := s.sides(ctx, delivery, actorID, role)
	if err != nil {
		return DeliveryResponse{}, err
	}
	if receiverAction && !receiver {
		return DeliveryResponse{}, ErrNotFound
	}
	if !receiverAction && !sender {
		return DeliveryResponse{}, ErrNotFound
	}

	var receivedBy *uuid.UUID
	if receiverAction {
		receivedBy = &actorID
	}

	updated, err := s.repo.Transition(ctx, deliveryID, status, receivedBy)
	if err != nil {
		return DeliveryResponse{}, err
	}

	return s.utils.ToResponse(updated), nil
}

func (s *service) sides(ctx context.Context, delivery Delivery, userID uuid.UUID, role string) (bool, bool, error) {
	if role == "super_admin" {
		return true, true, nil
	}

	var sender, receiver bool
	if delivery.SenderType == PartyCompany {
		sender = role == "company_store_manager"
	} else if delivery.SenderID != nil {
		member, _, err := s.projects.Membership(ctx, *delivery.SenderID, userID)
		if err != nil {
			return false, false, err
		}
		sender = member
	}

	if delivery.ReceiverType == PartyCompany {
		receiver = role == "company_store_manager"
	} else if delivery.ReceiverID != nil {
		member, _, err := s.projects.Membership(ctx, *delivery.ReceiverID, userID)
		if err != nil {
			return false, false, err
		}
		receiver = member
	}

	return sender, receiver, nil
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
