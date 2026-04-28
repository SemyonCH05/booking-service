package booking

import (
	"room-booking-service/internal/entity"
	mock_repo "room-booking-service/internal/repo/mocks"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockBehaviorRepo func(s *mock_repo.MockRepo)

type TestCreateRoom struct {
	name             string
	room             *entity.Room
	mockBehaviorRepo mockBehaviorRepo
	expectedRoom     *entity.Room
	expectedError    error
}

func TestService_CreateRoom(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	roomID := uuid.New()

	tests := []TestCreateRoom{
		{
			name: "OK",
			room: &entity.Room{
				Name:        "Room A",
				Description: "desc",
				Capacity:    8,
			},
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					CreateRoom(gomock.Any(), gomock.AssignableToTypeOf(&entity.Room{})).
					Return(roomID, &now, nil)
			},
			expectedRoom: &entity.Room{
				Id:          roomID,
				Name:        "Room A",
				Description: "desc",
				Capacity:    8,
				CreatedAt:   &now,
			},
			expectedError: nil,
		},
		{
			name: "RepoError",
			room: &entity.Room{
				Name:        "Room B",
				Description: "desc",
				Capacity:    4,
			},
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					CreateRoom(gomock.Any(), gomock.AssignableToTypeOf(&entity.Room{})).
					Return(uuid.Nil, nil, assert.AnError)
			},
			expectedRoom: &entity.Room{
				Id:          uuid.Nil,
				Name:        "Room B",
				Description: "desc",
				Capacity:    4,
				CreatedAt:   nil,
			},
			expectedError: assert.AnError,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			mockRepo := mock_repo.NewMockRepo(c)
			testCase.mockBehaviorRepo(mockRepo)

			service := New(mockRepo, nil)

			result, err := service.CreateRoom(t.Context(), testCase.room)
			assert.Equal(t, testCase.expectedRoom, result)
			if testCase.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type TestCreateSchedule struct {
	name             string
	schedule         *entity.Schedule
	mockBehaviorRepo mockBehaviorRepo
	expectedID       uuid.UUID
	expectedError    error
}

func TestService_CreateSchedule(t *testing.T) {
	scheduleID := uuid.New()
	roomID := uuid.New()

	tests := []TestCreateSchedule{
		{
			name: "OK",
			schedule: &entity.Schedule{
				RoomId:     roomID,
				DaysOfWeek: []entity.DayOfWeek{entity.Mo, entity.We},
				StartTime:  time.Date(1, 1, 1, 9, 0, 0, 0, time.UTC),
				EndTime:    time.Date(1, 1, 1, 18, 0, 0, 0, time.UTC),
			},
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					CreateSchedule(gomock.Any(), gomock.AssignableToTypeOf(&entity.Schedule{})).
					Return(scheduleID, nil)
			},
			expectedID:    scheduleID,
			expectedError: nil,
		},
		{
			name: "RepoError",
			schedule: &entity.Schedule{
				RoomId: roomID,
			},
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					CreateSchedule(gomock.Any(), gomock.AssignableToTypeOf(&entity.Schedule{})).
					Return(uuid.Nil, assert.AnError)
			},
			expectedID:    uuid.Nil,
			expectedError: assert.AnError,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			mockRepo := mock_repo.NewMockRepo(c)
			testCase.mockBehaviorRepo(mockRepo)

			service := New(mockRepo, nil)

			id, err := service.CreateSchedule(t.Context(), testCase.schedule)
			assert.Equal(t, testCase.expectedID, id)
			if testCase.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type TestGetSlots struct {
	name             string
	roomID           uuid.UUID
	date             time.Time
	mockBehaviorRepo mockBehaviorRepo
	expectedSlots    []*entity.Slot
	expectedError    error
}

func TestService_GetSlots(t *testing.T) {
	roomID := uuid.New()
	date := time.Now().UTC().Truncate(24 * time.Hour)

	expectedSlots := []*entity.Slot{
		{
			Id:     uuid.New(),
			RoomId: roomID,
			Start:  date.Add(9 * time.Hour),
			End:    date.Add(9*time.Hour + 30*time.Minute),
		},
		{
			Id:     uuid.New(),
			RoomId: roomID,
			Start:  date.Add(10 * time.Hour),
			End:    date.Add(10*time.Hour + 30*time.Minute),
		},
	}

	tests := []TestGetSlots{
		{
			name:   "OK",
			roomID: roomID,
			date:   date,
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					GetSlots(gomock.Any(), roomID, date).
					Return(expectedSlots, nil)
			},
			expectedSlots: expectedSlots,
			expectedError: nil,
		},
		{
			name:   "RepoError",
			roomID: roomID,
			date:   date,
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					GetSlots(gomock.Any(), roomID, date).
					Return(nil, assert.AnError)
			},
			expectedSlots: nil,
			expectedError: assert.AnError,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			mockRepo := mock_repo.NewMockRepo(c)
			testCase.mockBehaviorRepo(mockRepo)

			service := New(mockRepo, nil)

			slots, err := service.GetSlots(t.Context(), testCase.roomID, testCase.date)
			assert.Equal(t, testCase.expectedSlots, slots)
			if testCase.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type TestCancelBooking struct {
	name             string
	userID           uuid.UUID
	bookingID        uuid.UUID
	mockBehaviorRepo mockBehaviorRepo
	expectedBooking  *entity.Booking
	expectedError    error
}

func TestService_CancelBooking(t *testing.T) {
	userID := uuid.New()
	bookingID := uuid.New()
	slotID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	expectedBooking := &entity.Booking{
		Id:             bookingID,
		SlotId:         slotID,
		UserId:         userID,
		Status:         entity.Cancelled,
		ConferenceLink: nil,
		CreatedAt:      now,
	}

	tests := []TestCancelBooking{
		{
			name:      "OK",
			userID:    userID,
			bookingID: bookingID,
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					CancelBooking(gomock.Any(), userID, bookingID).
					Return(expectedBooking, nil)
			},
			expectedBooking: expectedBooking,
			expectedError:   nil,
		},
		{
			name:      "RepoError",
			userID:    userID,
			bookingID: bookingID,
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					CancelBooking(gomock.Any(), userID, bookingID).
					Return(nil, assert.AnError)
			},
			expectedBooking: nil,
			expectedError:   assert.AnError,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			mockRepo := mock_repo.NewMockRepo(c)
			testCase.mockBehaviorRepo(mockRepo)

			service := New(mockRepo, nil)

			result, err := service.CancelBooking(t.Context(), testCase.userID, testCase.bookingID)
			assert.Equal(t, testCase.expectedBooking, result)
			if testCase.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type TestGetBookings struct {
	name             string
	page             int
	pageSize         int
	mockBehaviorRepo mockBehaviorRepo
	expectedBookings []*entity.Booking
	expectedTotal    int
	expectedError    error
}

func TestService_GetBookings(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	expectedBookings := []*entity.Booking{
		{
			Id:        uuid.New(),
			SlotId:    uuid.New(),
			UserId:    uuid.New(),
			Status:    entity.Active,
			CreatedAt: now,
		},
		{
			Id:        uuid.New(),
			SlotId:    uuid.New(),
			UserId:    uuid.New(),
			Status:    entity.Cancelled,
			CreatedAt: now,
		},
	}

	tests := []TestGetBookings{
		{
			name:     "OK",
			page:     1,
			pageSize: 20,
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					GetBookings(gomock.Any(), 1, 20).
					Return(expectedBookings, 2, nil)
			},
			expectedBookings: expectedBookings,
			expectedTotal:    2,
			expectedError:    nil,
		},
		{
			name:     "RepoError",
			page:     1,
			pageSize: 20,
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					GetBookings(gomock.Any(), 1, 20).
					Return(nil, -1, assert.AnError)
			},
			expectedBookings: nil,
			expectedTotal:    -1,
			expectedError:    assert.AnError,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			mockRepo := mock_repo.NewMockRepo(c)
			testCase.mockBehaviorRepo(mockRepo)

			service := New(mockRepo, nil)

			bookings, total, err := service.GetBookings(t.Context(), testCase.page, testCase.pageSize)
			assert.Equal(t, testCase.expectedBookings, bookings)
			assert.Equal(t, testCase.expectedTotal, total)
			if testCase.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type TestGetRooms struct {
	name             string
	mockBehaviorRepo mockBehaviorRepo
	expectedRooms   []*entity.Room
	expectedError   error
}

func TestService_GetRooms(t *testing.T) {
	room1 := &entity.Room{
		Id:          uuid.New(),
		Name:        "Room 1",
		Description: "desc 1",
		Capacity:    10,
	}
	room2 := &entity.Room{
		Id:          uuid.New(),
		Name:        "Room 2",
		Description: "desc 2",
		Capacity:    20,
	}

	tests := []TestGetRooms{
		{
			name: "OK",
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					GetRooms(gomock.Any()).
					Return([]*entity.Room{room1, room2}, nil)
			},
			expectedRooms: []*entity.Room{room1, room2},
			expectedError: nil,
		},
		{
			name: "Empty",
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					GetRooms(gomock.Any()).
					Return([]*entity.Room{}, nil)
			},
			expectedRooms: []*entity.Room{},
			expectedError: nil,
		},
		{
			name: "RepoError",
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					GetRooms(gomock.Any()).
					Return(nil, assert.AnError)
			},
			expectedRooms: nil,
			expectedError: assert.AnError,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			mockRepo := mock_repo.NewMockRepo(c)
			testCase.mockBehaviorRepo(mockRepo)

			service := New(mockRepo, nil)

			rooms, err := service.GetRooms(t.Context())
			assert.Equal(t, testCase.expectedRooms, rooms)
			if testCase.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type TestCreateBooking struct {
	name             string
	booking          *entity.Booking
	mockBehaviorRepo mockBehaviorRepo
	expectedBooking  *entity.Booking
	expectedError    error
}

func TestService_CreateBooking(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	bookingID := uuid.New()
	slotID := uuid.New()
	userID := uuid.New()

	tests := []TestCreateBooking{
		{
			name: "OK",
			booking: &entity.Booking{
				SlotId: slotID,
				UserId: userID,
			},
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					CreateBooking(gomock.Any(), gomock.AssignableToTypeOf(&entity.Booking{})).
					Return(bookingID, &now, nil)
			},
			expectedBooking: &entity.Booking{
				Id:             bookingID,
				SlotId:         slotID,
				UserId:         userID,
				Status:         entity.Active,
				ConferenceLink: nil,
				CreatedAt:      now,
			},
			expectedError: nil,
		},
		{
			name: "RepoError",
			booking: &entity.Booking{
				SlotId: slotID,
				UserId: userID,
			},
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					CreateBooking(gomock.Any(), gomock.AssignableToTypeOf(&entity.Booking{})).
					Return(uuid.Nil, nil, assert.AnError)
			},
			expectedBooking: nil,
			expectedError:   assert.AnError,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			mockRepo := mock_repo.NewMockRepo(c)
			testCase.mockBehaviorRepo(mockRepo)

			service := New(mockRepo, nil)

			result, err := service.CreateBooking(t.Context(), testCase.booking)
			assert.Equal(t, testCase.expectedBooking, result)
			if testCase.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type TestIsSlotInPast struct {
	name             string
	slotID           uuid.UUID
	mockBehaviorRepo mockBehaviorRepo
	expectedFlag     bool
	expectedError    error
}

func TestService_IsSlotInPast(t *testing.T) {
	slotID := uuid.New()

	tests := []TestIsSlotInPast{
		{
			name:   "OK_True",
			slotID: slotID,
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					IsSlotInPast(gomock.Any(), slotID).
					Return(true, nil)
			},
			expectedFlag:  true,
			expectedError: nil,
		},
		{
			name:   "OK_False",
			slotID: slotID,
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					IsSlotInPast(gomock.Any(), slotID).
					Return(false, nil)
			},
			expectedFlag:  false,
			expectedError: nil,
		},
		{
			name:   "RepoError",
			slotID: slotID,
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					IsSlotInPast(gomock.Any(), slotID).
					Return(false, assert.AnError)
			},
			expectedFlag:  false,
			expectedError: assert.AnError,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			mockRepo := mock_repo.NewMockRepo(c)
			testCase.mockBehaviorRepo(mockRepo)

			service := New(mockRepo, nil)

			flag, err := service.IsSlotInPast(t.Context(), testCase.slotID)
			assert.Equal(t, testCase.expectedFlag, flag)
			if testCase.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type TestIsRoomExist struct {
	name             string
	roomID           uuid.UUID
	mockBehaviorRepo mockBehaviorRepo
	expectedFlag     bool
	expectedError    error
}

func TestService_IsRoomExist(t *testing.T) {
	roomID := uuid.New()

	tests := []TestIsRoomExist{
		{
			name:   "OK_True",
			roomID: roomID,
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					IsRoomExist(gomock.Any(), roomID).
					Return(true, nil)
			},
			expectedFlag:  true,
			expectedError: nil,
		},
		{
			name:   "OK_False",
			roomID: roomID,
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					IsRoomExist(gomock.Any(), roomID).
					Return(false, nil)
			},
			expectedFlag:  false,
			expectedError: nil,
		},
		{
			name:   "RepoError",
			roomID: roomID,
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					IsRoomExist(gomock.Any(), roomID).
					Return(false, assert.AnError)
			},
			expectedFlag:  false,
			expectedError: assert.AnError,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			mockRepo := mock_repo.NewMockRepo(c)
			testCase.mockBehaviorRepo(mockRepo)

			service := New(mockRepo, nil)

			flag, err := service.IsRoomExist(t.Context(), testCase.roomID)
			assert.Equal(t, testCase.expectedFlag, flag)
			if testCase.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type TestGetBookingsUser struct {
	name             string
	userID           uuid.UUID
	mockBehaviorRepo mockBehaviorRepo
	expectedBookings []*entity.Booking
	expectedError    error
}

func TestService_GetBookingsUser(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	expectedBookings := []*entity.Booking{
		{
			Id:        uuid.New(),
			SlotId:    uuid.New(),
			UserId:    userID,
			Status:    entity.Active,
			CreatedAt: now,
		},
		{
			Id:        uuid.New(),
			SlotId:    uuid.New(),
			UserId:    userID,
			Status:    entity.Cancelled,
			CreatedAt: now,
		},
	}

	tests := []TestGetBookingsUser{
		{
			name:   "OK",
			userID: userID,
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					GetBookingsUser(gomock.Any(), userID).
					Return(expectedBookings, nil)
			},
			expectedBookings: expectedBookings,
			expectedError:    nil,
		},
		{
			name:   "Empty",
			userID: userID,
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					GetBookingsUser(gomock.Any(), userID).
					Return([]*entity.Booking{}, nil)
			},
			expectedBookings: []*entity.Booking{},
			expectedError:    nil,
		},
		{
			name:   "RepoError",
			userID: userID,
			mockBehaviorRepo: func(s *mock_repo.MockRepo) {
				s.EXPECT().
					GetBookingsUser(gomock.Any(), userID).
					Return(nil, assert.AnError)
			},
			expectedBookings: nil,
			expectedError:    assert.AnError,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			mockRepo := mock_repo.NewMockRepo(c)
			testCase.mockBehaviorRepo(mockRepo)

			service := New(mockRepo, nil)

			bookings, err := service.GetBookingsUser(t.Context(), testCase.userID)
			assert.Equal(t, testCase.expectedBookings, bookings)
			if testCase.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
