package request

import (
	"github.com/google/uuid"
)

type AuthRequest struct {
	Role string `json:"role" validate:"required"`
}

type RoomRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"required"`
	Capacity    int    `json:"capacity" validate:"required"`
}

type ScheduleRequest struct {
	Id         uuid.UUID `json:"id" validate:"required"`
	RoomId     uuid.UUID `json:"roomId" validate:"required"`
	DaysOfWeek []int     `json:"daysOfWeek" validate:"required"`
	StartTime  string    `json:"startTime" validate:"required"` // HH:MM
	EndTime    string    `json:"endTime" validate:"required"`   // HH:MM
}

type BookingRequest struct {
	SlotId               uuid.UUID `json:"slotId" validate:"required"`
	CreateConferenceLink bool      `json:"createConferenceLink"`
}
