package deliveries

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
)

type DeliveryLineRequest struct {
	MaterialID string  `json:"material_id"`
	Quantity   float64 `json:"quantity"`
}

func (l DeliveryLineRequest) Validate() error {
	return validation.ValidateStruct(&l,
		validation.Field(&l.MaterialID, validation.Required, is.UUID),
		validation.Field(&l.Quantity, validation.Required, validation.Min(0.001)),
	)
}

type CreateCompanyDeliveryRequest struct {
	ReceiverProject string                `json:"receiver_project"`
	Lines           []DeliveryLineRequest `json:"lines"`
}

func (r CreateCompanyDeliveryRequest) Validate() error {
	if err := validation.ValidateStruct(&r,
		validation.Field(&r.ReceiverProject, validation.Required),
		validation.Field(&r.Lines, validation.Required, validation.Length(1, 100)),
	); err != nil {
		return err
	}
	return validateLines(r.Lines)
}

type CreateProjectDeliveryRequest struct {
	Lines []DeliveryLineRequest `json:"lines"`
}

func (r CreateProjectDeliveryRequest) Validate() error {
	if err := validation.ValidateStruct(&r,
		validation.Field(&r.Lines, validation.Required, validation.Length(1, 100)),
	); err != nil {
		return err
	}
	return validateLines(r.Lines)
}

func validateLines(lines []DeliveryLineRequest) error {
	for _, line := range lines {
		if err := line.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type DeliveryLineResponse struct {
	MaterialID uuid.UUID `json:"material_id"`
	Name       string    `json:"name"`
	Unit       string    `json:"unit"`
	Quantity   float64   `json:"quantity"`
}

type DeliveryResponse struct {
	ID                uuid.UUID              `json:"id"`
	SenderType        string                 `json:"sender_type"`
	SenderID          *uuid.UUID             `json:"sender_id"`
	SenderDisplayID   string                 `json:"sender_display_id,omitempty"`
	ReceiverType      string                 `json:"receiver_type"`
	ReceiverID        *uuid.UUID             `json:"receiver_id"`
	ReceiverDisplayID string                 `json:"receiver_display_id,omitempty"`
	Status            string                 `json:"status"`
	SentAt            time.Time              `json:"sent_at"`
	ReceivedAt        *time.Time             `json:"received_at"`
	SentBy            *uuid.UUID             `json:"sent_by"`
	ReceivedBy        *uuid.UUID             `json:"received_by"`
	SentByName        string                 `json:"sent_by_name,omitempty"`
	ReceivedByName    string                 `json:"received_by_name,omitempty"`
	Lines             []DeliveryLineResponse `json:"lines,omitempty"`
}

type ListDeliveriesResponse struct {
	Items []DeliveryResponse `json:"items"`
	Total int64              `json:"total"`
	Limit int32              `json:"limit"`
}
