package routing

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestProposalReducesDirectedTravel(t *testing.T) {
	stops := []Stop{{"a", 30, nil}, {"b", 30, nil}, {"c", 30, nil}}
	matrix := [][]int{{0, 50, 5}, {40, 0, 50}, {40, 5, 0}}
	before := [][]int{{0, 50, 5}, {40, 0, 50}, {40, 5, 0}}
	got, err := Propose(context.Background(), stops, matrix, 540, 1020)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Changed || !reflect.DeepEqual(got.Order, []int{0, 2, 1}) || got.Savings == nil || *got.Savings != 90 {
		t.Fatalf("%+v", got)
	}
	if got.Original.TravelMinutes != 100 || got.Proposed.TravelMinutes != 10 || len(got.Proposed.Conflicts) != 0 {
		t.Fatalf("%+v", got)
	}
	if !reflect.DeepEqual(matrix, before) || stops[0].ID != "a" || stops[1].ID != "b" {
		t.Fatal("inputs mutated")
	}
	// Repeated runs must select the same proposal.
	again, err := Propose(context.Background(), stops, matrix, 540, 1020)
	if err != nil || !reflect.DeepEqual(got, again) {
		t.Fatal("nondeterministic proposal")
	}
}

func TestProposalRepairsFixedStartConflict(t *testing.T) {
	fixed := 550
	stops := []Stop{{"flexible", 60, nil}, {"reservation", 30, &fixed}}
	got, err := Propose(context.Background(), stops, [][]int{{0, 10}, {10, 0}}, 540, 720)
	if err != nil || !got.Changed || len(got.Original.Conflicts) == 0 || len(got.Proposed.Conflicts) != 0 || got.Savings != nil {
		t.Fatalf("%+v %v", got, err)
	}
	if got.Proposed.Visits[0].ID != "reservation" || got.Proposed.Visits[0].Start != 550 || len(got.Proposed.Visits) != 2 {
		t.Fatal("reservation or activity lost")
	}
}

func TestProposalKeepsOriginalWhenImpossibleOrEqual(t *testing.T) {
	stops := []Stop{{"a", 60, nil}, {"b", 60, nil}}
	for _, tc := range []struct {
		name     string
		matrix   [][]int
		end      int
		feasible bool
	}{
		{"equal", [][]int{{0, 10}, {10, 0}}, 720, true},
		{"unavailable", [][]int{{0, -1}, {-1, 0}}, 720, false},
		{"impossible day", [][]int{{0, 10}, {10, 0}}, 600, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Propose(context.Background(), stops, tc.matrix, 540, tc.end)
			if err != nil || got.Changed || !reflect.DeepEqual(got.Order, []int{0, 1}) || len(got.Proposed.Visits) != 2 {
				t.Fatalf("%+v %v", got, err)
			}
			if tc.feasible != (got.Savings != nil) {
				t.Fatal("misleading savings")
			}
			if !tc.feasible && len(got.Proposed.Conflicts) == 0 {
				t.Fatal("hidden conflicts")
			}
		})
	}
}

func TestProposalCanAvoidUnavailableLeg(t *testing.T) {
	stops := []Stop{{"a", 10, nil}, {"b", 10, nil}}
	got, err := Propose(context.Background(), stops, [][]int{{0, -1}, {5, 0}}, 0, 60)
	if err != nil || !got.Changed || got.Proposed.TravelMinutes != 5 || got.Savings != nil {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestProposalValidationAndCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Propose(ctx, nil, nil, 0, 60); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := Propose(context.Background(), make([]Stop, 21), nil, 0, 60); err == nil {
		t.Fatal("accepted excessive size")
	}
	if _, err := Propose(context.Background(), []Stop{{"a", 0, nil}}, [][]int{{0}}, 0, 60); err == nil {
		t.Fatal("accepted invalid stop")
	}
	empty, err := Propose(context.Background(), nil, nil, 0, 60)
	if err != nil || empty.Changed || len(empty.Proposed.Visits) != 0 {
		t.Fatalf("%+v %v", empty, err)
	}
	single, err := Propose(context.Background(), []Stop{{"a", 10, nil}}, [][]int{{0}}, 0, 60)
	if err != nil || single.Changed || single.Proposed.TravelMinutes != 0 {
		t.Fatalf("%+v %v", single, err)
	}
}
