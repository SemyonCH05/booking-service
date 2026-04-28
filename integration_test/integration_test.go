package integrationtest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"room-booking-service/internal/entity"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

const (
	host           = "app"
	attempts       = 20
	httpURL        = "http://" + host + ":8080"
	requestTimeout = 5 * time.Second
	basePath       = httpURL
)

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

type dummyLoginReq struct {
	Role string `json:"role"`
}

type dummyLoginResp struct {
	Token string `json:"token"`
}

type createRoomReq struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Capacity    int    `json:"capacity,omitempty"`
}

type createScheduleReq struct {
	Id         uuid.UUID `json:"id,omitempty"`
	RoomId     uuid.UUID `json:"roomId"`
	DaysOfWeek []int     `json:"daysOfWeek"`
	StartTime  string    `json:"startTime"`
	EndTime    string    `json:"endTime"`
}

type createBookingReq struct {
	SlotId               uuid.UUID `json:"slotId"`
	CreateConferenceLink bool      `json:"createConferenceLink"`
}

type errorResp struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type roomCreateResp struct {
	Room *entity.Room `json:"room"`
}

type scheduleDetail struct {
	Id         uuid.UUID `json:"id"`
	RoomId     uuid.UUID `json:"roomId"`
	DaysOfWeek []int     `json:"daysOfWeek"`
	StartTime  string    `json:"startTime"`
	EndTime    string    `json:"endTime"`
}

type scheduleCreateResp struct {
	Schedule scheduleDetail `json:"schedule"`
}

type slotsResp struct {
	Slots []*entity.Slot `json:"slots"`
}

type bookingResp struct {
	Booking *entity.Booking `json:"booking"`
}

func doJSON(ctx context.Context, c *http.Client, method, url, token string, body any, out any) (int, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return 0, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, url, func() *bytes.Reader {
		if body == nil {
			return bytes.NewReader(nil)
		}
		return bytes.NewReader(buf.Bytes())
	}())
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	token = strings.TrimSpace(token)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if out == nil {
		return resp.StatusCode, nil
	}
	return resp.StatusCode, json.NewDecoder(resp.Body).Decode(out)
}

func waitForServer(t *testing.T, c *http.Client) {
	t.Helper()
	for i := 0; i < attempts; i++ {
		reqCtx, reqCancel := context.WithTimeout(context.Background(), requestTimeout)
		req, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, basePath+"/_info", nil)
		resp, err := c.Do(req)
		reqCancel()
		if err == nil && resp != nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("server didn't become ready after %d attempts: %s", attempts, basePath+"/_info")
}

