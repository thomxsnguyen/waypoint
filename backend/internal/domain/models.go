package domain

import (
	"strings"
	"time"
	_ "time/tzdata"
)

type Error struct {
	Status        int
	Code, Message string
}

func (e *Error) Error() string      { return e.Message }
func Invalid(message string) error  { return &Error{400, "VALIDATION_ERROR", message} }
func NotFound() error               { return &Error{404, "NOT_FOUND", "Resource not found."} }
func Conflict(message string) error { return &Error{409, "CONFLICT", message} }

type Trip struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Destination string `json:"destination"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	Timezone    string `json:"timezone"`
	Description string `json:"description"`
}
type TripInput struct {
	Name        string `json:"name"`
	Destination string `json:"destination"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	Timezone    string `json:"timezone"`
	Description string `json:"description"`
}

func (t *TripInput) Validate() error {
	t.Name = strings.TrimSpace(t.Name)
	t.Destination = strings.TrimSpace(t.Destination)
	if len(t.Name) == 0 || len(t.Name) > 120 {
		return Invalid("Trip name must contain 1–120 characters.")
	}
	if len(t.Destination) == 0 || len(t.Destination) > 200 {
		return Invalid("Destination must contain 1–200 characters.")
	}
	if len(t.Description) > 5000 {
		return Invalid("Description must be at most 5,000 characters.")
	}
	a, e1 := time.Parse(time.DateOnly, t.StartDate)
	b, e2 := time.Parse(time.DateOnly, t.EndDate)
	if e1 != nil || e2 != nil || a.Year() < 1 || b.Year() < 1 {
		return Invalid("Dates must use YYYY-MM-DD.")
	}
	if b.Before(a) {
		return Invalid("End date must not precede start date.")
	}
	if b.Sub(a) >= 366*24*time.Hour {
		return Invalid("Trips may contain at most 366 days.")
	}
	if t.Timezone == "" || t.Timezone == "Local" {
		return Invalid("Choose an IANA time zone, such as Asia/Tokyo.")
	}
	if _, err := time.LoadLocation(t.Timezone); err != nil {
		return Invalid("Choose a valid IANA time zone.")
	}
	return nil
}

type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}
type Attribution struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}
type Place struct {
	Provider        string        `json:"provider"`
	ProviderPlaceID string        `json:"provider_place_id"`
	Name            string        `json:"name"`
	Address         string        `json:"address"`
	Location        *Location     `json:"location"`
	ExternalURL     string        `json:"external_url"`
	Attributions    []Attribution `json:"attributions"`
}
type SavedPlace struct {
	ID              string `json:"id"`
	TripID          string `json:"trip_id"`
	ProviderPlaceID string `json:"provider_place_id"`
	Notes           string `json:"notes"`
	Details         *Place `json:"details"`
	Unavailable     bool   `json:"unavailable"`
}
type Activity struct {
	ID              string  `json:"id"`
	DayID           string  `json:"day_id"`
	SavedPlaceID    string  `json:"saved_place_id"`
	StartTime       *string `json:"start_time"`
	DurationMinutes int     `json:"duration_minutes"`
	Notes           string  `json:"notes"`
	Position        int     `json:"position"`
}
type Day struct {
	ID         string     `json:"id"`
	Date       string     `json:"date"`
	Activities []Activity `json:"activities"`
}
type ActivityInput struct {
	SavedPlaceID    string  `json:"saved_place_id"`
	StartTime       *string `json:"start_time"`
	DurationMinutes int     `json:"duration_minutes"`
	Notes           string  `json:"notes"`
}

func (a ActivityInput) Validate() error {
	if a.DurationMinutes < 1 || a.DurationMinutes > 1440 {
		return Invalid("Duration must be between 1 and 1,440 minutes.")
	}
	if len(a.Notes) > 5000 {
		return Invalid("Notes must be at most 5,000 characters.")
	}
	if a.StartTime != nil {
		if _, err := time.Parse("15:04", *a.StartTime); err != nil || len(*a.StartTime) != 5 {
			return Invalid("Start time must use HH:MM, or null.")
		}
	}
	return nil
}
func ValidateOrder(current, proposed []string) error {
	if len(current) != len(proposed) {
		return Invalid("Order must include every activity exactly once.")
	}
	ids := make(map[string]bool, len(current))
	for _, id := range current {
		ids[id] = true
	}
	for _, id := range proposed {
		if !ids[id] {
			return Invalid("Order contains a duplicate or unknown activity.")
		}
		delete(ids, id)
	}
	return nil
}
func ValidateNotes(notes string) error {
	if len(notes) > 5000 {
		return Invalid("Notes must be at most 5,000 characters.")
	}
	return nil
}
func ValidatePlaceID(id string) error {
	if len(id) < 1 || len(id) > 255 {
		return Invalid("A provider place ID is required.")
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return Invalid("Invalid provider place ID.")
		}
	}
	return nil
}
