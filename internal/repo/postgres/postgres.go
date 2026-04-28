package postgres

import (
	"context"
	"errors"
	"room-booking-service/internal/entity"
	"room-booking-service/pkg/postgres"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repo struct {
	*postgres.Postgres
}

func NewRepo(pg *postgres.Postgres) *Repo {
	return &Repo{
		Postgres: pg,
	}
}

func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

func (r *Repo) CreateRoom(ctx context.Context, room *entity.Room) (uuid.UUID, *time.Time, error) {
	sql, args, err := r.Builder.
		Insert("rooms").
		Columns("name", "description", "capacity").
		Values(room.Name, room.Description, room.Capacity).
		Suffix("RETURNING id, created_at").
		ToSql()
	if err != nil {
		return uuid.Nil, nil, errors.Join(errors.New("postgres - CreateRoom - build sql"), err)
	}
	var id uuid.UUID
	var createdAt *time.Time
	row := r.Pool.QueryRow(ctx, sql, args...)
	err = row.Scan(&id, &createdAt)
	if err != nil {
		return uuid.Nil, nil, errors.Join(errors.New("postgres - CreateRoom - exec"), err)
	}
	return id, utcPtr(createdAt), nil
}

func entityWeekdayToTime(d entity.DayOfWeek) time.Weekday {
	if d == 7 {
		return time.Sunday
	}
	return time.Weekday(d)
}

func combineUTCDateAndClock(calDay time.Time, clock time.Time) time.Time {
	return time.Date(
		calDay.Year(), calDay.Month(), calDay.Day(),
		clock.Hour(), clock.Minute(), clock.Second(), clock.Nanosecond(),
		time.UTC,
	)
}

func (r *Repo) generateSlotsAdd(ctx context.Context, schedule *entity.Schedule, fromDate, toDate time.Time) error {
	var bounds []time.Time
	for t := schedule.StartTime; !t.After(schedule.EndTime); t = t.Add(30 * time.Minute) {
		bounds = append(bounds, t)
	}

	// now := time.Now().UTC()
	// todayUTC := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	b := &pgx.Batch{}
	for calDay := fromDate; !calDay.After(toDate); calDay = calDay.AddDate(0, 0, 1) {
		wd := calDay.Weekday()

		for _, dow := range schedule.DaysOfWeek {
			if entityWeekdayToTime(dow) != wd {
				continue
			}
			for i := 0; i < len(bounds)-1; i++ {
				slotStart := combineUTCDateAndClock(calDay, bounds[i])
				slotEnd := combineUTCDateAndClock(calDay, bounds[i+1])
				if slotEnd.Sub(slotStart) < time.Minute*30 {
					continue
				}
				sql, args, err := r.Builder.
					Insert("slots").
					Columns("room_id", "start_at", "end_at").
					Values(schedule.RoomId, slotStart, slotEnd).
					Suffix("ON CONFLICT(room_id, start_at, end_at) DO NOTHING").
					ToSql()
				if err != nil {
					return errors.Join(errors.New("postgres - generateSlotsAdd - build sql batch"), err)
				}
				b.Queue(sql, args...)
			}
		}
	}

	batchResults := r.Pool.SendBatch(ctx, b)
	defer batchResults.Close()
	for i := 0; i < b.Len(); i++ {
		if _, err := batchResults.Exec(); err != nil {
			return errors.Join(errors.New("postgres - generateSlotsAdd - batch exec"), err)
		}
	}
	return nil
}

func (r *Repo) AddSlots(ctx context.Context) error {
	sqlSchedules, argsSchedules, err := r.Builder.
		Select("room_id", "days_of_week", "start_time", "end_time").
		From("schedules").
		ToSql()
	if err != nil {
		return errors.Join(errors.New("postgres - AddSlots - build sqlSchedules"), err)
	}
	rows, err := r.Pool.Query(ctx, sqlSchedules, argsSchedules...)
	if err != nil {
		return errors.Join(errors.New("postgres -AddSlots - exec sqlSchedules"), err)
	}
	defer rows.Close()
	schedules, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (*entity.Schedule, error) {
		var s entity.Schedule
		if err := row.Scan(&s.RoomId, &s.DaysOfWeek, &s.StartTime, &s.EndTime); err != nil {
			return nil, errors.Join(errors.New("postgres - AddSlots - Scan"), err)
		}
		return &s, nil
	})
	if err != nil {
		return errors.Join(errors.New("postgres - AddSlots - CollectRows"), err)
	}
	for _, schedule := range schedules {
		var rawFromDate pgtype.Date
		sqlFromDate, argsFromdate, err := r.Builder.
			Select("max(start_at)::date").
			From("slots").
			Where(sq.Eq{"room_id": schedule.RoomId}).
			ToSql()
		if err != nil {
			return errors.Join(errors.New("postgres - AddSlots - build sqlFromDate"), err)
		}
		err = r.Pool.QueryRow(ctx, sqlFromDate, argsFromdate...).Scan(&rawFromDate)
		if err != nil {
			return errors.Join(errors.New("postgres - AddSlots - QueryRow"), err)
		}
		var fromDate time.Time
		if !rawFromDate.Valid {
			fromDate = time.Now().UTC()
			fromDate = fromDate.AddDate(0, 0, 1)
		} else {
			fromDate = rawFromDate.Time
		}
		fromDateUTC := time.Date(fromDate.Year(), fromDate.Month(), fromDate.Day(), 0, 0, 0, 0, time.UTC)
		toDate := time.Now().UTC().AddDate(0, 0, 7)
		toDateUTC := time.Date(toDate.Year(), toDate.Month(), toDate.Day(), 0, 0, 0, 0, time.UTC)
		if fromDateUTC.After(toDateUTC) {
			continue
		}
		err = r.generateSlotsAdd(ctx, schedule, fromDateUTC, toDateUTC)
		if err != nil {
			return errors.Join(errors.New("postgres - AddSlots - generateSlotsAdd"), err)
		}
	}
	return nil
}

