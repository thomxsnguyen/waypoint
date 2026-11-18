package routing

import (
	"reflect"
	"testing"
)

func TestDirectedTravelAndWaiting(t *testing.T) {
	fixed := 600
	stops := []Stop{{ID: "a", Duration: 30, FixedStart: nil}, {ID: "b", Duration: 60, FixedStart: &fixed}}
	matrix := [][]int{{0, 10}, {40, 0}}
	got, err := Evaluate(stops, matrix, []int{0, 1}, 540, 1020)
	if err != nil || got.TravelMinutes != 10 || len(got.Conflicts) != 0 || got.Visits[1].Start != 600 {
		t.Fatalf("%+v %v", got, err)
	}
	reverse, err := Evaluate(stops, matrix, []int{1, 0}, 540, 1020)
	if err != nil || reverse.TravelMinutes != 40 || reverse.Visits[1].Start != 700 {
		t.Fatalf("%+v %v", reverse, err)
	}
	if !reflect.DeepEqual(matrix, [][]int{{0, 10}, {40, 0}}) || fixed != 600 || stops[0].ID != "a" {
		t.Fatal("input changed")
	}
}

func TestConflictsPreserveEveryStop(t *testing.T) {
	fixed := 550
	stops := []Stop{{ID: "a", Duration: 60, FixedStart: nil}, {ID: "b", Duration: 60, FixedStart: &fixed}, {ID: "c", Duration: 20, FixedStart: nil}}
	for _, tc := range []struct {
		name   string
		matrix [][]int
		reason string
	}{
		{"fixed and day bounds", [][]int{{0, 10, 10}, {10, 0, 10}, {10, 10, 0}}, "fixed start cannot be met"},
		{"unreachable", [][]int{{0, -1, 10}, {10, 0, 10}, {10, 10, 0}}, "route unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Evaluate(stops, tc.matrix, []int{0, 1, 2}, 540, 620)
			if err != nil || len(got.Visits) != 3 || len(got.Conflicts) == 0 || got.Conflicts[0].Reason != tc.reason {
				t.Fatalf("%+v %v", got, err)
			}
			if tc.name == "unreachable" && (got.TravelMinutes != -1 || got.Visits[2].Start != -1) {
				t.Fatal("invented arrival or travel estimate")
			}
			if tc.name == "fixed and day bounds" && got.Conflicts[len(got.Conflicts)-1].Reason != "activity exceeds day boundary" {
				t.Fatal("missing boundary conflict")
			}
		})
	}
}

func TestRejectInvalidOrdersAndMatrices(t *testing.T) {
	stops := []Stop{{ID: "a", Duration: 30, FixedStart: nil}, {ID: "b", Duration: 30, FixedStart: nil}}
	for _, order := range [][]int{{0}, {0, 0}, {-1, 1}, {0, 2}} {
		if _, err := Evaluate(stops, [][]int{{0, 10}, {10, 0}}, order, 0, 1440); err == nil {
			t.Fatalf("accepted %v", order)
		}
	}
	for _, matrix := range [][][]int{nil, {{0}, {10, 0}}, {{0, -2}, {10, 0}}, {{1, 10}, {10, 0}}} {
		if _, err := Evaluate(stops, matrix, []int{0, 1}, 0, 1440); err == nil {
			t.Fatalf("accepted %v", matrix)
		}
	}
	stops[1].ID = "a"
	if _, err := Evaluate(stops, [][]int{{0, 10}, {10, 0}}, []int{0, 1}, 0, 1440); err == nil {
		t.Fatal("accepted duplicate IDs")
	}
}
