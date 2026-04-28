package response

type Error string

const (
	InvalidRequest    Error = "INVALID_REQUEST"
	Unauthorized      Error = "UNAUTHORIZED"
	NotFound          Error = "NOT_FOUND"
	RoomNotFound      Error = "ROOM_NOT_FOUND"
	SlotNotFound      Error = "SLOT_NOT_FOUND"
	SlotAlreadyBooked Error = "SLOT_ALREADY_BOOKED"
	BookingNotFound   Error = "BOOKING_NOT_FOUND"
	Forbidden         Error = "FORBIDDEN"
	ScheduleExists    Error = "SCHEDULE_EXISTS"
	InternalError     Error = "INTERNAL_ERROR"
)

type ErrorResponse struct {
	ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code string `json:"code"`
	Msg  string `json:"message"`
}
