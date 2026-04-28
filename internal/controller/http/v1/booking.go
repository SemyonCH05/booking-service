package v1

import (
	"errors"
	"net/http"
	"room-booking-service/internal/controller/http/v1/request"
	"room-booking-service/internal/controller/http/v1/response"
	"room-booking-service/internal/entity"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	dummyAdminId = "00000000-0000-0000-0000-000000000001"
	dummyUserId  = "00000000-0000-0000-0000-000000000002"
)

func (r *V1) getInfo(ctx *fiber.Ctx) error {
	return ctx.SendStatus(http.StatusOK)
}

func (r *V1) auth(ctx *fiber.Ctx) error {
	var req request.AuthRequest
	if err := ctx.BodyParser(&req); err != nil {
		return errorResponse(ctx, http.StatusBadRequest, string(response.InvalidRequest), "invalid request")
	}
	if req.Role != "user" && req.Role != "admin" {
		return errorResponse(ctx, http.StatusBadRequest, string(response.InvalidRequest), "invalid request")
	}
	var id string
	if req.Role == "user" {
		id = dummyUserId
	} else {
		id = dummyAdminId
	}
	claims := jwt.MapClaims{
		"user_id": id,
		"role":    req.Role,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString([]byte(r.secret))
	if err != nil {
		return errorResponse(ctx, http.StatusInternalServerError, string(response.InternalError), "internal server error")
	}

	return ctx.JSON(fiber.Map{"token": t})
}

func (r *V1) getRooms(ctx *fiber.Ctx) error {
	rooms, err := r.b.GetRooms(ctx.UserContext())
	if err != nil {
		r.l.Error(errors.Join(errors.New("http/booking - getRooms - GetRooms"), err))
		return errorResponse(ctx, http.StatusInternalServerError, string(response.InternalError), "internal server error")
	}
	resp := response.RoomsResponse{
		Rooms: rooms,
	}
	return ctx.Status(http.StatusOK).JSON(resp)
}

func (r *V1) createRoom(ctx *fiber.Ctx) error {
	var req request.RoomRequest
	if err := ctx.BodyParser(&req); err != nil {
		return errorResponse(ctx, http.StatusBadRequest, string(response.InvalidRequest), "invalid request")
	}
	if err := r.v.Struct(req); err != nil {
		r.l.Error(err, "http/booking - createRooms - validate")
		return errorResponse(ctx, http.StatusBadRequest, string(response.InvalidRequest), "invalid request")
	}
	room := &entity.Room{
		Name:        req.Name,
		Description: req.Description,
		Capacity:    req.Capacity,
	}
	res, err := r.b.CreateRoom(ctx.UserContext(), room)
	if err != nil {
		r.l.Error(errors.Join(errors.New("http/booking - createRoom - CreateRoom"), err))
		return errorResponse(ctx, http.StatusInternalServerError, string(response.InternalError), "internal server error")
	}
	resp := response.RoomResponse{
		Room: res,
	}
	return ctx.Status(http.StatusCreated).JSON(resp)
}

func parseHHMM(s string) (time.Time, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return time.Time{}, errors.New("bad time")
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return time.Time{}, errors.New("bad hour")
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return time.Time{}, errors.New("bad minute")
	}
	return time.Date(1, 1, 1, h, m, 0, 0, time.UTC), nil
}

