package response

import (
	"room-booking-service/internal/entity"

	"github.com/google/uuid"
)

type RoomResponse struct {
	Room *entity.Room `json:"room"`
}

type RoomsResponse struct {
	Rooms []*entity.Room `json:"rooms"`
}

type SlotsResponse struct {
	Slots []*entity.Slot `json:"slots"`
}

type ScheduleDetail struct {
	Id         uuid.UUID `json:"id"`
	RoomId     uuid.UUID `json:"roomId"`
	DaysOfWeek []int     `json:"daysOfWeek"`
	StartTime  string    `json:"startTime"`
	EndTime    string    `json:"endTime"`
}

type ScheduleResponse struct {
	Schedule ScheduleDetail `json:"schedule"`
}

type BookingResponse struct {
	Booking *entity.Booking `json:"booking"`
}

type BookingsResponse struct {
	Bookings   []*entity.Booking `json:"bookings"`
	Pagination entity.Pagination `json:"pagination"`
}

type BookingsUserResponse struct {
	Bookings []*entity.Booking `json:"bookings"`
}
