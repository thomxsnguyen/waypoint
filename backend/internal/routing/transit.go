package routing

import (
	"context"
	"fmt"
	"slices"
	"time"
	"waypoint/internal/domain"
)

func TransitDateSupported(departure, now time.Time) bool {
	return !departure.IsZero() && !departure.Before(now.Add(-7*24*time.Hour)) && !departure.After(now.Add(100*24*time.Hour))
}

// TransitDeparture rejects DST-transition days because the planner's local
// minute model cannot distinguish repeated hours or represent skipped hours.
func TransitDeparture(date, zone string, minute int) (time.Time, error) {
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return time.Time{}, err
	}
	midnight, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return time.Time{}, err
	}
	if minute < 0 || minute >= 1440 {
		return time.Time{}, domain.Invalid("Invalid departure minute.")
	}
	if midnight.AddDate(0, 0, 1).Sub(midnight) != 24*time.Hour {
		return time.Time{}, domain.Invalid("Transit previews are unavailable on daylight-saving transition days. Choose walking or driving.")
	}
	return midnight.Add(time.Duration(minute) * time.Minute), nil
}

// RecheckTransit validates only the candidate and original route at actual leg
// departures. This bounded heuristic is not a time-dependent global search.
func RecheckTransit(ctx context.Context, provider MatrixProvider, ids []string, origin string, stops []Stop, p Proposal, departure time.Time, start, end int) (Proposal, error) {
	n := len(stops)
	originalOrder := make([]int, n)
	for i := range originalOrder {
		originalOrder[i] = i
	}
	evaluate := func(order []int) (Evaluation, error) {
		matrix := make([][]int, n)
		for i := range matrix {
			matrix[i] = make([]int, n)
			for j := range matrix[i] {
				if i != j {
					matrix[i][j] = -1
				}
			}
		}
		originMinutes := make([]int, n)
		clock := start
		for position, index := range order {
			if err := ctx.Err(); err != nil {
				return Evaluation{}, err
			}
			from := origin
			if position > 0 {
				from = ids[order[position-1]]
			}
			leg := 0
			if from != "" && from != ids[index] {
				if clock >= end || clock >= 1440 {
					leg = -1
				} else {
					result, err := provider.Matrix(ctx, []string{from, ids[index]}, Transit, departure.Add(time.Duration(clock-start)*time.Minute))
					if err != nil {
						return Evaluation{}, err
					}
					if len(result) != 2 || len(result[0]) != 2 {
						return Evaluation{}, fmt.Errorf("invalid transit recheck matrix")
					}
					leg = result[0][1]
				}
			}
			if position == 0 {
				originMinutes[index] = leg
			} else {
				matrix[order[position-1]][index] = leg
			}
			if leg < 0 {
				break
			}
			clock += leg
			stop := stops[index]
			if stop.Window != nil && clock < stop.Window.Start {
				clock = stop.Window.Start
			}
			if stop.FixedStart != nil && clock < *stop.FixedStart {
				clock = *stop.FixedStart
			}
			clock += stop.Duration
		}
		return Evaluate(stops, matrix, order, start, end, Options{OriginMinutes: originMinutes})
	}
	baseline, err := evaluate(originalOrder)
	if err != nil {
		return Proposal{}, err
	}
	candidate := baseline
	if !slices.Equal(originalOrder, p.Order) {
		candidate, err = evaluate(p.Order)
		if err != nil {
			return Proposal{}, err
		}
	}
	p.Original = baseline
	p.Proposed = candidate
	p.Savings = nil
	if len(candidate.Conflicts) > 0 || (len(baseline.Conflicts) == 0 && candidate.TravelMinutes >= baseline.TravelMinutes) {
		p.Proposed = baseline
		p.Order = originalOrder
	}
	p.Changed = !slices.Equal(originalOrder, p.Order)
	p.Explanation = "Transit departures were rechecked; no lower-travel feasible order was found."
	if len(p.Proposed.Conflicts) > 0 {
		p.Explanation = "Transit rechecking found conflicts; no feasible proposal is available."
	} else if p.Changed {
		p.Explanation = "A feasible transit order was found using estimates rechecked at scheduled departures."
	}
	if len(baseline.Conflicts) == 0 && len(p.Proposed.Conflicts) == 0 {
		savings := baseline.TravelMinutes - p.Proposed.TravelMinutes
		p.Savings = &savings
	}
	return p, nil
}
