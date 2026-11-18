package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/jackc/pgx/v5"
	"waypoint/internal/domain"
)

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// One SQL statement gives a consistent source snapshot. Timestamps make edits
// followed by value restoration invalidate an earlier preview as well.
func optimizationSnapshot(ctx context.Context, q queryRower, user, trip, day string) (string, error) {
	var raw string
	err := q.QueryRow(ctx, `SELECT jsonb_build_array(to_jsonb(t),to_jsonb(d),
 COALESCE((SELECT jsonb_agg(to_jsonb(a) ORDER BY a.position) FROM activities a WHERE a.day_id=d.id),'[]'::jsonb),
 COALESCE((SELECT jsonb_agg(to_jsonb(p) ORDER BY p.id) FROM saved_places p WHERE p.trip_id=t.id),'[]'::jsonb))::text
 FROM trips t JOIN itinerary_days d ON d.trip_id=t.id WHERE t.id=$1 AND t.user_id=$2 AND d.id=$3`, trip, user, day).Scan(&raw)
	if err != nil {
		return "", mapped(err)
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:]), nil
}
func (s *Store) OptimizationSnapshot(ctx context.Context, user, trip, day string) (string, error) {
	for _, id := range []string{trip, day} {
		if err := validID(id); err != nil {
			return "", err
		}
	}
	return optimizationSnapshot(ctx, s.Pool, user, trip, day)
}

type ApprovedVisit struct {
	ID    string
	Start int
}

func (s *Store) ApplyOptimization(ctx context.Context, user, trip, day, snapshot string, visits []ApprovedVisit) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockTrip(ctx, tx, user, trip); err != nil {
		return err
	}
	current, err := optimizationSnapshot(ctx, tx, user, trip, day)
	if err != nil {
		return err
	}
	if current != snapshot {
		return domain.Conflict("The itinerary changed. Generate a new preview before applying.")
	}
	rows, err := tx.Query(ctx, `SELECT id::text FROM activities WHERE day_id=$1 ORDER BY position`, day)
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	proposed := make([]string, len(visits))
	for i, v := range visits {
		proposed[i] = v.ID
		if v.Start < 0 || v.Start >= 1440 {
			return domain.Invalid("Invalid approved start time.")
		}
	}
	if err = domain.ValidateOrder(ids, proposed); err != nil {
		return err
	}
	for i, v := range visits {
		clock := fmt.Sprintf("%02d:%02d", v.Start/60, v.Start%60)
		if _, err = tx.Exec(ctx, `UPDATE activities SET position=$2,start_time=$3,updated_at=clock_timestamp() WHERE id=$1`, v.ID, i, clock); err != nil {
			return err
		}
	}
	// Also invalidate an applied empty schedule and any concurrent approval.
	if _, err = tx.Exec(ctx, `UPDATE trips SET updated_at=clock_timestamp() WHERE id=$1`, trip); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
