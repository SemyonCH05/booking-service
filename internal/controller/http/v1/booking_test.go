package v1

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"room-booking-service/internal/controller/http/v1/response"
	"room-booking-service/internal/entity"
	mock_usecase "room-booking-service/internal/usecase/mocks"
	"room-booking-service/pkg/logger"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type nopLogger struct{}

var _ logger.Interface = (*nopLogger)(nil)

func (n *nopLogger) Debug(message interface{}, args ...interface{}) {}
func (n *nopLogger) Info(message string, args ...interface{})       {}
func (n *nopLogger) Warn(message string, args ...interface{})       {}
func (n *nopLogger) Error(message interface{}, args ...interface{}) {}
func (n *nopLogger) Fatal(message interface{}, args ...interface{}) {}

func newV1ForTest(b *mock_usecase.MockBookingService) *V1 {
	return &V1{
		b:      b,
		l:      &nopLogger{},
		v:      validator.New(validator.WithRequiredStructEnabled()),
		secret: "test-secret",
	}
}

func doJSON(app *fiber.App, method, path string, body any) (*http.Response, []byte) {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	if err != nil {
		panic(err)
	}
	b, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	return res, b
}

func doRaw(app *fiber.App, method, path string, raw string) (*http.Response, []byte) {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(raw))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	if err != nil {
		panic(err)
	}
	b, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	return res, b
}

func decodeError(t *testing.T, body []byte) response.ErrorResponse {
	t.Helper()
	var er response.ErrorResponse
	_ = json.Unmarshal(body, &er)
	return er
}

