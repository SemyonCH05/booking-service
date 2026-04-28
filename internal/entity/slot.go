package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Slot struct {
	Id     uuid.UUID `json:"id"`
	RoomId uuid.UUID `json:"roomId"`
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
}

var SlotNotFound = errors.New("Slot not found")
var SlotIsBusy = errors.New("Slot is busy")
