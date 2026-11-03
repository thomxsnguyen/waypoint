package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"waypoint/internal/auth"
	"waypoint/internal/domain"
	"waypoint/internal/places"
	"waypoint/internal/store"
)

type API struct {
	Store  *store.Store
	Auth   auth.Verifier
	Places places.Provider
	Origin string
	limits limiter
}
type userKey struct{}

func user(r *http.Request) string { return r.Context().Value(userKey{}).(string) }

type endpoint func(http.ResponseWriter, *http.Request) error

func write(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if status != 204 {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": value})
	}
}
func failure(w http.ResponseWriter, err error) {
	e := &domain.Error{Status: 500, Code: "INTERNAL_ERROR", Message: "Something went wrong. Please try again."}
	if !errors.As(err, &e) {
		slog.Error("request failed", "error", err)
		e = &domain.Error{Status: 500, Code: "INTERNAL_ERROR", Message: "Something went wrong. Please try again."}
	}
	if e.Status == 429 {
		w.Header().Set("Retry-After", "60")
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.Status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": e.Code, "message": e.Message}})
}
func decode(w http.ResponseWriter, r *http.Request, out any) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return domain.Invalid("Content-Type must be application/json.")
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	body, err := io.ReadAll(r.Body)
	if err != nil || !bytes.HasPrefix(bytes.TrimSpace(body), []byte("{")) {
		return domain.Invalid("Body must be a JSON object smaller than 32 KiB.")
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return domain.Invalid("Invalid JSON body or unsupported field.")
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return domain.Invalid("Body must contain one JSON object.")
	}
	return nil
}
func (a *API) wrap(fn endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			failure(w, err)
		}
	}
}
func (a *API) protected(fn endpoint) http.HandlerFunc {
	return a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		token := r.Header.Get("Authorization")
		if !strings.HasPrefix(token, "Bearer ") {
			return &domain.Error{Status: 401, Code: "UNAUTHENTICATED", Message: "Please sign in to continue."}
		}
		subject, err := a.Auth.Verify(r.Context(), strings.TrimPrefix(token, "Bearer "))
		if err != nil {
			return &domain.Error{Status: 401, Code: "UNAUTHENTICATED", Message: "Your session has expired. Please sign in again."}
		}
		if !a.limits.allow(subject, 1, 240) {
			return &domain.Error{Status: 429, Code: "RATE_LIMITED", Message: "Too many requests. Please wait a minute."}
		}
		id, err := a.Store.User(r.Context(), subject)
		if err != nil {
			return err
		}
		return fn(w, r.WithContext(context.WithValue(r.Context(), userKey{}, id)))
	})
}
func (a *API) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if err := a.Store.Pool.Ping(r.Context()); err != nil {
			return &domain.Error{Status: 503, Code: "DATABASE_UNAVAILABLE", Message: "Database unavailable."}
		}
		write(w, 200, map[string]string{"status": "ok"})
		return nil
	}))
	mux.HandleFunc("GET /api/v1/trips", a.protected(a.listTrips))
	mux.HandleFunc("POST /api/v1/trips", a.protected(a.createTrip))
	mux.HandleFunc("GET /api/v1/trips/{trip}", a.protected(a.getTrip))
	mux.HandleFunc("PATCH /api/v1/trips/{trip}", a.protected(a.updateTrip))
	mux.HandleFunc("DELETE /api/v1/trips/{trip}", a.protected(a.deleteTrip))
	mux.HandleFunc("GET /api/v1/places/search", a.protected(a.search))
	mux.HandleFunc("GET /api/v1/trips/{trip}/places", a.protected(a.listPlaces))
	mux.HandleFunc("POST /api/v1/trips/{trip}/places", a.protected(a.savePlace))
	mux.HandleFunc("PATCH /api/v1/trips/{trip}/places/{place}", a.protected(a.updatePlace))
	mux.HandleFunc("DELETE /api/v1/trips/{trip}/places/{place}", a.protected(a.deletePlace))
	mux.HandleFunc("GET /api/v1/trips/{trip}/days", a.protected(a.days))
	mux.HandleFunc("POST /api/v1/days/{day}/activities", a.protected(a.addActivity))
	mux.HandleFunc("PATCH /api/v1/activities/{activity}", a.protected(a.updateActivity))
	mux.HandleFunc("DELETE /api/v1/activities/{activity}", a.protected(a.deleteActivity))
	mux.HandleFunc("PUT /api/v1/days/{day}/order", a.protected(a.order))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { failure(w, domain.NotFound()) })
	return a.middleware(mux)
}
func (a *API) listTrips(w http.ResponseWriter, r *http.Request) error {
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 || n > 100000 {
			return domain.Invalid("Invalid pagination offset.")
		}
		offset = n
	}
	data, err := a.Store.Trips(r.Context(), user(r), offset)
	if err == nil {
		write(w, 200, data)
	}
	return err
}
func (a *API) createTrip(w http.ResponseWriter, r *http.Request) error {
	var in domain.TripInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	data, err := a.Store.CreateTrip(r.Context(), user(r), in)
	if err == nil {
		write(w, 201, data)
	}
	return err
}
func (a *API) getTrip(w http.ResponseWriter, r *http.Request) error {
	data, err := a.Store.Trip(r.Context(), user(r), r.PathValue("trip"))
	if err == nil {
		write(w, 200, data)
	}
	return err
}
func (a *API) updateTrip(w http.ResponseWriter, r *http.Request) error {
	var in store.TripPatch
	if err := decode(w, r, &in); err != nil {
		return err
	}
	data, err := a.Store.UpdateTrip(r.Context(), user(r), r.PathValue("trip"), in)
	if err == nil {
		write(w, 200, data)
	}
	return err
}
func (a *API) deleteTrip(w http.ResponseWriter, r *http.Request) error {
	err := a.Store.DeleteTrip(r.Context(), user(r), r.PathValue("trip"))
	if err == nil {
		write(w, 204, nil)
	}
	return err
}
func (a *API) providerBudget(r *http.Request, cost int) error {
	if !a.limits.allow("places:"+user(r), cost, 240) || !a.limits.allow("places:global", cost, 2000) {
		return &domain.Error{Status: 429, Code: "RATE_LIMITED", Message: "Place lookup limit reached. Try again in a minute."}
	}
	return nil
}
func (a *API) search(w http.ResponseWriter, r *http.Request) error {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 2 || len(q) > 200 {
		return domain.Invalid("Search must contain 2–200 characters.")
	}
	trip, err := a.Store.Trip(r.Context(), user(r), r.URL.Query().Get("trip_id"))
	if err != nil {
		return err
	}
	if err = a.providerBudget(r, 1); err != nil {
		return err
	}
	data, err := a.Places.Search(r.Context(), q, trip.Destination)
	if err == nil {
		write(w, 200, data)
	}
	return err
}
func (a *API) listPlaces(w http.ResponseWriter, r *http.Request) error {
	data, err := a.Store.Places(r.Context(), user(r), r.PathValue("trip"))
	if err != nil {
		return err
	}
	// Return durable records even during provider outages / rate limiting.
	if r.URL.Query().Get("details") == "false" {
		write(w, 200, data)
		return nil
	}
	if err = a.providerBudget(r, len(data)); err != nil {
		for i := range data {
			data[i].Unavailable = true
		}
		write(w, 200, data)
		return nil
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	jobs := make(chan int)
	var wg sync.WaitGroup
	for n := 0; n < 5; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				p, e := a.Places.Details(ctx, data[i].ProviderPlaceID)
				if e != nil {
					data[i].Unavailable = true
					slog.Warn("place details unavailable", "error", e)
				} else {
					data[i].Details = &p
				}
			}
		}()
	}
	for i := range data {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	write(w, 200, data)
	return nil
}
func (a *API) savePlace(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		ProviderPlaceID string `json:"provider_place_id"`
		Notes           string `json:"notes"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	data, err := a.Store.SavePlace(r.Context(), user(r), r.PathValue("trip"), in.ProviderPlaceID, in.Notes)
	if err == nil {
		write(w, 201, data)
	}
	return err
}
func (a *API) updatePlace(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Notes *string `json:"notes"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if in.Notes == nil {
		return domain.Invalid("Notes are required.")
	}
	err := a.Store.ChangePlace(r.Context(), user(r), r.PathValue("trip"), r.PathValue("place"), in.Notes)
	if err == nil {
		write(w, 200, map[string]string{"notes": *in.Notes})
	}
	return err
}
func (a *API) deletePlace(w http.ResponseWriter, r *http.Request) error {
	err := a.Store.ChangePlace(r.Context(), user(r), r.PathValue("trip"), r.PathValue("place"), nil)
	if err == nil {
		write(w, 204, nil)
	}
	return err
}
func (a *API) days(w http.ResponseWriter, r *http.Request) error {
	data, err := a.Store.Days(r.Context(), user(r), r.PathValue("trip"))
	if err == nil {
		write(w, 200, data)
	}
	return err
}
func (a *API) addActivity(w http.ResponseWriter, r *http.Request) error {
	var in domain.ActivityInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	data, err := a.Store.AddActivity(r.Context(), user(r), r.PathValue("day"), in)
	if err == nil {
		write(w, 201, data)
	}
	return err
}
func (a *API) updateActivity(w http.ResponseWriter, r *http.Request) error {
	var in store.ActivityPatch
	if err := decode(w, r, &in); err != nil {
		return err
	}
	data, err := a.Store.UpdateActivity(r.Context(), user(r), r.PathValue("activity"), in)
	if err == nil {
		write(w, 200, data)
	}
	return err
}
func (a *API) deleteActivity(w http.ResponseWriter, r *http.Request) error {
	err := a.Store.DeleteActivity(r.Context(), user(r), r.PathValue("activity"))
	if err == nil {
		write(w, 204, nil)
	}
	return err
}
func (a *API) order(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		IDs []string `json:"activity_ids"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if in.IDs == nil {
		return domain.Invalid("activity_ids must be an array.")
	}
	err := a.Store.Reorder(r.Context(), user(r), r.PathValue("day"), in.IDs)
	if err == nil {
		write(w, 200, map[string]bool{"saved": true})
	}
	return err
}

type bucket struct {
	start time.Time
	count int
}
type limiter struct {
	mu      sync.Mutex
	buckets map[string]bucket
}

func (l *limiter) allow(key string, cost, max int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if l.buckets == nil {
		l.buckets = map[string]bucket{}
	}
	if len(l.buckets) > 10000 {
		for k, b := range l.buckets {
			if now.Sub(b.start) > time.Minute {
				delete(l.buckets, k)
			}
		}
		if len(l.buckets) > 10000 {
			return false
		}
	}
	b := l.buckets[key]
	if now.Sub(b.start) > time.Minute {
		b = bucket{start: now}
	}
	if b.count+cost > max {
		return false
	}
	b.count += cost
	l.buckets[key] = b
	return true
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (a *API) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		id := uuid.NewString()
		rw := &responseWriter{w, 200}
		rw.Header().Set("X-Request-ID", id)
		rw.Header().Set("X-Content-Type-Options", "nosniff")
		rw.Header().Set("Cache-Control", "no-store")
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("panic", "request_id", id)
				failure(rw, errors.New("request panic"))
			}
			slog.Info("http request", "request_id", id, "method", r.Method, "route", r.Pattern, "status", rw.status, "duration_ms", time.Since(start).Milliseconds())
		}()
		origin := r.Header.Get("Origin")
		if origin != "" {
			if origin != a.Origin {
				failure(rw, &domain.Error{Status: 403, Code: "FORBIDDEN_ORIGIN", Message: "Origin is not allowed."})
				return
			}
			rw.Header().Set("Access-Control-Allow-Origin", a.Origin)
			rw.Header().Set("Vary", "Origin")
			rw.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			rw.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		}
		if r.Method == "OPTIONS" {
			rw.WriteHeader(204)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
		next.ServeHTTP(rw, r)
	})
}
