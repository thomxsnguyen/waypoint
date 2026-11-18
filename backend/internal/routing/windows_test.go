package routing

import (
	"context"
	"testing"
)

func TestVisitingWindows(t *testing.T) {
	fixed := 550
	for _, tc := range []struct {
		name     string
		window   Window
		fixed    *int
		start    int
		conflict bool
	}{
		{"wait and exact close", Window{600, 630}, nil, 600, false},
		{"overrun", Window{600, 629}, nil, 600, true},
		{"fixed before opening", Window{600, 660}, &fixed, 600, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Evaluate([]Stop{{ID: "a", Duration: 30, Window: &tc.window, FixedStart: tc.fixed}}, [][]int{{0}}, []int{0}, 540, 720)
			if err != nil || got.Visits[0].Start != tc.start || (len(got.Conflicts) > 0) != tc.conflict {
				t.Fatalf("%+v %v", got, err)
			}
		})
	}
	for _, w := range []Window{{-1, 30}, {600, 600}, {700, 600}, {0, 1441}} {
		if _, err := Evaluate([]Stop{{ID: "a", Duration: 30, Window: &w}}, [][]int{{0}}, []int{0}, 0, 1440); err == nil {
			t.Fatalf("accepted %+v", w)
		}
	}
}
func TestWindowForcesFeasibleOrder(t *testing.T) {
	stops := []Stop{{ID: "flexible", Duration: 60}, {ID: "closes early", Duration: 30, Window: &Window{540, 570}}}
	got, err := Propose(context.Background(), stops, [][]int{{0, 10}, {10, 0}}, 540, 720)
	if err != nil || !got.Changed || len(got.Proposed.Conflicts) != 0 || got.Proposed.Visits[0].ID != "closes early" || got.Savings != nil {
		t.Fatalf("%+v %v", got, err)
	}
	stops[1].Window.End = 560
	got, err = Propose(context.Background(), stops, [][]int{{0, 10}, {10, 0}}, 540, 720)
	if err != nil || len(got.Proposed.Conflicts) == 0 || len(got.Proposed.Visits) != 2 {
		t.Fatalf("%+v %v", got, err)
	}
}