func (r *Repo) CreateSchedule(ctx context.Context, schedule *entity.Schedule) (uuid.UUID, error) {
	if schedule.Id == uuid.Nil {
		schedule.Id = uuid.New()
	}
	sql, args, err := r.Builder.
		Insert("schedules").
		Columns("id", "room_id", "days_of_week", "start_time", "end_time").
		Values(schedule.Id, schedule.RoomId, schedule.DaysOfWeek, schedule.StartTime, schedule.EndTime).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return uuid.Nil, errors.Join(errors.New("postgres - CreateSchedule - build sql"), err)
	}
	var id uuid.UUID
	row := r.Pool.QueryRow(ctx, sql, args...)
	err = row.Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23503":
				return uuid.Nil, entity.ErrRoom
			case "23505":
				return uuid.Nil, entity.ErrScheduleExists
			}
		}
		return uuid.Nil, errors.Join(errors.New("postgres - CreateSchedule - exec"), err)
	}
	now := time.Now().UTC()
	todayUTC := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	err = r.generateSlotsAdd(ctx, schedule, todayUTC, todayUTC.AddDate(0, 0, 7))
	return id, nil
}

func (r *Repo) IsSlotInPast(ctx context.Context, slotId uuid.UUID) (bool, error) {
	sql, args, err := r.Builder.
		Select("start_at < NOW()").
		From("slots").Where(sq.Eq{"id": slotId}).
		ToSql()
	if err != nil {
		return false, errors.Join(errors.New("postgres - IsSlotInPast - build sql"), err)
	}
	var flag bool
	err = r.Pool.QueryRow(ctx, sql, args...).Scan(&flag)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, entity.SlotNotFound
		}
		return false, errors.Join(errors.New("postgres - IsSlotInPast - exec"), err)
	}
	return flag, nil
}

func (r *Repo) checkSlot(ctx context.Context, slotId uuid.UUID) error {
	sql1, args1, err := r.Builder.
		Select("1").
		From("slots").
		Where(sq.Eq{"id": slotId}).
		ToSql()
	if err != nil {
		return errors.Join(errors.New("postgres - checkSlot - build sql"), err)
	}
	var a int
	err = r.Pool.QueryRow(ctx, sql1, args1...).Scan(&a)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.SlotNotFound
		}
		return errors.Join(errors.New("postgres - checkSlot - exec"), err)
	}
	sql2, args2, err := r.Builder.
		Select("1").
		From("bookings").
		Where(sq.And{
			sq.Eq{"slot_id": slotId},
			sq.Eq{"status": "active"},
		}).
		Limit(1).
		ToSql()
	err = r.Pool.QueryRow(ctx, sql2, args2...).Scan(&a)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return errors.Join(errors.New("postgres - checkSlot - exec"), err)
	}
	return entity.SlotIsBusy
}

