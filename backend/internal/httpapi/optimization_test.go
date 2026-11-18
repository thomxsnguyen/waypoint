package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
	"waypoint/internal/domain"
	"waypoint/internal/routing"
	"waypoint/internal/store"
)

type testRoutes struct {
	calls int
	err   error
	hook  func()
}

func (p *testRoutes) Matrix(_ context.Context, ids []string, mode routing.TravelMode) ([][]int, error) {
	p.calls++
	if p.hook != nil {
		p.hook()
	}
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
	for _, window := range []string{
		`{"activity_id":"foreign","start_minute":540,"end_minute":600}`,
		fmt.Sprintf(`{"activity_id":%q,"start_minute":600}`, before[0].Activities[0].ID),
		fmt.Sprintf(`{"activity_id":%q,"start_minute":600,"end_minute":500}`, before[0].Activities[0].ID),
	} {
		a.call("POST", path, `{"travel_mode":"WALK","start_minute":540,"end_minute":720,"visiting_windows":[`+window+`]}`, 400)
	}
	if provider.calls != 0 {
		t.Fatal("invalid visiting window called provider")
	}
	constrained := fmt.Sprintf(`{"travel_mode":"WALK","start_minute":540,"end_minute":720,"visiting_windows":[{"activity_id":%q,"start_minute":600,"end_minute":620}]}`, before[0].Activities[0].ID)
	var infeasible struct {
		ApprovalID *string `json:"approval_id"`
		Proposed   previewSchedule
	}
	if err := json.Unmarshal(a.call("POST", path, constrained, 200), &infeasible); err != nil {
		t.Fatal(err)
	}
	if infeasible.ApprovalID != nil || len(infeasible.Proposed.Conflicts) == 0 {
		t.Fatal("impossible window was approved")
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

func TestOptimizationApproval(t *testing.T) {
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
		p := decodeData[domain.SavedPlace](t, a.call("POST", base+"/places", fmt.Sprintf(`{"provider_place_id":"approval-%d"}`, i), 201))
		a.call("POST", "/api/v1/days/"+days[0].ID+"/activities", fmt.Sprintf(`{"saved_place_id":%q,"duration_minutes":30,"notes":"keep me"}`, p.ID), 201)
	}
	dayPath := base + "/days/" + days[0].ID
	body := `{"travel_mode":"WALK","start_minute":540,"end_minute":720}`
	preview := func() string {
		var out struct {
			ApprovalID string `json:"approval_id"`
		}
		raw := a.call("POST", dayPath+"/optimization-preview", body, 200)
		if err := json.Unmarshal(raw, &out); err != nil || out.ApprovalID == "" {
			t.Fatalf("%s %v", raw, err)
		}
		return out.ApprovalID
	}
	applyBody := func(id string) string { return fmt.Sprintf(`{"approval_id":%q}`, id) }
	// Infeasible previews cannot be approved.
	impossible := a.call("POST", dayPath+"/optimization-preview", `{"travel_mode":"WALK","start_minute":540,"end_minute":550}`, 200)
	var noApproval struct {
		ID *string `json:"approval_id"`
	}
	if err := json.Unmarshal(impossible, &noApproval); err != nil || noApproval.ID != nil {
		t.Fatalf("infeasible approval: %s", impossible)
	}
	// Mutations during provider computation invalidate preview issuance.
	provider.hook = func() { a.call("PATCH", base, `{"description":"changed during routing"}`, 200) }
	a.call("POST", dayPath+"/optimization-preview", body, 409)
	provider.hook = nil
	id := preview()
	b.call("POST", dayPath+"/optimization-apply", applyBody(id), 404)
	a.call("POST", base+"/days/"+days[1].ID+"/optimization-apply", applyBody(id), 409)
	a.call("POST", dayPath+"/optimization-apply", applyBody("invented"), 409)
	a.call("POST", dayPath+"/optimization-apply", `{"approval_id":"x","visits":[]}`, 400)
	before := decodeData[[]domain.Day](t, a.call("GET", base+"/days", "", 200))
	activityPath := "/api/v1/activities/" + before[0].Activities[0].ID
	a.call("PATCH", activityPath, `{"notes":"edited"}`, 200)
	a.call("PATCH", activityPath, `{"notes":"keep me"}`, 200)
	a.call("POST", dayPath+"/optimization-apply", applyBody(id), 409)
	id = preview()
	api.approvals.mu.Lock()
	expired := api.approvals.entries[id]
	expired.Expires = time.Now().Add(-time.Second)
	api.approvals.entries[id] = expired
	api.approvals.mu.Unlock()
	a.call("POST", dayPath+"/optimization-apply", applyBody(id), 409)
	id = preview()
	calls := provider.calls
	a.call("POST", dayPath+"/optimization-apply", applyBody(id), 200)
	a.call("POST", dayPath+"/optimization-apply", applyBody(id), 409)
	after := decodeData[[]domain.Day](t, a.call("GET", base+"/days", "", 200))
	if provider.calls != calls {
		t.Fatal("apply called provider")
	}
	if after[0].Activities[0].ID != before[0].Activities[1].ID || *after[0].Activities[0].StartTime != "09:00" || *after[0].Activities[1].StartTime != "09:33" {
		t.Fatalf("%+v", after[0])
	}
	for _, act := range after[0].Activities {
		if act.Notes != "keep me" || act.DurationMinutes != 30 {
			t.Fatal("apply changed notes/duration")
		}
	}
	// Two simultaneous approvals of the same snapshot: exactly one commits.
	id = preview()
	statuses := make(chan int, 2)
	for i := 0; i < 2; i++ {
		go func() {
			req := httptest.NewRequest("POST", dayPath+"/optimization-apply", strings.NewReader(applyBody(id)))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer user-a")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			statuses <- rec.Code
		}()
	}
	first, second := <-statuses, <-statuses
	if !((first == 200 && second == 409) || (first == 409 && second == 200)) {
		t.Fatalf("concurrent statuses %d %d", first, second)
	}
}