func TestHandlers_GetInfo(t *testing.T) {
	app := fiber.New()
	r := newV1ForTest(nil)
	app.Get("/_info", r.getInfo)

	res, _ := doJSON(app, http.MethodGet, "/_info", nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestHandlers_Auth(t *testing.T) {
	app := fiber.New()
	c := gomock.NewController(t)
	defer c.Finish()

	mockB := mock_usecase.NewMockBookingService(c)
	r := newV1ForTest(mockB)
	app.Post("/dummyLogin", r.auth)

	type tc struct {
		name       string
		rawBody    string
		jsonBody   any
		wantStatus int
	}

	tests := []tc{
		{name: "BadJSON", rawBody: "{", wantStatus: http.StatusBadRequest},
		{name: "BadRole", jsonBody: map[string]any{"role": "guest"}, wantStatus: http.StatusBadRequest},
		{name: "OK_User", jsonBody: map[string]any{"role": "user"}, wantStatus: http.StatusOK},
		{name: "OK_Admin", jsonBody: map[string]any{"role": "admin"}, wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var res *http.Response
			var body []byte
			if tt.rawBody != "" {
				res, body = doRaw(app, http.MethodPost, "/dummyLogin", tt.rawBody)
			} else {
				res, body = doJSON(app, http.MethodPost, "/dummyLogin", tt.jsonBody)
			}
			assert.Equal(t, tt.wantStatus, res.StatusCode)
			if tt.wantStatus != http.StatusOK {
				er := decodeError(t, body)
				assert.NotEmpty(t, er.ErrorDetail.Code)
				return
			}
			var resp map[string]string
			_ = json.Unmarshal(body, &resp)
			assert.NotEmpty(t, resp["token"])
		})
	}
}

func TestHandlers_GetRooms(t *testing.T) {
	rooms := []*entity.Room{
		{Id: uuid.New(), Name: "A", Description: "d", Capacity: 1},
		{Id: uuid.New(), Name: "B", Description: "d2", Capacity: 2},
	}

	type tc struct {
		name       string
		mock       func(b *mock_usecase.MockBookingService)
		wantStatus int
	}

	tests := []tc{
		{
			name: "OK",
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().GetRooms(gomock.Any()).Return(rooms, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "ServiceError",
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().GetRooms(gomock.Any()).Return(nil, assert.AnError)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			mockB := mock_usecase.NewMockBookingService(c)
			r := newV1ForTest(mockB)
			app := fiber.New()
			app.Get("/rooms/list", r.getRooms)

			tt.mock(mockB)
			res, body := doJSON(app, http.MethodGet, "/rooms/list", nil)
			assert.Equal(t, tt.wantStatus, res.StatusCode)
			if tt.wantStatus != http.StatusOK {
				er := decodeError(t, body)
				assert.Equal(t, string(response.InternalError), er.ErrorDetail.Code)
				return
			}
			var resp struct {
				Rooms []*entity.Room `json:"rooms"`
			}
			_ = json.Unmarshal(body, &resp)
			assert.Equal(t, rooms, resp.Rooms)
		})
	}
}

func TestHandlers_CreateRoom(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	roomID := uuid.New()
	createdRoom := &entity.Room{
		Id:          roomID,
		Name:        "Room",
		Description: "Desc",
		Capacity:    3,
		CreatedAt:   &now,
	}

	tests := []struct {
		name       string
		rawBody    string
		jsonBody   any
		mock       func(b *mock_usecase.MockBookingService)
		wantStatus int
		wantCode   string
	}{
		{
			name:       "BadJSON",
			rawBody:    "{",
			mock:       func(b *mock_usecase.MockBookingService) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:       "ValidationError",
			jsonBody:   map[string]any{"name": "", "description": "d", "capacity": 1},
			mock:       func(b *mock_usecase.MockBookingService) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:     "OK",
			jsonBody: map[string]any{"name": "Room", "description": "Desc", "capacity": 3},
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().
					CreateRoom(gomock.Any(), gomock.Any()).
					Return(createdRoom, nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:     "ServiceError",
			jsonBody: map[string]any{"name": "Room", "description": "Desc", "capacity": 3},
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().CreateRoom(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   string(response.InternalError),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			mockB := mock_usecase.NewMockBookingService(c)
			r := newV1ForTest(mockB)
			app := fiber.New()
			app.Post("/rooms/create", r.createRoom)

			tt.mock(mockB)
			var res *http.Response
			var body []byte
			if tt.rawBody != "" {
				res, body = doRaw(app, http.MethodPost, "/rooms/create", tt.rawBody)
			} else {
				res, body = doJSON(app, http.MethodPost, "/rooms/create", tt.jsonBody)
			}
			assert.Equal(t, tt.wantStatus, res.StatusCode)
			if tt.wantStatus == http.StatusCreated {
				var resp struct {
					Room *entity.Room `json:"room"`
				}
				_ = json.Unmarshal(body, &resp)
				assert.Equal(t, createdRoom, resp.Room)
				return
			}
			er := decodeError(t, body)
			if tt.wantCode != "" {
				assert.Equal(t, tt.wantCode, er.ErrorDetail.Code)
			}
		})
	}
}

func TestHandlers_CreateSchedule(t *testing.T) {
	roomID := uuid.New()
	scheduleID := uuid.New()
	reqID := uuid.New()

	okBody := map[string]any{
		"id":         reqID.String(),
		"roomId":     roomID.String(),
		"daysOfWeek": []int{1, 3},
		"startTime":  "09:00",
		"endTime":    "18:00",
	}

	tests := []struct {
		name       string
		pathRoomID string
		rawBody    string
		jsonBody   any
		mock       func(b *mock_usecase.MockBookingService)
		wantStatus int
		wantCode   string
	}{
		{
			name:       "BadRoomID",
			pathRoomID: "bad-uuid",
			jsonBody:   okBody,
			mock:       func(b *mock_usecase.MockBookingService) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:       "BadJSON",
			pathRoomID: roomID.String(),
			rawBody:    "{",
			mock:       func(b *mock_usecase.MockBookingService) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:       "InvalidDay",
			pathRoomID: roomID.String(),
			jsonBody: map[string]any{
				"id":         reqID.String(),
				"roomId":     roomID.String(),
				"daysOfWeek": []int{0},
				"startTime":  "09:00",
				"endTime":    "18:00",
			},
			mock:       func(b *mock_usecase.MockBookingService) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:       "BadStartTime",
			pathRoomID: roomID.String(),
			jsonBody: map[string]any{
				"id":         reqID.String(),
				"roomId":     roomID.String(),
				"daysOfWeek": []int{1},
				"startTime":  "9",
				"endTime":    "18:00",
			},
			mock:       func(b *mock_usecase.MockBookingService) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:       "StartNotBeforeEnd",
			pathRoomID: roomID.String(),
			jsonBody: map[string]any{
				"id":         reqID.String(),
				"roomId":     roomID.String(),
				"daysOfWeek": []int{1},
				"startTime":  "18:00",
				"endTime":    "09:00",
			},
			mock:       func(b *mock_usecase.MockBookingService) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:       "RoomNotFound",
			pathRoomID: roomID.String(),
			jsonBody:   okBody,
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().CreateSchedule(gomock.Any(), gomock.Any()).Return(uuid.Nil, entity.ErrRoom)
			},
			wantStatus: http.StatusNotFound,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:       "ScheduleExists",
			pathRoomID: roomID.String(),
			jsonBody:   okBody,
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().CreateSchedule(gomock.Any(), gomock.Any()).Return(uuid.Nil, entity.ErrScheduleExists)
			},
			wantStatus: http.StatusConflict,
			wantCode:   string(response.ScheduleExists),
		},
		{
			name:       "ServiceError",
			pathRoomID: roomID.String(),
			jsonBody:   okBody,
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().CreateSchedule(gomock.Any(), gomock.Any()).Return(uuid.Nil, assert.AnError)
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   string(response.InternalError),
		},
		{
			name:       "OK",
			pathRoomID: roomID.String(),
			jsonBody:   okBody,
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().CreateSchedule(gomock.Any(), gomock.Any()).Return(scheduleID, nil)
			},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			mockB := mock_usecase.NewMockBookingService(c)
			r := newV1ForTest(mockB)
			app := fiber.New()
			app.Post("/rooms/:roomId/schedule/create", r.createSchedule)

			tt.mock(mockB)
			path := "/rooms/" + tt.pathRoomID + "/schedule/create"

			var res *http.Response
			var body []byte
			if tt.rawBody != "" {
				res, body = doRaw(app, http.MethodPost, path, tt.rawBody)
			} else {
				res, body = doJSON(app, http.MethodPost, path, tt.jsonBody)
			}

			assert.Equal(t, tt.wantStatus, res.StatusCode)
			if tt.wantStatus == http.StatusCreated {
				var resp struct {
					Schedule struct {
						Id uuid.UUID `json:"id"`
					} `json:"schedule"`
				}
				_ = json.Unmarshal(body, &resp)
				assert.Equal(t, scheduleID, resp.Schedule.Id)
				return
			}

			er := decodeError(t, body)
			if tt.wantCode != "" {
				assert.Equal(t, tt.wantCode, er.ErrorDetail.Code)
			}
		})
	}
}

