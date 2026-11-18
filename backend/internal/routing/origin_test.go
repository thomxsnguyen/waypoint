package routing

import (
	"context"
	"testing"
)

func TestOriginTravel(t *testing.T) {
	stops := []Stop{{ID: "a", Duration: 30}, {ID: "b", Duration: 30}}
	matrix := [][]int{{0, 10}, {10, 0}}
	p, err := Propose(context.Background(), stops, matrix, 540, 720, Options{OriginMinutes: []int{40, 5}})
	if err != nil || !p.Changed || p.Proposed.Visits[0].ID != "b" || p.Proposed.Visits[0].Start != 545 || p.Proposed.TravelMinutes != 15 || *p.Savings != 35 {
		t.Fatalf("%+v %v", p, err)
	}
	p, err = Propose(context.Background(), stops, matrix, 540, 720, Options{OriginMinutes: []int{-1, -1}})
	if err != nil || len(p.Proposed.Conflicts) == 0 || p.Proposed.TravelMinutes != -1 {
		t.Fatalf("%+v %v", p, err)
	}
	if _, err = Propose(context.Background(), stops, matrix, 540, 720, Options{OriginMinutes: []int{3}}); err == nil {
		t.Fatal("invalid origin accepted")
	}
}
