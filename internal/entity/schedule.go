package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type DayOfWeek int

const (
	Mo DayOfWeek = 1
	Tu DayOfWeek = 2
	We DayOfWeek = 3
	Th DayOfWeek = 4
	Fr DayOfWeek = 5
	Sa DayOfWeek = 6
	Su DayOfWeek = 7
)

type Schedule struct {
	Id         uuid.UUID   `json:"id"`
	RoomId     uuid.UUID   `json:"roomId"`
	DaysOfWeek []DayOfWeek `json:"daysOfWeek"`
	StartTime  time.Time   `json:"startTime"` // HH:MM
	EndTime    time.Time   `json:"endTime"`   // HH:MM
}

var ErrScheduleExists = errors.New("schedule for room already exists")
