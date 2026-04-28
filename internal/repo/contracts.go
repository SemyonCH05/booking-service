package repo

import (
	"context"
	"room-booking-service/internal/entity"
	"time"

	"github.com/google/uuid"
)

//go:generate mockgen -source=contracts.go -destination=mocks/mock.go
type Repo interface {
	CreateRoom(ctx context.Context, room *entity.Room) (uuid.UUID, *time.Time, error)
	CreateSchedule(ctx context.Context, schedule *entity.Schedule) (uuid.UUID, error)
	CreateBooking(ctx context.Context, booking *entity.Booking) (uuid.UUID, *time.Time, error)
	AddSlots(ctx context.Context) error
	IsSlotInPast(ctx context.Context, slotId uuid.UUID) (bool, error)
	CancelBooking(ctx context.Context, userId, bookingId uuid.UUID) (*entity.Booking, error)
	GetRooms(ctx context.Context) ([]*entity.Room, error)
	IsRoomExist(ctx context.Context, roomId uuid.UUID) (bool, error)
	GetSlots(ctx context.Context, roomId uuid.UUID, date time.Time) ([]*entity.Slot, error)
	GetBookings(ctx context.Context, page, pageSize int) ([]*entity.Booking, int, error)
	GetBookingsUser(ctx context.Context, userId uuid.UUID) ([]*entity.Booking, error)
}