func dummyLogin(t *testing.T, c *http.Client, role string) string {
	t.Helper()
	var resp dummyLoginResp
	status, err := doJSON(
		context.Background(),
		c,
		http.MethodPost,
		basePath+"/dummyLogin",
		"",
		dummyLoginReq{Role: role},
		&resp,
	)
	if err != nil {
		t.Fatalf("dummyLogin decode err: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("dummyLogin role=%s unexpected status=%d body=%+v", role, status, resp)
	}
	if resp.Token == "" {
		t.Fatalf("dummyLogin role=%s returned empty token", role)
	}
	return strings.TrimSpace(resp.Token)
}

func mustStatus(t *testing.T, status int, expected int) {
	t.Helper()
	if status != expected {
		t.Fatalf("unexpected status: got=%d want=%d", status, expected)
	}
}

func TestE2E_CreateRoomScheduleAndBookingUser(t *testing.T) {
	c := &http.Client{Timeout: requestTimeout}

	conn, err := net.DialTimeout("tcp", host+":8080", 200*time.Millisecond)
	if err != nil {
		t.Skipf("integration requires docker-compose network (can't reach %s:8080): %v", host, err)
	}
	conn.Close()

	waitForServer(t, c)

	tomorrow := time.Now().UTC().AddDate(0, 0, 1)
	dateStr := tomorrow.Format("2006-01-02")

	weekday := int(tomorrow.Weekday())
	dayOfWeekAPI := weekday
	if tomorrow.Weekday() == time.Sunday {
		dayOfWeekAPI = 7
	}
	startTime := "09:00"
	endTime := "10:00"

	adminToken := dummyLogin(t, c, "admin")
	userToken := dummyLogin(t, c, "user")

	// create room
	var roomResp roomCreateResp
	roomReq := createRoomReq{Name: "Room A", Description: "desc", Capacity: 8}
	status, err := doJSON(context.Background(), c, http.MethodPost, basePath+"/rooms/create", adminToken, roomReq, &roomResp)
	mustStatus(t, status, http.StatusCreated)
	if err != nil {
		t.Fatalf("createRoom decode err: %v", err)
	}
	if roomResp.Room == nil || roomResp.Room.Id == uuid.Nil {
		t.Fatalf("createRoom: missing room id in response: %+v", roomResp)
	}
	roomID := roomResp.Room.Id

	// create schedule
	var schedResp scheduleCreateResp
	schedReq := createScheduleReq{
		Id:         uuid.New(),
		RoomId:     roomID,
		DaysOfWeek: []int{dayOfWeekAPI},
		StartTime:  startTime,
		EndTime:    endTime,
	}
	status, err = doJSON(context.Background(), c, http.MethodPost, fmt.Sprintf("%s/rooms/%s/schedule/create", basePath, roomID), adminToken, schedReq, &schedResp)
	mustStatus(t, status, http.StatusCreated)
	if err != nil {
		t.Fatalf("createSchedule decode err: %v", err)
	}

	// get slots for that day
	var slots slotsResp
	for i := 0; i < 10; i++ {
		status, err = doJSON(context.Background(), c, http.MethodGet, fmt.Sprintf("%s/rooms/%s/slots/list?date=%s", basePath, roomID, dateStr), userToken, nil, &slots)
		if err == nil && status == http.StatusOK && len(slots.Slots) > 0 {
			break
		}
		time.Sleep(200 * time.Millisecond)
		slots = slotsResp{}
	}
	if status != http.StatusOK {
		t.Fatalf("getSlots bad status=%d err=%v slots=%+v", status, err, slots)
	}
	if len(slots.Slots) == 0 {
		t.Fatalf("getSlots returned empty list for date=%s roomID=%s", dateStr, roomID)
	}
	slotID := slots.Slots[0].Id

	// create booking
	var booking bookingResp
	bookingReq := createBookingReq{SlotId: slotID, CreateConferenceLink: false}
	status, err = doJSON(context.Background(), c, http.MethodPost, basePath+"/bookings/create", userToken, bookingReq, &booking)
	mustStatus(t, status, http.StatusCreated)
	if err != nil {
		t.Fatalf("createBooking decode err: %v", err)
	}
	if booking.Booking == nil || booking.Booking.Id == uuid.Nil {
		t.Fatalf("createBooking missing booking: %+v", booking)
	}
	if booking.Booking.Status != entity.Active {
		t.Fatalf("createBooking unexpected status: %s", booking.Booking.Status)
	}
}

func TestE2E_CancelBookingUser(t *testing.T) {
	c := &http.Client{Timeout: requestTimeout}

	conn, err := net.DialTimeout("tcp", host+":8080", 200*time.Millisecond)
	if err != nil {
		t.Skipf("integration requires docker-compose network (can't reach %s:8080): %v", host, err)
	}
	conn.Close()

	waitForServer(t, c)

	tomorrow := time.Now().UTC().AddDate(0, 0, 1)
	dateStr := tomorrow.Format("2006-01-02")

	weekday := int(tomorrow.Weekday())
	dayOfWeekAPI := weekday
	if tomorrow.Weekday() == time.Sunday {
		dayOfWeekAPI = 7
	}
	startTime := "09:00"
	endTime := "10:00"

	adminToken := dummyLogin(t, c, "admin")
	userToken := dummyLogin(t, c, "user")

	// create room
	var roomResp roomCreateResp
	roomReq := createRoomReq{Name: "Room A", Description: "desc", Capacity: 8}
	status, err := doJSON(context.Background(), c, http.MethodPost, basePath+"/rooms/create", adminToken, roomReq, &roomResp)
	mustStatus(t, status, http.StatusCreated)
	if err != nil {
		t.Fatalf("createRoom decode err: %v", err)
	}
	if roomResp.Room == nil || roomResp.Room.Id == uuid.Nil {
		t.Fatalf("createRoom: missing room id in response: %+v", roomResp)
	}
	roomID := roomResp.Room.Id

	// create schedule
	var schedResp scheduleCreateResp
	schedReq := createScheduleReq{
		Id:         uuid.New(),
		RoomId:     roomID,
		DaysOfWeek: []int{dayOfWeekAPI},
		StartTime:  startTime,
		EndTime:    endTime,
	}
	status, err = doJSON(context.Background(), c, http.MethodPost, fmt.Sprintf("%s/rooms/%s/schedule/create", basePath, roomID), adminToken, schedReq, &schedResp)
	mustStatus(t, status, http.StatusCreated)
	if err != nil {
		t.Fatalf("createSchedule decode err: %v", err)
	}

	// get slots for that day
	var slots slotsResp
	for i := 0; i < 10; i++ {
		status, err = doJSON(context.Background(), c, http.MethodGet, fmt.Sprintf("%s/rooms/%s/slots/list?date=%s", basePath, roomID, dateStr), userToken, nil, &slots)
		if err == nil && status == http.StatusOK && len(slots.Slots) > 0 {
			break
		}
		time.Sleep(200 * time.Millisecond)
		slots = slotsResp{}
	}
	if status != http.StatusOK {
		t.Fatalf("getSlots bad status=%d err=%v slots=%+v", status, err, slots)
	}
	if len(slots.Slots) == 0 {
		t.Fatalf("getSlots returned empty list for date=%s roomID=%s", dateStr, roomID)
	}
	slotID := slots.Slots[0].Id

	// create booking
	var booking bookingResp
	bookingReq := createBookingReq{SlotId: slotID, CreateConferenceLink: false}
	status, err = doJSON(context.Background(), c, http.MethodPost, basePath+"/bookings/create", userToken, bookingReq, &booking)
	mustStatus(t, status, http.StatusCreated)
	if err != nil {
		t.Fatalf("createBooking decode err: %v", err)
	}
	if booking.Booking == nil || booking.Booking.Id == uuid.Nil {
		t.Fatalf("createBooking missing booking: %+v", booking)
	}
	if booking.Booking.Status != entity.Active {
		t.Fatalf("createBooking unexpected status: %s", booking.Booking.Status)
	}
	bookingID := booking.Booking.Id

	// cancel booking twice idempotent
	var cancel1 bookingResp
	status, err = doJSON(context.Background(), c, http.MethodPost, fmt.Sprintf("%s/bookings/%s/cancel", basePath, bookingID), userToken, nil, &cancel1)
	mustStatus(t, status, http.StatusOK)
	if err != nil {
		t.Fatalf("cancel decode err: %v", err)
	}
	if cancel1.Booking == nil {
		t.Fatalf("cancel missing booking: %+v", cancel1)
	}
	if cancel1.Booking.Status != entity.Cancelled {
		t.Fatalf("cancel unexpected status: %s", cancel1.Booking.Status)
	}

	var cancel2 bookingResp
	status, err = doJSON(context.Background(), c, http.MethodPost, fmt.Sprintf("%s/bookings/%s/cancel", basePath, bookingID), userToken, nil, &cancel2)
	mustStatus(t, status, http.StatusOK)
	if err != nil {
		t.Fatalf("cancel2 decode err: %v", err)
	}
	if cancel2.Booking == nil {
		t.Fatalf("cancel2 missing booking: %+v", cancel2)
	}
	if cancel2.Booking.Status != entity.Cancelled {
		t.Fatalf("cancel2 unexpected status: %s", cancel2.Booking.Status)
	}
}
