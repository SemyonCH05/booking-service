CREATE TYPE role AS ENUM ('admin', 'user');

CREATE TYPE status AS ENUM ('active', 'cancelled');

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email VARCHAR UNIQUE 
    CONSTRAINT proper_email CHECK (
      email ~* '^[A-Za-z0-9.!#$%&''*+/=?^_`{|}~-]+@[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)+$'
    ) NOT NULL,
    role role NOT NULL,
    created_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL, 
    description TEXT,
    capacity INTEGER,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID UNIQUE REFERENCES rooms(id) ON DELETE RESTRICT ON UPDATE CASCADE NOT NULL,
    days_of_week SMALLINT[] NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    CONSTRAINT check_times CHECK (start_time < end_time)
);

CREATE TABLE IF NOT EXISTS slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID REFERENCES rooms(id) ON DELETE RESTRICT ON UPDATE CASCADE NOT NULL,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT check_times CHECK (start_at < end_at)
);

CREATE TABLE IF NOT EXISTS bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slot_id UUID REFERENCES slots(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
    status status NOT NULL,
    conference_link TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS slots_duplicate_uidx ON slots (room_id, start_at, end_at);
CREATE UNIQUE INDEX IF NOT EXISTS bookings_active_slot_uidx ON bookings (slot_id) WHERE status = 'active';