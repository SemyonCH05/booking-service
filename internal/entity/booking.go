package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	Active    Status = "active"
	Cancelled Status = "cancelled"
)

type Booking struct {
	Id             uuid.UUID `json:"id"`
	SlotId         uuid.UUID `json:"slotId"`
	UserId         uuid.UUID `json:"userId"`
	Status         Status    `json:"status"`
	ConferenceLink *string   `json:"conferenceLink"`
	CreatedAt      time.Time `json:"createdAt"`
}

var OtherUserBooking = errors.New("other user booking")
var BookingNotFound = errors.New("booking not found")
