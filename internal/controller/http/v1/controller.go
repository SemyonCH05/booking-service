package v1

import (
	"room-booking-service/internal/usecase"
	"room-booking-service/pkg/logger"

	"github.com/go-playground/validator/v10"
)

type V1 struct {
	b      usecase.BookingService
	l      logger.Interface
	v      *validator.Validate
	secret string
}