func TestHandlers_GetSlots(t *testing.T) {
	roomID := uuid.New()
	date := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)
	dateStr := date.Format("2006-01-02")

	slots := []*entity.Slot{
		{Id: uuid.New(), RoomId: roomID, Start: date.Add(9 * time.Hour), End: date.Add(10 * time.Hour)},
	}

	tests := []struct {
		name       string
		pathRoomID string
		query      string
		mock       func(b *mock_usecase.MockBookingService)
		wantStatus int
		wantCode   string
	}{
		{
			name:       "BadRoomID",
			pathRoomID: "bad-uuid",
			query:      "?date=" + dateStr,
			mock:       func(b *mock_usecase.MockBookingService) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:       "MissingDate",
			pathRoomID: roomID.String(),
			query:      "",
			mock:       func(b *mock_usecase.MockBookingService) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:       "BadDate",
			pathRoomID: roomID.String(),
			query:      "?date=2026-99-99",
			mock:       func(b *mock_usecase.MockBookingService) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:       "IsRoomExistError",
			pathRoomID: roomID.String(),
			query:      "?date=" + dateStr,
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().IsRoomExist(gomock.Any(), roomID).Return(false, assert.AnError)
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   string(response.InternalError),
		},
		{
			name:       "RoomNotFound",
			pathRoomID: roomID.String(),
			query:      "?date=" + dateStr,
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().IsRoomExist(gomock.Any(), roomID).Return(false, nil)
			},
			wantStatus: http.StatusNotFound,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:       "GetSlotsError",
			pathRoomID: roomID.String(),
			query:      "?date=" + dateStr,
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().IsRoomExist(gomock.Any(), roomID).Return(true, nil)
				b.EXPECT().GetSlots(gomock.Any(), roomID, date).Return(nil, assert.AnError)
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   string(response.InternalError),
		},
		{
			name:       "OK",
			pathRoomID: roomID.String(),
			query:      "?date=" + dateStr,
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().IsRoomExist(gomock.Any(), roomID).Return(true, nil)
				b.EXPECT().GetSlots(gomock.Any(), roomID, date).Return(slots, nil)
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			mockB := mock_usecase.NewMockBookingService(c)
			r := newV1ForTest(mockB)
			app := fiber.New()
			app.Get("/rooms/:roomId/slots/list", r.getSlots)

			tt.mock(mockB)
			path := "/rooms/" + tt.pathRoomID + "/slots/list" + tt.query
			res, body := doJSON(app, http.MethodGet, path, nil)
			assert.Equal(t, tt.wantStatus, res.StatusCode)
			if tt.wantStatus == http.StatusOK {
				var resp struct {
					Slots []*entity.Slot `json:"slots"`
				}
				_ = json.Unmarshal(body, &resp)
				assert.Equal(t, slots, resp.Slots)
				return
			}
			er := decodeError(t, body)
			if tt.wantCode != "" {
				assert.Equal(t, tt.wantCode, er.ErrorDetail.Code)
			}
		})
	}
}

