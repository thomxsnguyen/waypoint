package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"
	"waypoint/internal/domain"
	"waypoint/internal/routing"
	"waypoint/internal/store"
)

type testRoutes struct {
	calls int
	err   error
}

func (p *testRoutes) Matrix(_ context.Context, ids []string, mode routing.TravelMode) ([][]int, error) {
	p.calls++
	if p.err != nil {
		return nil, p.err
	}
	return [][]int{{0, 10}, {3, 0}}, nil
}
func TestOptimizationPreview(t *testing.T) {
	pool := database(t)
	provider := &testRoutes{}
	api := &API{Store: &store.Store{Pool: pool}, Auth: testAuth{}, Places: testPlaces{}, Routes: provider}
	handler := api.Router()
	a := client{t, handler, "user-a"}
	b := client{t, handler, "user-b"}
	trip := decodeData[domain.Trip](t, a.call("POST", "/api/v1/trips", tripBody, 201))
	base := "/api/v1/trips/" + trip.ID
	days := decodeData[[]domain.Day](t, a.call("GET", base+"/days", "", 200))
	for i := 0; i < 2; i++ {
		place := decodeData[domain.SavedPlace](t, a.call("POST", base+"/places", fmt.Sprintf(`{"provider_place_id":"place-%d"}`, i), 201))
		a.call("POST", "/api/v1/days/"+days[0].ID+"/activities", fmt.Sprintf(`{"saved_place_id":%q,"duration_minutes":30,"start_time":"09:00"}`, place.ID), 201)
	}
	before := decodeData[[]domain.Day](t, a.call("GET", base+"/days", "", 200))
	path := base + "/days/" + days[0].ID + "/optimization-preview"
	body := `{"travel_mode":"WALK","start_minute":540,"end_minute":720}`
	client{t, handler, ""}.call("POST", path, body, 401)
	b.call("POST", path, body, 404)
	a.call("POST", base+"/days/00000000-0000-0000-0000-000000000000/optimization-preview", body, 404)
	for _, invalid := range []string{`{}`, `{"travel_mode":"TRANSIT","start_minute":540,"end_minute":720}`, `{"travel_mode":"WALK","start_minute":720,"end_minute":540}`, `{"travel_mode":"WALK","start_minute":540,"end_minute":720,"fixed_activity_ids":["foreign"]}`} {
		a.call("POST", path, invalid, 400)
	}
	if provider.calls != 0 {
		t.Fatal("invalid or unauthorized request called provider")
	}
	var preview struct {
		Changed  bool
		Savings  int `json:"savings_minutes"`
		Original previewSchedule
		Proposed previewSchedule
	}
	raw := a.call("POST", path, body, 200)
	if err := json.Unmarshal(raw, &preview); err != nil {
		t.Fatal(err)
	}
	if !preview.Changed || preview.Savings != 7 || preview.Proposed.Visits[0].ActivityID != before[0].Activities[1].ID {
		t.Fatalf("%s", raw)
	}
	fixedBody := fmt.Sprintf(`{"travel_mode":"DRIVE","start_minute":540,"end_minute":720,"fixed_activity_ids":[%q]}`, before[0].Activities[0].ID)
	raw = a.call("POST", path, fixedBody, 200)
	if err := json.Unmarshal(raw, &preview); err != nil {
		t.Fatal(err)
	}
	if preview.Changed || preview.Proposed.Visits[0].ActivityID != before[0].Activities[0].ID {
		t.Fatalf("fixed start ignored: %s", raw)
	}
	provider.err = context.DeadlineExceeded
	a.call("POST", path, body, 503)
	provider.err = nil
	count := provider.calls
	api.limits.buckets["routes:global"] = bucket{start: time.Now(), count: 4000}
	a.call("POST", path, body, 429)
	if provider.calls != count {
		t.Fatal("quota rejection called provider")
	}
	delete(api.limits.buckets, "routes:global")
	owner, err := api.Store.User(context.Background(), "user-a")
	if err != nil {
		t.Fatal(err)
	}
	api.limits.buckets["routes:"+owner] = bucket{start: time.Now(), count: 800}
	a.call("POST", path, body, 429)
	if provider.calls != count {
		t.Fatal("user quota rejection called provider")
	}
	api.Routes = nil
	a.call("POST", path, body, 503)
	after := decodeData[[]domain.Day](t, a.call("GET", base+"/days", "", 200))
	if !reflect.DeepEqual(before, after) {
		t.Fatal("preview changed persisted itinerary")
	}
}