func (r *V1) createSchedule(ctx *fiber.Ctx) error {
	roomId, err := uuid.Parse(ctx.Params("roomId"))
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, string(response.InvalidRequest), "invalid request")
	}
	var req request.ScheduleRequest
	if err := ctx.BodyParser(&req); err != nil {
		return errorResponse(ctx, http.StatusBadRequest, string(response.InvalidRequest), "invalid request")
	}
	if err := r.v.Struct(req); err != nil {
		r.l.Error(err, "http/booking - createSchedule - validate")
		return errorResponse(ctx, http.StatusBadRequest, string(response.InvalidRequest), "invalid request")
	}
	for _, day := range req.DaysOfWeek {
		if day < 1 || day > 7 {
			return errorResponse(ctx, http.StatusBadRequest, string(response.InvalidRequest), "invalid request")
		}
	}
	start, err := parseHHMM(req.StartTime)
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, string(response.InvalidRequest), "invalid request")
	}
	end, err := parseHHMM(req.EndTime)
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, string(response.InvalidRequest), "invalid request")
	}
	if !start.Before(end) {
		return errorResponse(ctx, http.StatusBadRequest, string(response.InvalidRequest), "invalid request")
	}
	schedule := &entity.Schedule{
		Id:         req.Id,
		RoomId:     roomId,
		DaysOfWeek: make([]entity.DayOfWeek, len(req.DaysOfWeek)),
		StartTime:  start,
		EndTime:    end,
	}
	for i := 0; i < len(schedule.DaysOfWeek); i++ {
		schedule.DaysOfWeek[i] = entity.DayOfWeek(req.DaysOfWeek[i])
	}
	id, err := r.b.CreateSchedule(ctx.UserContext(), schedule)
	if err != nil {
		if errors.Is(err, entity.ErrRoom) {
			return errorResponse(ctx, http.StatusNotFound, string(response.InvalidRequest), "invalid request")
		} else if errors.Is(err, entity.ErrScheduleExists) {
			return errorResponse(ctx, http.StatusConflict, string(response.ScheduleExists), "schedule for this room already exists and cannot be changed")
		}
		r.l.Error(errors.Join(errors.New("http/booking - createSchedule - CreateSchedule"), err))
		return errorResponse(ctx, http.StatusInternalServerError, string(response.InternalError), "internal server error")
	}
	scheduleDetail := response.ScheduleDetail{
		Id:         id,
		RoomId:     req.RoomId,
		DaysOfWeek: req.DaysOfWeek,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
	}
	resp := response.ScheduleResponse{
		Schedule: scheduleDetail,
	}
	return ctx.Status(http.StatusCreated).JSON(resp)
}

func (r *V1) getSlots(ctx *fiber.Ctx) error {
	roomId, err := uuid.Parse(ctx.Params("roomId"))
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, string(response.InvalidRequest), "invalid request")
	}
	dateStr := ctx.Query("date")
	if dateStr == "" {
		return errorResponse(ctx, http.StatusBadRequest, string(response.InvalidRequest), "invalid request")
	}
	d, err := time.ParseInLocation("2006-01-02", dateStr, time.UTC)
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, string(response.InvalidRequest), "invalid request")
	}
	date := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	isExist, err := r.b.IsRoomExist(ctx.UserContext(), roomId)
	if err != nil {
		r.l.Error(errors.Join(errors.New("http/booking - getSlots - IsRoomExist"), err))
		return errorResponse(ctx, http.StatusInternalServerError, string(response.InternalError), "internal server error")
	}
	if !isExist {
		return errorResponse(ctx, http.StatusNotFound, string(response.InvalidRequest), "invalid request")
	}
	slots, err := r.b.GetSlots(ctx.UserContext(), roomId, date)
	if err != nil {
		r.l.Error(errors.Join(errors.New("http/booking - getSlots - GetSlots"), err))
		return errorResponse(ctx, http.StatusInternalServerError, string(response.InternalError), "internal server error")
	}
	resp := response.SlotsResponse{
		Slots: slots,
	}
	return ctx.Status(http.StatusOK).JSON(resp)
}

