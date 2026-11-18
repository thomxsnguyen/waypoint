package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"
	"waypoint/internal/domain"
	"waypoint/internal/routing"
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
		Mode  routing.TravelMode `json:"travel_mode"`
		Start *int               `json:"start_minute"`
		End   *int               `json:"end_minute"`
		Fixed []string           `json:"fixed_activity_ids"`
	}
	if err = decode(w, r, &in); err != nil {
		return err
	}
	if in.Start == nil || in.End == nil || *in.Start < 0 || *in.End > 1440 || *in.Start >= *in.End {
		return domain.Invalid("Provide increasing day bounds between 0 and 1440 minutes.")
	}
	if in.Mode != routing.Walk && in.Mode != routing.Drive {
		return domain.Invalid("Choose WALK or DRIVE.")
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
	stops := make([]routing.Stop, len(day.Activities))
	for i, activity := range day.Activities {
		stops[i] = routing.Stop{ID: activity.ID, Duration: activity.DurationMinutes}
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
	cost := len(ids) * len(ids)
	if len(ids) < 2 {
		cost = 0
	}
	if !a.limits.allow("routes:"+user(r), cost, 800) || !a.limits.allow("routes:global", cost, 4000) {
		return &domain.Error{Status: 429, Code: "ROUTES_RATE_LIMIT", Message: "Travel-time lookup limit reached. Try again in a minute."}
	}
	matrix, err := a.Routes.Matrix(r.Context(), ids, in.Mode)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &domain.Error{Status: 503, Code: "ROUTES_UNAVAILABLE", Message: "Travel-time estimates could not complete. Try again."}
	}
	if err != nil {
		return err
	}
	proposal, err := routing.Propose(r.Context(), stops, matrix, *in.Start, *in.End)
	if err != nil {
		return err
	}
	write(w, 200, map[string]any{
		"day_id": day.ID, "timezone": trip.Timezone, "travel_mode": in.Mode,
		"original": scheduleDTO(proposal.Original), "proposed": scheduleDTO(proposal.Proposed),
		"changed": proposal.Changed, "savings_minutes": proposal.Savings, "explanation": proposal.Explanation,
		"assumptions": []string{"Static travel estimates rounded up to whole minutes.", "No travel to the first stop or return leg is included.", "Only explicitly selected start times are fixed; opening hours are not checked.", "This heuristic does not guarantee the best route. Preview does not save changes."},
	})
	return nil
}
