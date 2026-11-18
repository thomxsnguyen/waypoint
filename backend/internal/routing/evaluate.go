// Package routing evaluates single-day schedules without changing persisted plans.
package routing

import "fmt"

// Stop uses local wall-clock minutes from midnight. FixedStart is optional.
type Stop struct {
	ID         string
	Duration   int
	FixedStart *int
	Window     *Window
}

type Window struct{ Start, End int }

func (w Window) Valid() bool { return w.Start >= 0 && w.End <= 1440 && w.Start < w.End }

type Options struct{ OriginMinutes []int }

type Visit struct {
	ID         string
	Start, End int
}

type Conflict struct {
	ActivityID string
	Reason     string
}

type Evaluation struct {
	Visits        []Visit
	Conflicts     []Conflict
	TravelMinutes int
}

// Evaluate requires an exact permutation of stops. Matrix indices refer to stops,
// not order; directed legs may differ. -1 represents an unavailable route.
// After an unavailable leg, downstream times cannot be estimated and use -1.
func Evaluate(stops []Stop, matrix [][]int, order []int, start, end int, options ...Options) (Evaluation, error) {
	result := Evaluation{Visits: []Visit{}, Conflicts: []Conflict{}}
	n := len(stops)
	if start < 0 || end > 1440 || start >= end || n > 100 || len(matrix) != n || len(order) != n {
		return result, fmt.Errorf("invalid day bounds or schedule dimensions")
	}
	origin := make([]int, n)
	if len(options) > 1 {
		return result, fmt.Errorf("invalid options")
	}
	if len(options) == 1 && options[0].OriginMinutes != nil {
		if len(options[0].OriginMinutes) != n {
			return result, fmt.Errorf("invalid origin vector")
		}
		origin = options[0].OriginMinutes
	}
	for _, minutes := range origin {
		if minutes < -1 || minutes > 1440 {
			return result, fmt.Errorf("invalid origin duration")
		}
	}
	seenIDs := map[string]bool{}
	seen := make([]bool, n)
	for i, stop := range stops {
		if stop.ID == "" || seenIDs[stop.ID] || stop.Duration < 1 || stop.Duration > 1440 || len(matrix[i]) != n {
			return result, fmt.Errorf("invalid stop or matrix row")
		}
		seenIDs[stop.ID] = true
		if stop.FixedStart != nil && (*stop.FixedStart < 0 || *stop.FixedStart >= 1440) {
			return result, fmt.Errorf("invalid fixed start")
		}
		if stop.Window != nil && !stop.Window.Valid() {
			return result, fmt.Errorf("invalid visiting window")
		}
		for j, minutes := range matrix[i] {
			if minutes < -1 || minutes > 1440 || (i == j && minutes != 0) {
				return result, fmt.Errorf("invalid travel duration")
			}
		}
	}
	for _, index := range order {
		if index < 0 || index >= n || seen[index] {
			return result, fmt.Errorf("order must be an exact permutation")
		}
		seen[index] = true
	}
	clock := start
	reachable := true
	for position, index := range order {
		stop := stops[index]
		leg := origin[index]
		if position > 0 {
			leg = matrix[order[position-1]][index]
		}
		{

			if leg < 0 {
				reachable = false
				result.Conflicts = append(result.Conflicts, Conflict{stop.ID, "route unavailable"})
			} else {
				result.TravelMinutes += leg
				clock += leg
			}
		}
		if !reachable {
			result.Visits = append(result.Visits, Visit{stop.ID, -1, -1})
			result.Conflicts = append(result.Conflicts, Conflict{stop.ID, "arrival cannot be estimated"})
			continue
		}
		if stop.Window != nil && clock < stop.Window.Start {
			clock = stop.Window.Start
		}
		if stop.FixedStart != nil {
			if clock > *stop.FixedStart {
				result.Conflicts = append(result.Conflicts, Conflict{stop.ID, "fixed start cannot be met"})
			} else {
				clock = *stop.FixedStart
			}
		}
		visit := Visit{stop.ID, clock, clock + stop.Duration}
		clock = visit.End
		result.Visits = append(result.Visits, visit)
		if stop.Window != nil && clock > stop.Window.End {
			result.Conflicts = append(result.Conflicts, Conflict{stop.ID, "activity exceeds visiting window"})
		}
		if clock > end {
			result.Conflicts = append(result.Conflicts, Conflict{stop.ID, "activity exceeds day boundary"})
		}
	}
	// A partial route sum must never be presented as a complete travel estimate.
	if !reachable {
		result.TravelMinutes = -1
	}
	return result, nil
}