func (r *Repo) CreateBooking(ctx context.Context, booking *entity.Booking) (uuid.UUID, *time.Time, error) {
	err := r.checkSlot(ctx, booking.SlotId)
	if err != nil {
		return uuid.Nil, nil, err
	}
	sql, args, err := r.Builder.
		Insert("bookings").
		Columns("slot_id", "user_id", "status", "conference_link").
		Values(booking.SlotId, booking.UserId, entity.Active, booking.ConferenceLink).
		Suffix("RETURNING id, created_at").
		ToSql()
	if err != nil {
		return uuid.Nil, nil, errors.Join(errors.New("postgres - CreateBooking - build sql"), err)
	}
	var id uuid.UUID
	var createdAt *time.Time
	row := r.Pool.QueryRow(ctx, sql, args...)
	err = row.Scan(&id, &createdAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return uuid.Nil, nil, entity.SlotIsBusy
			}
		}
		return uuid.Nil, nil, errors.Join(errors.New("postgres - CreateBooking - exec"), err)
	}
	return id, utcPtr(createdAt), nil
}

func (r *Repo) CancelBooking(ctx context.Context, userId, bookingId uuid.UUID) (*entity.Booking, error) {
	var ownerId uuid.UUID
	checkSql, checkArgs, err := r.Builder.
		Select("user_id").
		From("bookings").
		Where(sq.Eq{"id": bookingId}).
		ToSql()
	if err != nil {
		return nil, errors.Join(errors.New("postgres - CancelBooking - build checkSql"), err)
	}
	err = r.Pool.QueryRow(ctx, checkSql, checkArgs...).Scan(&ownerId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.BookingNotFound
		}
		return nil, errors.Join(errors.New("postgres - CancelBooking - check owner"), err)
	}
	if ownerId != userId {
		return nil, entity.OtherUserBooking
	}
	sql, args, err := r.Builder.
		Update("bookings").
		Set("status", entity.Cancelled).
		Where(sq.Eq{"id": bookingId, "user_id": userId}).
		Suffix("RETURNING slot_id, status, conference_link, created_at").
		ToSql()
	if err != nil {
		return nil, errors.Join(errors.New("postgres - CancelBooking - build sql"), err)
	}
	booking := &entity.Booking{}
	row := r.Pool.QueryRow(ctx, sql, args...)
	err = row.Scan(
		&booking.SlotId,
		&booking.Status,
		&booking.ConferenceLink,
		&booking.CreatedAt,
	)
	if err != nil {
		return nil, errors.Join(errors.New("postgres - CancelBooking - exec"), err)
	}
	booking.Id = bookingId
	booking.UserId = userId
	booking.CreatedAt = booking.CreatedAt.UTC()
	return booking, nil
}

func (r *Repo) GetRooms(ctx context.Context) ([]*entity.Room, error) {
	sql, args, err := r.Builder.
		Select("id", "name", "description", "capacity", "created_at").
		From("rooms").
		ToSql()
	if err != nil {
		return nil, errors.Join(errors.New("postgres - GetRooms - build sql"), err)
	}
	rows, err := r.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, errors.Join(errors.New("postgres - GetRooms - query"), err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*entity.Room, error) {
		var r entity.Room
		if err := row.Scan(&r.Id, &r.Name, &r.Description, &r.Capacity, &r.CreatedAt); err != nil {
			return nil, errors.Join(errors.New("postgres - GetRooms - scan"), err)
		}
		r.CreatedAt = utcPtr(r.CreatedAt)
		return &r, nil
	})
}

func (r *Repo) IsRoomExist(ctx context.Context, roomId uuid.UUID) (bool, error) {
	sql, args, err := r.Builder.
		Select("id").
		From("rooms").
		Where(sq.Eq{"id": roomId}).
		Limit(1).
		ToSql()
	if err != nil {
		return false, errors.Join(errors.New("postgres - GetRooms - build sql"), err)
	}
	var id pgtype.UUID
	err = r.Pool.QueryRow(ctx, sql, args...).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, errors.Join(errors.New("postgres - GetRooms - exec"), err)
	}
	if !id.Valid {
		return false, nil
	}
	return true, nil
}

