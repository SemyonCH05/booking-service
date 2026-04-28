package v1

import (
	"room-booking-service/internal/controller/http/middleware"
	"room-booking-service/internal/usecase"
	"room-booking-service/pkg/logger"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

func NewRouters(apiV1Group fiber.Router, b usecase.BookingService, l logger.Interface, secretKey string) {
	r := &V1{
		b:      b,
		l:      l,
		v:      validator.New(validator.WithRequiredStructEnabled()),
		secret: secretKey,
	}
	jwtUser := middleware.Jwt(secretKey, "user")
	jwtAdmin := middleware.Jwt(secretKey, "admin")
	jwtAny := middleware.Jwt(secretKey, "user", "admin")
	apiV1Group.Post("/dummyLogin", r.auth)
	apiV1Group.Get("/_info", r.getInfo)
	roomsGroup := apiV1Group.Group("/rooms")
	{
		roomsGroup.Get("/list", jwtAny, r.getRooms)
		roomsGroup.Post("/create", r.createRoom, jwtAdmin)
		roomsGroup.Post("/:roomId/schedule/create", jwtAdmin, r.createSchedule)
		roomsGroup.Get("/:roomId/slots/list", jwtAny, r.getSlots)
	}
	bookingGroup := apiV1Group.Group("/bookings")
	{
		bookingGroup.Post("/create", jwtUser, r.createBooking)
		bookingGroup.Get("/list", jwtAdmin, r.getBookings)
		bookingGroup.Get("/my", jwtUser, r.getBookingsUser)
		bookingGroup.Post("/:bookingId/cancel", jwtUser, r.cancelBooking)
	}
}
