package usecase

import (
	"context"
	"room-booking-service/internal/entity"
	"time"

	"github.com/google/uuid"
)

//go:generate mockgen -source=contracts.go -destination=mocks/mock.go
type BookingService interface {
	AddSlots(ctx context.Context)
	CreateRoom(ctx context.Context, room *entity.Room) (*entity.Room, error)
	CreateSchedule(ctx context.Context, schedule *entity.Schedule) (uuid.UUID, error)
	IsSlotInPast(ctx context.Context, slotId uuid.UUID) (bool, error)
	CreateBooking(ctx context.Context, booking *entity.Booking) (*entity.Booking, error)
	CancelBooking(ctx context.Context, userId, bookingId uuid.UUID) (*entity.Booking, error)
	GetRooms(ctx context.Context) ([]*entity.Room, error)
	IsRoomExist(ctx context.Context, roomId uuid.UUID) (bool, error)
	GetSlots(ctx context.Context, roomId uuid.UUID, date time.Time) ([]*entity.Slot, error)
	GetBookings(ctx context.Context, page, pageSize int) ([]*entity.Booking, int, error)
	GetBookingsUser(ctx context.Context, userId uuid.UUID) ([]*entity.Booking, error)
}