func TestHandlers_CreateBooking(t *testing.T) {
	userID := uuid.New()
	slotID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	created := &entity.Booking{
		Id:        uuid.New(),
		SlotId:    slotID,
		UserId:    userID,
		Status:    entity.Active,
		CreatedAt: now,
	}

	tests := []struct {
		name       string
		userIDStr  string
		rawBody    string
		jsonBody   any
		mock       func(b *mock_usecase.MockBookingService)
		wantStatus int
		wantCode   string
	}{
		{
			name:       "BadUserID",
			userIDStr:  "not-uuid",
			jsonBody:   map[string]any{"slotId": slotID.String(), "createConferenceLink": false},
			mock:       func(b *mock_usecase.MockBookingService) {},
			wantStatus: http.StatusUnauthorized,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:       "BadJSON",
			userIDStr:  userID.String(),
			rawBody:    "{",
			mock:       func(b *mock_usecase.MockBookingService) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:      "SlotNotFound_IsSlotInPast",
			userIDStr: userID.String(),
			jsonBody:  map[string]any{"slotId": slotID.String(), "createConferenceLink": false},
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().IsSlotInPast(gomock.Any(), slotID).Return(false, entity.SlotNotFound)
			},
			wantStatus: http.StatusNotFound,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:      "IsSlotInPastError",
			userIDStr: userID.String(),
			jsonBody:  map[string]any{"slotId": slotID.String(), "createConferenceLink": false},
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().IsSlotInPast(gomock.Any(), slotID).Return(false, assert.AnError)
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   string(response.InternalError),
		},
		{
			name:      "SlotInPast",
			userIDStr: userID.String(),
			jsonBody:  map[string]any{"slotId": slotID.String(), "createConferenceLink": false},
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().IsSlotInPast(gomock.Any(), slotID).Return(true, nil)
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:      "SlotNotFound_CreateBooking",
			userIDStr: userID.String(),
			jsonBody:  map[string]any{"slotId": slotID.String(), "createConferenceLink": false},
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().IsSlotInPast(gomock.Any(), slotID).Return(false, nil)
				b.EXPECT().CreateBooking(gomock.Any(), gomock.Any()).Return(nil, entity.SlotNotFound)
			},
			wantStatus: http.StatusNotFound,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:      "SlotIsBusy",
			userIDStr: userID.String(),
			jsonBody:  map[string]any{"slotId": slotID.String(), "createConferenceLink": false},
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().IsSlotInPast(gomock.Any(), slotID).Return(false, nil)
				b.EXPECT().CreateBooking(gomock.Any(), gomock.Any()).Return(nil, entity.SlotIsBusy)
			},
			wantStatus: http.StatusConflict,
			wantCode:   string(response.SlotAlreadyBooked),
		},
		{
			name:      "CreateBookingError",
			userIDStr: userID.String(),
			jsonBody:  map[string]any{"slotId": slotID.String(), "createConferenceLink": false},
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().IsSlotInPast(gomock.Any(), slotID).Return(false, nil)
				b.EXPECT().CreateBooking(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   string(response.InternalError),
		},
		{
			name:      "OK",
			userIDStr: userID.String(),
			jsonBody:  map[string]any{"slotId": slotID.String(), "createConferenceLink": false},
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().IsSlotInPast(gomock.Any(), slotID).Return(false, nil)
				b.EXPECT().CreateBooking(gomock.Any(), gomock.Any()).Return(created, nil)
			},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			mockB := mock_usecase.NewMockBookingService(c)
			r := newV1ForTest(mockB)
			app := fiber.New()
			app.Post("/bookings/create", func(ctx *fiber.Ctx) error {
				ctx.Locals("user_id", tt.userIDStr)
				return r.createBooking(ctx)
			})

			tt.mock(mockB)

			var res *http.Response
			var body []byte
			if tt.rawBody != "" {
				res, body = doRaw(app, http.MethodPost, "/bookings/create", tt.rawBody)
			} else {
				res, body = doJSON(app, http.MethodPost, "/bookings/create", tt.jsonBody)
			}

			assert.Equal(t, tt.wantStatus, res.StatusCode)
			if tt.wantStatus == http.StatusCreated {
				var resp struct {
					Booking *entity.Booking `json:"booking"`
				}
				_ = json.Unmarshal(body, &resp)
				assert.Equal(t, created, resp.Booking)
				return
			}
			er := decodeError(t, body)
			if tt.wantCode != "" {
				assert.Equal(t, tt.wantCode, er.ErrorDetail.Code)
			}
		})
	}
}

