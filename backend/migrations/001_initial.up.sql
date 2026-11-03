CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  auth_provider_id TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE trips (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 120),
  destination TEXT NOT NULL CHECK (length(destination) BETWEEN 1 AND 200),
  start_date DATE NOT NULL,
  end_date DATE NOT NULL,
  timezone TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (end_date >= start_date AND end_date - start_date < 366)
);
CREATE INDEX ON trips(user_id);
CREATE TABLE saved_places (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  provider TEXT NOT NULL DEFAULT 'google' CHECK (provider = 'google'),
  provider_place_id TEXT NOT NULL,
  notes TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (trip_id, provider_place_id),
  UNIQUE (id, trip_id)
);
CREATE TABLE itinerary_days (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  date DATE NOT NULL,
  UNIQUE (trip_id, date),
  UNIQUE (id, trip_id)
);
CREATE TABLE activities (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id UUID NOT NULL,
  day_id UUID NOT NULL,
  saved_place_id UUID NOT NULL,
  start_time TIME,
  duration_minutes INTEGER NOT NULL DEFAULT 60 CHECK (duration_minutes BETWEEN 1 AND 1440),
  notes TEXT NOT NULL DEFAULT '',
  position INTEGER NOT NULL CHECK (position >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  FOREIGN KEY (day_id, trip_id) REFERENCES itinerary_days(id, trip_id) ON DELETE CASCADE,
  FOREIGN KEY (saved_place_id, trip_id) REFERENCES saved_places(id, trip_id),
  UNIQUE (day_id, position) DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX ON activities(saved_place_id);
