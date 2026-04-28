package booking

import (
	"context"
	"errors"
	"room-booking-service/internal/entity"
	"room-booking-service/internal/repo"
	"room-booking-service/pkg/logger"
	"time"

	"github.com/google/uuid"
)

type BookingService struct {
	repository repo.Repo
	l          logger.Interface
}

func New(repo repo.Repo, l logger.Interface) *BookingService {
	return &BookingService{
		repository: repo,
		l:          l,
	}
}

func (s *BookingService) CreateRoom(ctx context.Context, room *entity.Room) (*entity.Room, error) {
	id, createdAt, err := s.repository.CreateRoom(ctx, room)
	room.Id = id
	room.CreatedAt = createdAt
	if err != nil {
		return room, errors.Join(errors.New("bookingService - CreateRoom - repository.CreateRoom"), err)
	}
	return room, nil
}

func (s *BookingService) AddSlots(ctx context.Context) {
	err := s.repository.AddSlots(ctx)
	if err != nil {
		s.l.Error(errors.Join(errors.New("bookingService - AddSlots - before ticker repository.AddSlots"), err))
	}
	ticker := time.NewTicker(time.Hour * 24)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.repository.AddSlots(ctx); err != nil {
				s.l.Error(errors.Join(errors.New("bookingService - AddSlots - repository.AddSlots"), err))
			}
		}
	}
}

func (s *BookingService) CreateSchedule(ctx context.Context, schedule *entity.Schedule) (uuid.UUID, error) {
	id, err := s.repository.CreateSchedule(ctx, schedule)
	if err != nil {
		return uuid.Nil, errors.Join(errors.New("bookingService - CreateSchedule - repository.CreateSchedule"), err)
	}
	return id, nil
}

func (s *BookingService) IsSlotInPast(ctx context.Context, slotId uuid.UUID) (bool, error) {
	flag, err := s.repository.IsSlotInPast(ctx, slotId)
	if err != nil {
		return false, errors.Join(errors.New("bookingService - IsSlotInPast - repository.IsSlotInPast"), err)
	}
	return flag, nil
}

func (s *BookingService) CreateBooking(ctx context.Context, booking *entity.Booking) (*entity.Booking, error) {
	id, createdAt, err := s.repository.CreateBooking(ctx, booking)
	if err != nil {
		return nil, errors.Join(errors.New("bookingService - CreateBooking - repository.CreateBooking"), err)
	}
	booking.Id = id
	booking.CreatedAt = *createdAt
	booking.Status = entity.Active
	booking.ConferenceLink = nil
	return booking, nil
}

func (s *BookingService) CancelBooking(ctx context.Context, userId, bookingId uuid.UUID) (*entity.Booking, error) {
	booking, err := s.repository.CancelBooking(ctx, userId, bookingId)
	if err != nil {
		return nil, errors.Join(errors.New("bookingService - CancelBooking - repository.CancelBooking"), err)
	}
	return booking, nil
}

func (s *BookingService) GetRooms(ctx context.Context) ([]*entity.Room, error) {
	rooms, err := s.repository.GetRooms(ctx)
	if err != nil {
		return nil, errors.Join(errors.New("bookingService - GetRooms - repository.GetRooms"), err)
	}
	return rooms, nil
}

func (s *BookingService) IsRoomExist(ctx context.Context, roomId uuid.UUID) (bool, error) {
	isExist, err := s.repository.IsRoomExist(ctx, roomId)
	if err != nil {
		return false, errors.Join(errors.New("bookingService - IsRoomExist - repository.IsRoomExist"), err)
	}
	return isExist, nil
}

func (s *BookingService) GetSlots(ctx context.Context, roomId uuid.UUID, date time.Time) ([]*entity.Slot, error) {
	slots, err := s.repository.GetSlots(ctx, roomId, date)
	if err != nil {
		return nil, errors.Join(errors.New("bookingService - GetSlots - repository.GetSlots"), err)
	}
	return slots, nil
}

func (s *BookingService) GetBookings(ctx context.Context, page, pageSize int) ([]*entity.Booking, int, error) {
	bookings, total, err := s.repository.GetBookings(ctx, page, pageSize)
	if err != nil {
		return nil, -1, errors.Join(errors.New("bookingService - GetBookings - repository.GetBookings"), err)
	}
	return bookings, total, nil
}

func (s *BookingService) GetBookingsUser(ctx context.Context, userId uuid.UUID) ([]*entity.Booking, error) {
	bookings, err := s.repository.GetBookingsUser(ctx, userId)
	if err != nil {
		return nil, errors.Join(errors.New("bookingService - GetBookingsUser - repository.GetBookingsUser"), err)
	}
	return bookings, nil
}