func TestHandlers_GetBookings(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	bookings := []*entity.Booking{
		{Id: uuid.New(), SlotId: uuid.New(), UserId: uuid.New(), Status: entity.Active, CreatedAt: now},
	}

	tests := []struct {
		name       string
		query      string
		mock       func(b *mock_usecase.MockBookingService)
		wantStatus int
		wantCode   string
	}{
		{
			name:       "BadPage",
			query:      "?page=0&pageSize=20",
			mock:       func(b *mock_usecase.MockBookingService) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:       "BadPageSize",
			query:      "?page=1&pageSize=0",
			mock:       func(b *mock_usecase.MockBookingService) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:  "ServiceError",
			query: "?page=1&pageSize=20",
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().GetBookings(gomock.Any(), 1, 20).Return(nil, -1, assert.AnError)
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   string(response.InternalError),
		},
		{
			name:  "OK",
			query: "?page=1&pageSize=20",
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().GetBookings(gomock.Any(), 1, 20).Return(bookings, 1, nil)
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			mockB := mock_usecase.NewMockBookingService(c)
			r := newV1ForTest(mockB)
			app := fiber.New()
			app.Get("/bookings/list", r.getBookings)

			tt.mock(mockB)
			res, body := doJSON(app, http.MethodGet, "/bookings/list"+tt.query, nil)
			assert.Equal(t, tt.wantStatus, res.StatusCode)
			if tt.wantStatus == http.StatusOK {
				var resp struct {
					Bookings   []*entity.Booking `json:"bookings"`
					Pagination struct {
						Page     int `json:"page"`
						PageSize int `json:"pageSize"`
						Total    int `json:"total"`
					} `json:"pagination"`
				}
				_ = json.Unmarshal(body, &resp)
				assert.Equal(t, bookings, resp.Bookings)
				assert.Equal(t, 1, resp.Pagination.Total)
				return
			}
			er := decodeError(t, body)
			if tt.wantCode != "" {
				assert.Equal(t, tt.wantCode, er.ErrorDetail.Code)
			}
		})
	}
}

