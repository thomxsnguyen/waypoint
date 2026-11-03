package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"waypoint/internal/domain"
	"waypoint/internal/store"
	"waypoint/migrations"
)

type testAuth struct{}

func (testAuth) Verify(_ context.Context, token string) (string, error) {
	if token == "user-a" || token == "user-b" {
		return token, nil
	}
	return "", errors.New("invalid")
}

type testPlaces struct{}

func (testPlaces) Search(_ context.Context, _, _ string) ([]domain.Place, error) {
	return []domain.Place{{Provider: "google", ProviderPlaceID: "tokyo-museum", Name: "Tokyo Museum", Location: &domain.Location{Lat: 35, Lng: 139}}}, nil
}
func (testPlaces) Details(_ context.Context, id string) (domain.Place, error) {
	if id == "unavailable" {
		return domain.Place{}, errors.New("provider unavailable")
	}
	return domain.Place{Provider: "google", ProviderPlaceID: id, Name: id, Location: &domain.Location{Lat: 35, Lng: 139}}, nil
}
func database(t *testing.T) *pgxpool.Pool {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("TEST_DATABASE_URL not set; integration tests require isolated PostgreSQL")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	schema := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close(); _, _ = admin.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`); admin.Close() })
	if err = migrations.Up(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err = migrations.Up(ctx, pool); err != nil {
		t.Fatalf("migration not idempotent: %v", err)
	}
	return pool
}

type client struct {
	t       *testing.T
	handler http.Handler
	token   string
}

func (c client) call(method, path, body string, want int) json.RawMessage {
	c.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, req)
	if rec.Code != want {
		c.t.Fatalf("%s %s: got %d want %d: %s", method, path, rec.Code, want, rec.Body.String())
	}
	var envelope struct{ Data json.RawMessage }
	if rec.Code != 204 {
		if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
			c.t.Fatal(err)
		}
	}
	return envelope.Data
}
func decodeData[T any](t *testing.T, raw json.RawMessage) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

const tripBody = `{"name":"Tokyo","destination":"Tokyo, Japan","start_date":"2026-10-31","end_date":"2026-11-02","timezone":"America/Los_Angeles","description":"Autumn"}`

func TestPersistentJourneyAndIsolation(t *testing.T) {
	pool := database(t)
	api := &API{Store: &store.Store{Pool: pool}, Auth: testAuth{}, Places: testPlaces{}, Origin: "http://localhost:3000"}
	handler := api.Router()
	a := client{t, handler, "user-a"}
	b := client{t, handler, "user-b"}
	anon := client{t, handler, ""}
	anon.call("GET", "/api/v1/trips", "", 401)
	client{t, handler, "forged"}.call("GET", "/api/v1/trips", "", 401)
	trip := decodeData[domain.Trip](t, a.call("POST", "/api/v1/trips", tripBody, 201))
	base := "/api/v1/trips/" + trip.ID
	a.call("POST", "/api/v1/trips", `{"user_id":"override"}`, 400)
	b.call("GET", base, "", 404)
	b.call("PATCH", base, `{"name":"stolen"}`, 404)
	b.call("DELETE", base, "", 404)
	if trips := decodeData[[]domain.Trip](t, b.call("GET", "/api/v1/trips", "", 200)); len(trips) != 0 {
		t.Fatal("trip leakage")
	}
	days := decodeData[[]domain.Day](t, a.call("GET", base+"/days", "", 200))
	if len(days) != 3 || days[1].Date != "2026-11-01" {
		t.Fatal("calendar day generation across DST", days)
	}
	b.call("GET", base+"/days", "", 404)
	b.call("GET", base+"/places", "", 404)
	b.call("POST", base+"/places", `{"provider_place_id":"tokyo-museum"}`, 404)
	a.call("GET", "/api/v1/places/search?trip_id="+trip.ID+"&q=museum", "", 200)
	b.call("GET", "/api/v1/places/search?trip_id="+trip.ID+"&q=museum", "", 404)
	a.call("GET", "/api/v1/places/search?trip_id="+trip.ID+"&q=", "", 400)
	place := decodeData[domain.SavedPlace](t, a.call("POST", base+"/places", `{"provider_place_id":"tokyo-museum","notes":"tickets"}`, 201))
	duplicate := decodeData[domain.SavedPlace](t, a.call("POST", base+"/places", `{"provider_place_id":"tokyo-museum"}`, 201))
	if duplicate.ID != place.ID || duplicate.Notes != "tickets" {
		t.Fatal("duplicate save changed record")
	}
	a.call("PATCH", base+"/places/"+place.ID, `{"notes":"book ahead"}`, 200)
	b.call("PATCH", base+"/places/"+place.ID, `{"notes":"stolen"}`, 404)
	b.call("DELETE", base+"/places/"+place.ID, "", 404)
	create := func(day string) domain.Activity {
		return decodeData[domain.Activity](t, a.call("POST", "/api/v1/days/"+day+"/activities", fmt.Sprintf(`{"saved_place_id":%q,"start_time":"09:30","duration_minutes":90,"notes":"First stop"}`, place.ID), 201))
	}
	first := create(days[0].ID)
	second := create(days[0].ID)
	third := create(days[0].ID)
	otherDay := create(days[1].ID)
	b.call("POST", "/api/v1/days/"+days[0].ID+"/activities", fmt.Sprintf(`{"saved_place_id":%q,"duration_minutes":60}`, place.ID), 404)
	for _, method := range []string{"PATCH", "DELETE"} {
		body := ""
		if method == "PATCH" {
			body = `{"notes":"stolen"}`
		}
		b.call(method, "/api/v1/activities/"+first.ID, body, 404)
	}
	a.call("PATCH", "/api/v1/activities/"+first.ID, `{"start_time":null,"notes":"Updated","duration_minutes":45}`, 200)
	a.call("PATCH", "/api/v1/activities/"+first.ID, `{"duration_minutes":0}`, 400)
	a.call("DELETE", base+"/places/"+place.ID, "", 409)
	orderPath := "/api/v1/days/" + days[0].ID + "/order"
	for _, ids := range [][]string{{first.ID, second.ID}, {first.ID, first.ID, third.ID}, {first.ID, second.ID, otherDay.ID}} {
		raw, _ := json.Marshal(map[string]any{"activity_ids": ids})
		a.call("PUT", orderPath, string(raw), 400)
	}
	raw, _ := json.Marshal(map[string]any{"activity_ids": []string{third.ID, first.ID, second.ID}})
	b.call("PUT", orderPath, string(raw), 404)
	a.call("PUT", orderPath, string(raw), 200)
	// A fresh router and identity lookup reads the same PostgreSQL data.
	a.handler = (&API{Store: &store.Store{Pool: pool}, Auth: testAuth{}, Places: testPlaces{}, Origin: api.Origin}).Router()
	persisted := decodeData[[]domain.Day](t, a.call("GET", base+"/days", "", 200))
	if persisted[0].Activities[0].ID != third.ID || persisted[0].Activities[1].StartTime != nil || persisted[0].Activities[1].Notes != "Updated" {
		t.Fatal("itinerary did not persist", persisted)
	}
	a.call("DELETE", "/api/v1/activities/"+first.ID, "", 204)
	persisted = decodeData[[]domain.Day](t, a.call("GET", base+"/days", "", 200))
	if len(persisted[0].Activities) != 2 || persisted[0].Activities[1].Position != 1 {
		t.Fatal("positions not compacted")
	}
	a.call("PATCH", base, `{"start_date":"2026-11-01"}`, 409)
	unchanged := decodeData[domain.Trip](t, a.call("GET", base, "", 200))
	if unchanged.StartDate != trip.StartDate {
		t.Fatal("failed resize did not roll back")
	}
	a.call("PATCH", base, `{"end_date":"2026-11-03"}`, 200)
	a.call("PATCH", base, `{"end_date":"2026-11-01"}`, 200)
	persisted = decodeData[[]domain.Day](t, a.call("GET", base+"/days", "", 200))
	if len(persisted) != 2 {
		t.Fatal("empty days not reconciled")
	}
	otherTrip := decodeData[domain.Trip](t, a.call("POST", "/api/v1/trips", tripBody, 201))
	otherDays := decodeData[[]domain.Day](t, a.call("GET", "/api/v1/trips/"+otherTrip.ID+"/days", "", 200))
	a.call("POST", "/api/v1/days/"+otherDays[0].ID+"/activities", fmt.Sprintf(`{"saved_place_id":%q,"duration_minutes":60}`, place.ID), 400)
	_, err := pool.Exec(context.Background(), `INSERT INTO activities(trip_id,day_id,saved_place_id,duration_minutes,position) VALUES($1,$2,$3,60,0)`, otherTrip.ID, otherDays[0].ID, place.ID)
	if err == nil {
		t.Fatal("composite FK allowed cross-trip activity")
	}
	a.call("POST", base+"/places", `{"provider_place_id":"unavailable"}`, 201)
	saved := decodeData[[]domain.SavedPlace](t, a.call("GET", base+"/places", "", 200))
	if !saved[1].Unavailable || saved[0].Notes != "book ahead" {
		t.Fatal("provider outage lost saved content")
	}
	a.call("DELETE", base, "", 204)
	for _, table := range []string{"saved_places", "itinerary_days", "activities"} {
		var n int
		if err = pool.QueryRow(context.Background(), `SELECT count(*) FROM `+table+` WHERE trip_id=$1`, trip.ID).Scan(&n); err != nil || n != 0 {
			t.Fatalf("cascade %s: %d %v", table, n, err)
		}
	}
}
func TestConcurrentAppends(t *testing.T) {
	pool := database(t)
	s := &store.Store{Pool: pool}
	ctx := context.Background()
	user, err := s.User(ctx, "concurrent")
	if err != nil {
		t.Fatal(err)
	}
	trip, err := s.CreateTrip(ctx, user, domain.TripInput{Name: "Test", Destination: "Tokyo", StartDate: "2026-01-01", EndDate: "2026-01-01", Timezone: "Asia/Tokyo"})
	if err != nil {
		t.Fatal(err)
	}
	days, err := s.Days(ctx, user, trip.ID)
	if err != nil {
		t.Fatal(err)
	}
	place, err := s.SavePlace(ctx, user, trip.ID, "place", "")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.AddActivity(ctx, user, days[0].ID, domain.ActivityInput{SavedPlaceID: place.ID, DurationMinutes: 60})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	days, err = s.Days(ctx, user, trip.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(days[0].Activities) != 12 {
		t.Fatal("lost activities")
	}
	for i, a := range days[0].Activities {
		if a.Position != i {
			t.Fatal("noncontiguous ordering")
		}
	}
}
func TestMigrationRollback(t *testing.T) {
	pool := database(t)
	sql, err := os.ReadFile("../../migrations/001_initial.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(context.Background(), string(sql)); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(context.Background(), `DELETE FROM schema_migrations WHERE version=1`); err != nil {
		t.Fatal(err)
	}
	if err = migrations.Up(context.Background(), pool); err != nil {
		t.Fatal(err)
	}
}
