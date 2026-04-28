package http

import (
	"room-booking-service/config"
	"room-booking-service/internal/controller/http/middleware"
	v1 "room-booking-service/internal/controller/http/v1"
	"room-booking-service/internal/usecase"
	"room-booking-service/pkg/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
)

// NewRouter -.
// Swagger spec:
// @title		RoomBookingService
// @version     1.0
// @host        localhost:8080
// @BasePath    /
func NewRouter(app *fiber.App, cfg *config.Config, b usecase.BookingService, l logger.Interface) {
	app.Use(middleware.Logger(l))
	app.Use(middleware.Recovery(l))
	if cfg.Swagger.Enabled {
		app.Get("/swagger/*", swagger.HandlerDefault)
	}
	apiGroup := app.Group("")
	{
		v1.NewRouters(apiGroup, b, l, cfg.HTTP.JwtSecret)
	}
}