func (r *Repo) GetSlots(ctx context.Context, roomId uuid.UUID, date time.Time) ([]*entity.Slot, error) {
	sql, args, err := r.Builder.
		Select("s.id", "s.start_at", "s.end_at").
		From("slots s").
		LeftJoin("bookings b ON s.id = b.slot_id AND b.status = 'active'").
		Where(sq.And{
			sq.Eq{"s.room_id": roomId},
			sq.GtOrEq{"s.start_at": date},
			sq.Lt{"s.start_at": date.AddDate(0, 0, 1)},
			sq.Eq{"b.id": nil},
		}).
		OrderBy("s.start_at ASC").
		ToSql()
	if err != nil {
		return nil, errors.Join(errors.New("postgres - GetSlots - build sql"), err)
	}
	rows, err := r.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, errors.Join(errors.New("postgres - GetSlots - query"), err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*entity.Slot, error) {
		var s entity.Slot
		if err := row.Scan(&s.Id, &s.Start, &s.End); err != nil {
			return nil, errors.Join(errors.New("postgres - GetSlots - scan"), err)
		}
		s.Start = s.Start.UTC()
		s.End = s.End.UTC()
		s.RoomId = roomId
		return &s, nil
	})
}

func (r *Repo) GetBookings(ctx context.Context, page, pageSize int) ([]*entity.Booking, int, error) {
	var total int
	sqlT, argsT, err := r.Builder.
		Select("COUNT(*)").
		From("bookings").
		ToSql()
	if err != nil {
		return nil, -1, errors.Join(errors.New("postgres - GetBookings - build sqlT"), err)
	}
	err = r.Pool.QueryRow(ctx, sqlT, argsT...).Scan(&total)
	if err != nil {
		return nil, -1, errors.Join(errors.New("postgres - GetBookings - QueryRow"), err)
	}
	offset := (page - 1) * pageSize
	sql, args, err := r.Builder.
		Select("id", "slot_id", "user_id", "status", "conference_link", "created_at").
		From("bookings").
		OrderBy("created_at DESC").
		Limit(uint64(pageSize)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, -1, errors.Join(errors.New("postgres - GetBookings - build sql"), err)
	}
	rows, err := r.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, -1, errors.Join(errors.New("postgres - GetBookings - query"), err)
	}
	defer rows.Close()

	bookings, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (*entity.Booking, error) {
		var b entity.Booking
		if err := row.Scan(&b.Id, &b.SlotId, &b.UserId, &b.Status, &b.ConferenceLink, &b.CreatedAt); err != nil {
			return nil, errors.Join(errors.New("postgres - GetBookings - scan"), err)
		}
		b.CreatedAt = b.CreatedAt.UTC()
		return &b, nil
	})
	if err != nil {
		return nil, -1, err
	}
	return bookings, total, nil
}

func (r *Repo) GetBookingsUser(ctx context.Context, userId uuid.UUID) ([]*entity.Booking, error) {
	sql, args, err := r.Builder.
		Select("b.id", "b.slot_id", "b.user_id", "b.status", "b.conference_link", "b.created_at").
		From("bookings b JOIN slots s ON b.slot_id = s.id").
		Where(sq.And{
			sq.Eq{"b.user_id": userId},
			sq.Expr("s.start_at >= NOW()"),
		}).
		ToSql()
	if err != nil {
		return nil, errors.Join(errors.New("postgres - GetBookingsUser - build sql"), err)
	}
	rows, err := r.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, errors.Join(errors.New("postgres - GetBookingsUser - query"), err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*entity.Booking, error) {
		var b entity.Booking
		if err := row.Scan(&b.Id, &b.SlotId, &b.UserId, &b.Status, &b.ConferenceLink, &b.CreatedAt); err != nil {
			return nil, errors.Join(errors.New("postgres - GetBookingsUser - scan"), err)
		}
		b.CreatedAt = b.CreatedAt.UTC()
		return &b, nil
	})
}