func TestHandlers_GetBookingsUser(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	bookings := []*entity.Booking{
		{Id: uuid.New(), SlotId: uuid.New(), UserId: userID, Status: entity.Active, CreatedAt: now},
	}

	tests := []struct {
		name       string
		userIDStr  string
		mock       func(b *mock_usecase.MockBookingService)
		wantStatus int
		wantCode   string
	}{
		{
			name:       "BadUserID",
			userIDStr:  "not-uuid",
			mock:       func(b *mock_usecase.MockBookingService) {},
			wantStatus: http.StatusUnauthorized,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:      "ServiceError",
			userIDStr: userID.String(),
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().GetBookingsUser(gomock.Any(), userID).Return(nil, assert.AnError)
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   string(response.InternalError),
		},
		{
			name:      "OK",
			userIDStr: userID.String(),
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().GetBookingsUser(gomock.Any(), userID).Return(bookings, nil)
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			mockB := mock_usecase.NewMockBookingService(c)
			r := newV1ForTest(mockB)
			app := fiber.New()
			app.Get("/bookings/my", func(ctx *fiber.Ctx) error {
				ctx.Locals("user_id", tt.userIDStr)
				return r.getBookingsUser(ctx)
			})

			tt.mock(mockB)
			res, body := doJSON(app, http.MethodGet, "/bookings/my", nil)
			assert.Equal(t, tt.wantStatus, res.StatusCode)
			if tt.wantStatus == http.StatusOK {
				var resp struct {
					Bookings []*entity.Booking `json:"bookings"`
				}
				_ = json.Unmarshal(body, &resp)
				assert.Equal(t, bookings, resp.Bookings)
				return
			}
			er := decodeError(t, body)
			if tt.wantCode != "" {
				assert.Equal(t, tt.wantCode, er.ErrorDetail.Code)
			}
		})
	}
}

func TestHandlers_CancelBooking(t *testing.T) {
	userID := uuid.New()
	bookingID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	cancelled := &entity.Booking{
		Id:        bookingID,
		SlotId:    uuid.New(),
		UserId:    userID,
		Status:    entity.Cancelled,
		CreatedAt: now,
	}

	tests := []struct {
		name       string
		userIDStr  string
		pathID     string
		mock       func(b *mock_usecase.MockBookingService)
		wantStatus int
		wantCode   string
	}{
		{
			name:       "BadUserID",
			userIDStr:  "not-uuid",
			pathID:     bookingID.String(),
			mock:       func(b *mock_usecase.MockBookingService) {},
			wantStatus: http.StatusUnauthorized,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:       "BadBookingID",
			userIDStr:  userID.String(),
			pathID:     "bad-uuid",
			mock:       func(b *mock_usecase.MockBookingService) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:      "OtherUserBooking",
			userIDStr: userID.String(),
			pathID:    bookingID.String(),
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().CancelBooking(gomock.Any(), userID, bookingID).Return(nil, entity.OtherUserBooking)
			},
			wantStatus: http.StatusForbidden,
			wantCode:   string(response.Forbidden),
		},
		{
			name:      "BookingNotFound",
			userIDStr: userID.String(),
			pathID:    bookingID.String(),
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().CancelBooking(gomock.Any(), userID, bookingID).Return(nil, entity.BookingNotFound)
			},
			wantStatus: http.StatusNotFound,
			wantCode:   string(response.InvalidRequest),
		},
		{
			name:      "ServiceError",
			userIDStr: userID.String(),
			pathID:    bookingID.String(),
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().CancelBooking(gomock.Any(), userID, bookingID).Return(nil, assert.AnError)
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   string(response.InternalError),
		},
		{
			name:      "OK",
			userIDStr: userID.String(),
			pathID:    bookingID.String(),
			mock: func(b *mock_usecase.MockBookingService) {
				b.EXPECT().CancelBooking(gomock.Any(), userID, bookingID).Return(cancelled, nil)
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			mockB := mock_usecase.NewMockBookingService(c)
			r := newV1ForTest(mockB)
			app := fiber.New()
			app.Post("/bookings/:bookingId/cancel", func(ctx *fiber.Ctx) error {
				ctx.Locals("user_id", tt.userIDStr)
				return r.cancelBooking(ctx)
			})

			tt.mock(mockB)
			res, body := doJSON(app, http.MethodPost, "/bookings/"+tt.pathID+"/cancel", nil)
			assert.Equal(t, tt.wantStatus, res.StatusCode)
			if tt.wantStatus == http.StatusOK {
				var resp struct {
					Booking *entity.Booking `json:"booking"`
				}
				_ = json.Unmarshal(body, &resp)
				assert.Equal(t, cancelled, resp.Booking)
				return
			}
			er := decodeError(t, body)
			if tt.wantCode != "" {
				assert.Equal(t, tt.wantCode, er.ErrorDetail.Code)
			}
		})
	}
}
