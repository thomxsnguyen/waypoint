package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"
	"waypoint/internal/domain"
	"waypoint/internal/routing"
	"waypoint/internal/store"
)

type previewSchedule struct {
	Visits        []previewVisit    `json:"visits"`
	Conflicts     []previewConflict `json:"conflicts"`
	TravelMinutes int               `json:"travel_minutes"`
}
type previewVisit struct {
	ActivityID string `json:"activity_id"`
	Start      int    `json:"start_minute"`
	End        int    `json:"end_minute"`
}
type previewConflict struct {
	ActivityID string `json:"activity_id"`
	Reason     string `json:"reason"`
}

func scheduleDTO(e routing.Evaluation) previewSchedule {
	out := previewSchedule{Visits: []previewVisit{}, Conflicts: []previewConflict{}, TravelMinutes: e.TravelMinutes}
	for _, v := range e.Visits {
		out.Visits = append(out.Visits, previewVisit{v.ID, v.Start, v.End})
	}
	for _, c := range e.Conflicts {
		out.Conflicts = append(out.Conflicts, previewConflict{c.ActivityID, c.Reason})
	}
	return out
}
func (a *API) optimizationPreview(w http.ResponseWriter, r *http.Request) error {
	snapshot, err := a.Store.OptimizationSnapshot(r.Context(), user(r), r.PathValue("trip"), r.PathValue("day"))
	if err != nil {
		return err
	}
	trip, err := a.Store.Trip(r.Context(), user(r), r.PathValue("trip"))
	if err != nil {
		return err
	}
	days, err := a.Store.Days(r.Context(), user(r), trip.ID)
	if err != nil {
		return err
	}
	var day *domain.Day
	for i := range days {
		if days[i].ID == r.PathValue("day") {
			day = &days[i]
			break
		}
	}
	if day == nil {
		return domain.NotFound()
	}
	var in struct {
		Origin  string             `json:"origin_saved_place_id"`
		Mode    routing.TravelMode `json:"travel_mode"`
		Start   *int               `json:"start_minute"`
		End     *int               `json:"end_minute"`
		Fixed   []string           `json:"fixed_activity_ids"`
		Windows []struct {
			ID    string `json:"activity_id"`
			Start *int   `json:"start_minute"`
			End   *int   `json:"end_minute"`
		} `json:"visiting_windows"`
	}
	if err = decode(w, r, &in); err != nil {
		return err
	}
	if in.Start == nil || in.End == nil || *in.Start < 0 || *in.End > 1440 || *in.Start >= *in.End {
		return domain.Invalid("Provide increasing day bounds between 0 and 1440 minutes.")
	}
	if in.Mode != routing.Walk && in.Mode != routing.Drive && in.Mode != routing.Transit {
		return domain.Invalid("Choose WALK, DRIVE, or TRANSIT.")
	}
	if len(day.Activities) > 20 {
		return domain.Invalid("Optimization supports at most 20 activities per day.")
	}
	fixed := map[string]bool{}
	for _, id := range in.Fixed {
		if fixed[id] {
			return domain.Invalid("Fixed activity IDs must be unique.")
		}
		fixed[id] = true
	}
	windows := map[string]*routing.Window{}
	for _, window := range in.Windows {
		if window.Start == nil || window.End == nil {
			return domain.Invalid("Visiting windows require both bounds.")
		}
		value := routing.Window{Start: *window.Start, End: *window.End}
		if !value.Valid() || windows[window.ID] != nil {
			return domain.Invalid("Provide unique visiting windows with increasing bounds between 0 and 1440.")
		}
		windows[window.ID] = &value
	}
	stops := make([]routing.Stop, len(day.Activities))
	for i, activity := range day.Activities {
		stops[i] = routing.Stop{ID: activity.ID, Duration: activity.DurationMinutes, Window: windows[activity.ID]}
		delete(windows, activity.ID)
		if fixed[activity.ID] {
			if activity.StartTime == nil {
				return domain.Invalid("A fixed activity must have a start time.")
			}
			parsed, e := time.Parse("15:04", *activity.StartTime)
			if e != nil {
				return e
			}
			minute := parsed.Hour()*60 + parsed.Minute()
			stops[i].FixedStart = &minute
			delete(fixed, activity.ID)
		}
	}
	if len(windows) > 0 {
		return domain.Invalid("Visiting windows must belong to the selected day.")
	}
	if len(fixed) > 0 {
		return domain.Invalid("Fixed activities must belong to the selected day.")
	}
	if a.Routes == nil {
		return &domain.Error{Status: 503, Code: "ROUTES_UNAVAILABLE", Message: "Travel-time estimates are not configured."}
	}
	places, err := a.Store.Places(r.Context(), user(r), trip.ID)
	if err != nil {
		return err
	}
	byID := map[string]string{}
	for _, p := range places {
		byID[p.ID] = p.ProviderPlaceID
	}
	ids := make([]string, len(stops))
	for i, activity := range day.Activities {
		ids[i] = byID[activity.SavedPlaceID]
		if ids[i] == "" {
			return domain.Conflict("The itinerary changed. Refresh before requesting a preview.")
		}
	}
	var originID string
	if in.Origin != "" {
		originID = byID[in.Origin]
		if originID == "" {
			return domain.Invalid("Starting place must belong to this trip.")
		}
		ids = append(ids, originID)
	}
	var departure []time.Time
	if in.Mode == routing.Transit {
		if len(ids) > 10 {
			return domain.Invalid("Transit supports at most 10 locations including the starting place.")
		}
		value, e := routing.TransitDeparture(day.Date, trip.Timezone, *in.Start)
		if e != nil {
			return e
		}
		if !routing.TransitDateSupported(value, time.Now()) {
			return domain.Invalid("Transit is available from 7 days past through 100 days future.")
		}
		departure = []time.Time{value}
	}
	cost := len(ids) * len(ids)
	if len(ids) < 2 {
		cost = 0
	}
	if in.Mode == routing.Transit {
		cost += 8 * len(stops)
	}
	if !a.limits.allow("routes:"+user(r), cost, 800) || !a.limits.allow("routes:global", cost, 4000) {
		return &domain.Error{Status: 429, Code: "ROUTES_RATE_LIMIT", Message: "Travel-time lookup limit reached. Try again in a minute."}
	}
	matrix, err := a.Routes.Matrix(r.Context(), ids, in.Mode, departure...)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &domain.Error{Status: 503, Code: "ROUTES_UNAVAILABLE", Message: "Travel-time estimates could not complete. Try again."}
	}
	if err != nil {
		return err
	}
	options := routing.Options{}
	if originID != "" {
		n := len(stops)
		if len(matrix) != n+1 || len(matrix[n]) != n+1 {
			return errors.New("invalid origin matrix")
		}
		options.OriginMinutes = matrix[n][:n]
		matrix = matrix[:n]
		for i := range matrix {
			if len(matrix[i]) != n+1 {
				return errors.New("invalid matrix")
			}
			matrix[i] = matrix[i][:n]
		}
	}
	proposal, err := routing.Propose(r.Context(), stops, matrix, *in.Start, *in.End, options)
	if err != nil {
		return err
	}
	if in.Mode == routing.Transit {
		proposal, err = routing.RecheckTransit(r.Context(), a.Routes, ids[:len(stops)], originID, stops, proposal, departure[0], *in.Start, *in.End)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return &domain.Error{Status: 503, Code: "ROUTES_UNAVAILABLE", Message: "Transit rechecking timed out. Try again."}
		}
		if err != nil {
			return err
		}
	}
	current, err := a.Store.OptimizationSnapshot(r.Context(), user(r), trip.ID, day.ID)
	if err != nil {
		return err
	}
	if current != snapshot {
		return domain.Conflict("The itinerary changed. Generate a new preview.")
	}
	var approvalID any
	var expiresAt any
	if len(proposal.Proposed.Conflicts) == 0 {
		expires := time.Now().Add(10 * time.Minute)
		visits := make([]store.ApprovedVisit, len(proposal.Proposed.Visits))
		for i, v := range proposal.Proposed.Visits {
			visits[i] = store.ApprovedVisit{ID: v.ID, Start: v.Start}
		}
		id, e := a.approvals.issue(approval{Owner: user(r), Trip: trip.ID, Day: day.ID, Snapshot: snapshot, Visits: visits, Expires: expires})
		if e != nil {
			return e
		}
		approvalID = id
		expiresAt = expires.UTC().Format(time.RFC3339)
	}
	travelAssumption := "Static travel estimates rounded up to whole minutes."
	if in.Mode == routing.Transit {
		travelAssumption = "Transit legs rechecked at scheduled departures; service availability can change. Rounded up to whole minutes."
	}
	originAssumption := "No travel to the first stop or return leg is included."
	if originID != "" {
		originAssumption = "Travel from the selected starting place is included; no return leg is included."
	}
	write(w, 200, map[string]any{
		"origin_saved_place_id": in.Origin,
		"visiting_windows":      in.Windows,
		"approval_id":           approvalID, "expires_at": expiresAt,
		"day_id": day.ID, "timezone": trip.Timezone, "travel_mode": in.Mode,
		"original": scheduleDTO(proposal.Original), "proposed": scheduleDTO(proposal.Proposed),
		"changed": proposal.Changed, "savings_minutes": proposal.Savings, "explanation": proposal.Explanation,
		"assumptions": []string{travelAssumption, originAssumption, "Only selected start times and supplied visiting windows are enforced; venue opening hours are not verified.", "This heuristic does not guarantee the best route. Preview does not save changes."},
	})
	return nil
}
