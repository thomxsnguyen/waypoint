package routing

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestTransitProviderContract(t *testing.T) {
	departure := time.Now().Add(time.Hour).Truncate(time.Second)
	g := NewGoogle("test")
	g.Client.Transport = transport(func(r *http.Request) (*http.Response, error) {
		var body struct {
			Mode       string `json:"travelMode"`
			Departure  string `json:"departureTime"`
			Preference string `json:"routingPreference"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Mode != "TRANSIT" || body.Departure != departure.UTC().Format(time.RFC3339) || body.Preference != "" {
			t.Fatalf("%+v", body)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(validMatrix))}, nil
	})
	if _, err := g.Matrix(context.Background(), []string{"a", "b"}, Transit, departure); err != nil {
		t.Fatal(err)
	}
	for _, date := range []time.Time{time.Time{}, time.Now().Add(-8 * 24 * time.Hour), time.Now().Add(101 * 24 * time.Hour)} {
		if _, err := g.Matrix(context.Background(), []string{"a", "b"}, Transit, date); err == nil {
			t.Fatal("invalid transit date")
		}
	}
	if _, err := g.Matrix(context.Background(), make([]string, 11), Transit, departure); err == nil {
		t.Fatal("excessive matrix accepted")
	}
	if _, err := TransitDeparture("2026-11-01", "America/Los_Angeles", 90); err == nil {
		t.Fatal("ambiguous DST accepted")
	}
	if _, err := TransitDeparture("2026-03-08", "America/Los_Angeles", 150); err == nil {
		t.Fatal("missing DST hour accepted")
	}
	got, err := TransitDeparture("2026-11-12", "Asia/Tokyo", 540)
	if err != nil || got.UTC().Format(time.RFC3339) != "2026-11-12T00:00:00Z" {
		t.Fatalf("%v %v", got, err)
	}
}

type timedRoutes struct {
	departures []time.Time
	late       bool
}

func (p *timedRoutes) Matrix(_ context.Context, ids []string, _ TravelMode, departures ...time.Time) ([][]int, error) {
	p.departures = append(p.departures, departures[0])
	leg := 5
	if p.late && ids[0] == "b" {
		leg = 200
	}
	return [][]int{{0, leg}, {leg, 0}}, nil
}
func TestTransitRechecksAfterWaitingAndFallsBack(t *testing.T) {
	departure := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	provider := &timedRoutes{}
	stops := []Stop{{ID: "a", Duration: 30, Window: &Window{600, 700}}, {ID: "b", Duration: 30}}
	initial, err := Propose(context.Background(), stops, [][]int{{0, 10}, {3, 0}}, 540, 720)
	if err != nil {
		t.Fatal(err)
	}
	p, err := RecheckTransit(context.Background(), provider, []string{"a", "b"}, "", stops, initial, departure, 540, 720)
	if err != nil || p.Changed || len(provider.departures) != 2 || provider.departures[0].Hour() != 10 || provider.departures[0].Minute() != 30 {
		t.Fatalf("%+v %v departures %v", p, err, provider.departures)
	}
	// Morning-sample candidate becomes infeasible at actual departures; original survives.
	provider = &timedRoutes{late: true}
	p, err = RecheckTransit(context.Background(), provider, []string{"a", "b"}, "", stops, initial, departure, 540, 720)
	if err != nil || p.Changed || len(p.Proposed.Conflicts) != 0 {
		t.Fatalf("%+v %v", p, err)
	}
	// Starting place leg must be checked at day departure as well.
	provider = &timedRoutes{}
	_, err = RecheckTransit(context.Background(), provider, []string{"a", "b"}, "hotel", stops, initial, departure, 540, 720)
	if err != nil || !provider.departures[0].Equal(departure) {
		t.Fatalf("%v %v", provider.departures, err)
	}
}
