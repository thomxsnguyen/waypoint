package store

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"waypoint/internal/domain"
)

type Store struct{ Pool *pgxpool.Pool }

func mapped(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.NotFound()
	}
	return err
}
func validID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return domain.NotFound()
	}
	return nil
}
func (s *Store) User(ctx context.Context, subject string) (string, error) {
	var id string
	err := s.Pool.QueryRow(ctx, `INSERT INTO users(auth_provider_id) VALUES($1) ON CONFLICT(auth_provider_id) DO UPDATE SET auth_provider_id=EXCLUDED.auth_provider_id RETURNING id`, subject).Scan(&id)
	return id, err
}

const tripCols = `id::text,name,destination,start_date::text,end_date::text,timezone,description`

func scanTrip(row pgx.Row) (domain.Trip, error) {
	var t domain.Trip
	err := row.Scan(&t.ID, &t.Name, &t.Destination, &t.StartDate, &t.EndDate, &t.Timezone, &t.Description)
	return t, mapped(err)
}
func (s *Store) Trips(ctx context.Context, user string, offset int) ([]domain.Trip, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+tripCols+` FROM trips WHERE user_id=$1 ORDER BY start_date,id LIMIT 50 OFFSET $2`, user, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Trip{}
	for rows.Next() {
		t, err := scanTrip(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
func (s *Store) Trip(ctx context.Context, user, id string) (domain.Trip, error) {
	if err := validID(id); err != nil {
		return domain.Trip{}, err
	}
	return scanTrip(s.Pool.QueryRow(ctx, `SELECT `+tripCols+` FROM trips WHERE id=$1 AND user_id=$2`, id, user))
}

// All trip mutations acquire the same parent row lock. This serializes date
// reconciliation, place deletion and activity ordering, including across tabs.
func lockTrip(ctx context.Context, tx pgx.Tx, user, id string) error {
	if err := validID(id); err != nil {
		return err
	}
	var found string
	return mapped(tx.QueryRow(ctx, `SELECT id FROM trips WHERE id=$1 AND user_id=$2 FOR UPDATE`, id, user).Scan(&found))
}
func reconcileDays(ctx context.Context, tx pgx.Tx, id, start, end string) error {
	var busy bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM itinerary_days d JOIN activities a ON a.day_id=d.id WHERE d.trip_id=$1 AND (d.date<$2::date OR d.date>$3::date))`, id, start, end).Scan(&busy)
	if err != nil {
		return err
	}
	if busy {
		return domain.Conflict("Remove activities from excluded days before shortening the trip.")
	}
	if _, err = tx.Exec(ctx, `DELETE FROM itinerary_days WHERE trip_id=$1 AND (date<$2::date OR date>$3::date)`, id, start, end); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO itinerary_days(trip_id,date) SELECT $1,$2::date+n FROM generate_series(0,$3::date-$2::date) AS n ON CONFLICT(trip_id,date) DO NOTHING`, id, start, end)
	return err
}
func (s *Store) CreateTrip(ctx context.Context, user string, in domain.TripInput) (domain.Trip, error) {
	if err := in.Validate(); err != nil {
		return domain.Trip{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return domain.Trip{}, err
	}
	defer tx.Rollback(ctx)
	t, err := scanTrip(tx.QueryRow(ctx, `INSERT INTO trips(user_id,name,destination,start_date,end_date,timezone,description) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING `+tripCols, user, in.Name, in.Destination, in.StartDate, in.EndDate, in.Timezone, in.Description))
	if err != nil {
		return t, err
	}
	if err = reconcileDays(ctx, tx, t.ID, in.StartDate, in.EndDate); err != nil {
		return t, err
	}
	return t, tx.Commit(ctx)
}

type TripPatch struct {
	Name        *string `json:"name"`
	Destination *string `json:"destination"`
	StartDate   *string `json:"start_date"`
	EndDate     *string `json:"end_date"`
	Timezone    *string `json:"timezone"`
	Description *string `json:"description"`
}

func (s *Store) UpdateTrip(ctx context.Context, user, id string, p TripPatch) (domain.Trip, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return domain.Trip{}, err
	}
	defer tx.Rollback(ctx)
	if err = lockTrip(ctx, tx, user, id); err != nil {
		return domain.Trip{}, err
	}
	old, err := scanTrip(tx.QueryRow(ctx, `SELECT `+tripCols+` FROM trips WHERE id=$1`, id))
	if err != nil {
		return old, err
	}
	in := domain.TripInput{Name: old.Name, Destination: old.Destination, StartDate: old.StartDate, EndDate: old.EndDate, Timezone: old.Timezone, Description: old.Description}
	for _, pair := range []struct {
		src *string
		dst *string
	}{{p.Name, &in.Name}, {p.Destination, &in.Destination}, {p.StartDate, &in.StartDate}, {p.EndDate, &in.EndDate}, {p.Timezone, &in.Timezone}, {p.Description, &in.Description}} {
		if pair.src != nil {
			*pair.dst = *pair.src
		}
	}
	if err = in.Validate(); err != nil {
		return old, err
	}
	if err = reconcileDays(ctx, tx, id, in.StartDate, in.EndDate); err != nil {
		return old, err
	}
	t, err := scanTrip(tx.QueryRow(ctx, `UPDATE trips SET name=$2,destination=$3,start_date=$4,end_date=$5,timezone=$6,description=$7,updated_at=now() WHERE id=$1 RETURNING `+tripCols, id, in.Name, in.Destination, in.StartDate, in.EndDate, in.Timezone, in.Description))
	if err != nil {
		return t, err
	}
	return t, tx.Commit(ctx)
}
func (s *Store) DeleteTrip(ctx context.Context, user, id string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockTrip(ctx, tx, user, id); err != nil {
		return err
	}
	// Delete activities first so the saved-place FK never depends on cascade order.
	if _, err = tx.Exec(ctx, `DELETE FROM activities WHERE trip_id=$1`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM trips WHERE id=$1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Store) Places(ctx context.Context, user, trip string) ([]domain.SavedPlace, error) {
	if _, err := s.Trip(ctx, user, trip); err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT p.id::text,p.trip_id::text,p.provider_place_id,p.notes FROM saved_places p JOIN trips t ON t.id=p.trip_id WHERE p.trip_id=$1 AND t.user_id=$2 ORDER BY p.created_at,p.id`, trip, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.SavedPlace{}
	for rows.Next() {
		var p domain.SavedPlace
		if err = rows.Scan(&p.ID, &p.TripID, &p.ProviderPlaceID, &p.Notes); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (s *Store) SavePlace(ctx context.Context, user, trip, providerID, notes string) (domain.SavedPlace, error) {
	var p domain.SavedPlace
	if err := domain.ValidatePlaceID(providerID); err != nil {
		return p, err
	}
	if err := domain.ValidateNotes(notes); err != nil {
		return p, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return p, err
	}
	defer tx.Rollback(ctx)
	if err = lockTrip(ctx, tx, user, trip); err != nil {
		return p, err
	}
	err = tx.QueryRow(ctx, `SELECT id::text,trip_id::text,provider_place_id,notes FROM saved_places WHERE trip_id=$1 AND provider_place_id=$2`, trip, providerID).Scan(&p.ID, &p.TripID, &p.ProviderPlaceID, &p.Notes)
	if err == nil {
		return p, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return p, err
	}
	var count int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM saved_places WHERE trip_id=$1`, trip).Scan(&count); err != nil {
		return p, err
	}
	if count >= 100 {
		return p, domain.Conflict("A trip can contain at most 100 saved places.")
	}
	err = tx.QueryRow(ctx, `INSERT INTO saved_places(trip_id,provider_place_id,notes) VALUES($1,$2,$3) RETURNING id::text,trip_id::text,provider_place_id,notes`, trip, providerID, notes).Scan(&p.ID, &p.TripID, &p.ProviderPlaceID, &p.Notes)
	if err != nil {
		return p, err
	}
	return p, tx.Commit(ctx)
}
func (s *Store) ChangePlace(ctx context.Context, user, trip, id string, notes *string) error {
	if err := validID(id); err != nil {
		return err
	}
	if notes != nil {
		if err := domain.ValidateNotes(*notes); err != nil {
			return err
		}
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockTrip(ctx, tx, user, trip); err != nil {
		return err
	}
	var found string
	if err = tx.QueryRow(ctx, `SELECT id FROM saved_places WHERE id=$1 AND trip_id=$2`, id, trip).Scan(&found); err != nil {
		return mapped(err)
	}
	if notes == nil {
		var used bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM activities WHERE saved_place_id=$1)`, id).Scan(&used); err != nil {
			return err
		}
		if used {
			return domain.Conflict("Remove this place's itinerary activities before removing the saved place.")
		}
		_, err = tx.Exec(ctx, `DELETE FROM saved_places WHERE id=$1`, id)
	} else {
		_, err = tx.Exec(ctx, `UPDATE saved_places SET notes=$2 WHERE id=$1`, id, *notes)
	}
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Store) Days(ctx context.Context, user, trip string) ([]domain.Day, error) {
	if _, err := s.Trip(ctx, user, trip); err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT d.id::text,d.date::text,COALESCE((SELECT jsonb_agg(jsonb_build_object('id',a.id,'day_id',a.day_id,'saved_place_id',a.saved_place_id,'start_time',to_char(a.start_time,'HH24:MI'),'duration_minutes',a.duration_minutes,'notes',a.notes,'position',a.position) ORDER BY a.position) FROM activities a WHERE a.day_id=d.id),'[]'::jsonb) FROM itinerary_days d JOIN trips t ON t.id=d.trip_id WHERE d.trip_id=$1 AND t.user_id=$2 ORDER BY d.date`, trip, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Day{}
	for rows.Next() {
		var d domain.Day
		var raw []byte
		if err = rows.Scan(&d.ID, &d.Date, &raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &d.Activities); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func lockParent(ctx context.Context, tx pgx.Tx, user, id string, activity bool) (trip, day string, err error) {
	if err = validID(id); err != nil {
		return
	}
	if activity {
		err = tx.QueryRow(ctx, `SELECT a.trip_id::text,a.day_id::text FROM activities a JOIN trips t ON t.id=a.trip_id WHERE a.id=$1 AND t.user_id=$2`, id, user).Scan(&trip, &day)
	} else {
		day = id
		err = tx.QueryRow(ctx, `SELECT d.trip_id::text FROM itinerary_days d JOIN trips t ON t.id=d.trip_id WHERE d.id=$1 AND t.user_id=$2`, id, user).Scan(&trip)
	}
	if err != nil {
		err = mapped(err)
		return
	}
	err = lockTrip(ctx, tx, user, trip)
	if err != nil {
		return
	}
	// Recheck after waiting on the parent: another transaction may have removed it.
	var exists bool
	if activity {
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM activities WHERE id=$1)`, id).Scan(&exists)
	} else {
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM itinerary_days WHERE id=$1)`, id).Scan(&exists)
	}
	if err == nil && !exists {
		err = domain.NotFound()
	}
	return
}

const activityCols = `id::text,day_id::text,saved_place_id::text,to_char(start_time,'HH24:MI'),duration_minutes,notes,position`

func scanActivity(row pgx.Row) (domain.Activity, error) {
	var a domain.Activity
	err := row.Scan(&a.ID, &a.DayID, &a.SavedPlaceID, &a.StartTime, &a.DurationMinutes, &a.Notes, &a.Position)
	return a, mapped(err)
}
func (s *Store) AddActivity(ctx context.Context, user, day string, in domain.ActivityInput) (domain.Activity, error) {
	var a domain.Activity
	if err := in.Validate(); err != nil {
		return a, err
	}
	if err := validID(in.SavedPlaceID); err != nil {
		return a, domain.Invalid("Choose a saved place from this trip.")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return a, err
	}
	defer tx.Rollback(ctx)
	trip, _, err := lockParent(ctx, tx, user, day, false)
	if err != nil {
		return a, err
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM saved_places WHERE id=$1 AND trip_id=$2)`, in.SavedPlaceID, trip).Scan(&exists); err != nil {
		return a, err
	}
	if !exists {
		return a, domain.Invalid("Choose a saved place from this trip.")
	}
	var position int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM activities WHERE day_id=$1`, day).Scan(&position); err != nil {
		return a, err
	}
	if position >= 100 {
		return a, domain.Conflict("A day can contain at most 100 activities.")
	}
	a, err = scanActivity(tx.QueryRow(ctx, `INSERT INTO activities(trip_id,day_id,saved_place_id,start_time,duration_minutes,notes,position) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING `+activityCols, trip, day, in.SavedPlaceID, in.StartTime, in.DurationMinutes, in.Notes, position))
	if err != nil {
		return a, err
	}
	return a, tx.Commit(ctx)
}

type ActivityPatch struct {
	StartTime       json.RawMessage `json:"start_time"`
	DurationMinutes *int            `json:"duration_minutes"`
	Notes           *string         `json:"notes"`
}

func (s *Store) UpdateActivity(ctx context.Context, user, id string, p ActivityPatch) (domain.Activity, error) {
	var a domain.Activity
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return a, err
	}
	defer tx.Rollback(ctx)
	if _, _, err = lockParent(ctx, tx, user, id, true); err != nil {
		return a, err
	}
	a, err = scanActivity(tx.QueryRow(ctx, `SELECT `+activityCols+` FROM activities WHERE id=$1`, id))
	if err != nil {
		return a, err
	}
	in := domain.ActivityInput{SavedPlaceID: a.SavedPlaceID, StartTime: a.StartTime, DurationMinutes: a.DurationMinutes, Notes: a.Notes}
	if p.StartTime != nil {
		if err = json.Unmarshal(p.StartTime, &in.StartTime); err != nil {
			return a, domain.Invalid("Start time must be HH:MM or null.")
		}
	}
	if p.DurationMinutes != nil {
		in.DurationMinutes = *p.DurationMinutes
	}
	if p.Notes != nil {
		in.Notes = *p.Notes
	}
	if err = in.Validate(); err != nil {
		return a, err
	}
	a, err = scanActivity(tx.QueryRow(ctx, `UPDATE activities SET start_time=$2,duration_minutes=$3,notes=$4,updated_at=now() WHERE id=$1 RETURNING `+activityCols, id, in.StartTime, in.DurationMinutes, in.Notes))
	if err != nil {
		return a, err
	}
	return a, tx.Commit(ctx)
}
func (s *Store) DeleteActivity(ctx context.Context, user, id string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, day, err := lockParent(ctx, tx, user, id, true)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM activities WHERE id=$1`, id); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE activities a SET position=o.pos FROM (SELECT id,(row_number() OVER(ORDER BY position)-1)::int AS pos FROM activities WHERE day_id=$1) o WHERE a.id=o.id`, day)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Store) Reorder(ctx context.Context, user, day string, ids []string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, _, err = lockParent(ctx, tx, user, day, false); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT id::text FROM activities WHERE day_id=$1 ORDER BY position`, day)
	if err != nil {
		return err
	}
	current, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	if err = domain.ValidateOrder(current, ids); err != nil {
		return err
	}
	for i, id := range ids {
		if _, err = tx.Exec(ctx, `UPDATE activities SET position=$2,updated_at=now() WHERE id=$1`, id, i); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
