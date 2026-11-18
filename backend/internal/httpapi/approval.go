package httpapi

import (
	"github.com/google/uuid"
	"net/http"
	"sync"
	"time"
	"waypoint/internal/domain"
	"waypoint/internal/store"
)

type approval struct {
	Owner, Trip, Day, Snapshot string
	Visits                     []store.ApprovedVisit
	Expires                    time.Time
}
type approvalCache struct {
	mu      sync.Mutex
	entries map[string]approval
}

func (c *approvalCache) issue(value approval) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = map[string]approval{}
	}
	now := time.Now()
	for id, p := range c.entries {
		if !now.Before(p.Expires) {
			delete(c.entries, id)
		}
	}
	if len(c.entries) >= 1000 {
		return "", &domain.Error{Status: 503, Code: "PREVIEW_CAPACITY", Message: "Preview capacity reached. Try again later."}
	}
	id := uuid.NewString()
	c.entries[id] = value
	return id, nil
}
func (c *approvalCache) get(id string) (approval, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	p, ok := c.entries[id]
	if !ok || !time.Now().Before(p.Expires) {
		delete(c.entries, id)
		return approval{}, false
	}
	return p, true
}
func (c *approvalCache) remove(id string) { c.mu.Lock(); defer c.mu.Unlock(); delete(c.entries, id) }
func (a *API) optimizationApply(w http.ResponseWriter, r *http.Request) error {
	trip, err := a.Store.Trip(r.Context(), user(r), r.PathValue("trip"))
	if err != nil {
		return err
	}
	var in struct {
		ID string `json:"approval_id"`
	}
	if err = decode(w, r, &in); err != nil {
		return err
	}
	p, ok := a.approvals.get(in.ID)
	if !ok || p.Owner != user(r) || p.Trip != trip.ID || p.Day != r.PathValue("day") {
		return domain.Conflict("Preview is unavailable or expired. Generate a new preview.")
	}
	if err = a.Store.ApplyOptimization(r.Context(), user(r), trip.ID, p.Day, p.Snapshot, p.Visits); err != nil {
		return err
	}
	a.approvals.remove(in.ID)
	write(w, 200, map[string]bool{"saved": true})
	return nil
}