func (r *V1) createBooking(ctx *fiber.Ctx) error {
	rawUserId := ctx.Locals("user_id").(string)
	userId, err := uuid.Parse(rawUserId)
	if err != nil {
		return errorResponse(ctx, http.StatusUnauthorized, string(response.InvalidRequest), "invalid reqest")
	}
	var req request.BookingRequest
	if err := ctx.BodyParser(&req); err != nil {
		return errorResponse(ctx, http.StatusBadRequest, string(response.InvalidRequest), "invalid request")
	}
	if err := r.v.Struct(req); err != nil {
		r.l.Error(err, "http/booking - createBooking - validate")
		return errorResponse(ctx, http.StatusBadRequest, string(response.InvalidRequest), "invalid request")
	}
	slotInPast, err := r.b.IsSlotInPast(ctx.UserContext(), req.SlotId)
	if err != nil {
		if errors.Is(err, entity.SlotNotFound) {
			return errorResponse(ctx, http.StatusNotFound, string(response.InvalidRequest), "invalid request")
		}
		r.l.Error(errors.Join(errors.New("http/booking - createBooking - IsSlotInPast"), err))
		return errorResponse(ctx, http.StatusInternalServerError, string(response.InternalError), "internal server error")
	}
	if slotInPast {
		return errorResponse(ctx, http.StatusBadRequest, string(response.InvalidRequest), "invalid request")
	}
	booking := &entity.Booking{
		SlotId: req.SlotId,
		UserId: userId,
	}
	b, err := r.b.CreateBooking(ctx.UserContext(), booking)
	if err != nil {
		if errors.Is(err, entity.SlotNotFound) {
			return errorResponse(ctx, http.StatusNotFound, string(response.InvalidRequest), "invalid request")
		} else if errors.Is(err, entity.SlotIsBusy) {
			return errorResponse(ctx, http.StatusConflict, string(response.SlotAlreadyBooked), "slot is already booked")
		}
		r.l.Error(errors.Join(errors.New("http/booking - createBooking - CreateBooking"), err))
		return errorResponse(ctx, http.StatusInternalServerError, string(response.InternalError), "internal server error")
	}
	resp := response.BookingResponse{
		Booking: b,
	}
	return ctx.Status(http.StatusCreated).JSON(resp)
}

func (r *V1) getBookings(ctx *fiber.Ctx) error {
	page := ctx.QueryInt("page", 1)
	pageSize := ctx.QueryInt("pageSize", 20)
	pageSize = min(pageSize, 100)
	if page < 1 || pageSize < 1 {
		return errorResponse(ctx, http.StatusBadRequest, string(response.InvalidRequest), "invalid request")
	}
	bookings, total, err := r.b.GetBookings(ctx.UserContext(), page, pageSize)
	if err != nil {
		r.l.Error(errors.Join(errors.New("http/booking - getBookings - GetBookings"), err))
		return errorResponse(ctx, http.StatusInternalServerError, string(response.InternalError), "internal server error")
	}
	resp := response.BookingsResponse{
		Bookings: bookings,
		Pagination: entity.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}

	return ctx.Status(http.StatusOK).JSON(resp)
}

func (r *V1) getBookingsUser(ctx *fiber.Ctx) error {
	rawUserId := ctx.Locals("user_id").(string)
	userId, err := uuid.Parse(rawUserId)
	if err != nil {
		return errorResponse(ctx, http.StatusUnauthorized, string(response.InvalidRequest), "invalid reqest")
	}
	bookings, err := r.b.GetBookingsUser(ctx.UserContext(), userId)
	if err != nil {
		r.l.Error(errors.Join(errors.New("http/booking - getBookingsUser - GetBookingsUser"), err))
		return errorResponse(ctx, http.StatusInternalServerError, string(response.InternalError), "internal server error")
	}
	resp := response.BookingsUserResponse{
		Bookings: bookings,
	}
	return ctx.Status(http.StatusOK).JSON(resp)
}

func (r *V1) cancelBooking(ctx *fiber.Ctx) error {
	rawUserId := ctx.Locals("user_id").(string)
	userId, err := uuid.Parse(rawUserId)
	if err != nil {
		return errorResponse(ctx, http.StatusUnauthorized, string(response.InvalidRequest), "invalid reqest")
	}
	bookingId, err := uuid.Parse(ctx.Params("bookingId"))
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, string(response.InvalidRequest), "invalid request")
	}
	booking, err := r.b.CancelBooking(ctx.UserContext(), userId, bookingId)
	if err != nil {
		if errors.Is(err, entity.OtherUserBooking) {
			return errorResponse(ctx, http.StatusForbidden, string(response.Forbidden), "cannot cancel another user's booking")
		} else if errors.Is(err, entity.BookingNotFound) {
			return errorResponse(ctx, http.StatusNotFound, string(response.InvalidRequest), "invalid request")
		}
		return errorResponse(ctx, http.StatusInternalServerError, string(response.InternalError), "internal server error")
	}
	resp := response.BookingResponse{
		Booking: booking,
	}
	return ctx.Status(http.StatusOK).JSON(resp)
}
