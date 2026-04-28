package v1

import (
	"room-booking-service/internal/controller/http/v1/response"

	"github.com/gofiber/fiber/v2"
)

func errorResponse(ctx *fiber.Ctx, code int, codeMsg, msg string) error {
	return ctx.Status(code).JSON(response.ErrorResponse{
		ErrorDetail: response.ErrorDetail{
			Msg:  msg,
			Code: codeMsg,
		},
	})
}
