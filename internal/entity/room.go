package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Room struct {
	Id          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Capacity    int        `json:"capacity"`
	CreatedAt   *time.Time `json:"createdAt"`
}

var ErrRoom = errors.New("room is not exists")
