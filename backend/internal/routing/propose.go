package routing

import (
	"context"
	"fmt"
	"slices"
)

// Proposal contains complete schedules. Savings is only meaningful when both
// schedules are feasible. No persistence changes are performed by Propose.
type Proposal struct {
	Original    Evaluation
	Proposed    Evaluation
	Order       []int
	Changed     bool
	Savings     *int
	Explanation string
}

// Propose uses static directed travel estimates and a bounded heuristic; it does
// not prove global optimality or infeasibility. Stops begin in their saved order.
func Propose(ctx context.Context, stops []Stop, matrix [][]int, start, end int, options ...Options) (Proposal, error) {
	if err := ctx.Err(); err != nil {
		return Proposal{}, err
	}
	n := len(stops)
	if n > 20 {
		return Proposal{}, fmt.Errorf("optimization supports at most 20 activities")
	}
	originalOrder := make([]int, n)
	for i := range originalOrder {
		originalOrder[i] = i
	}
	baseline, err := Evaluate(stops, matrix, originalOrder, start, end, options...)
	if err != nil {
		return Proposal{}, err
	}
	origin := make([]int, n)
	if len(options) == 1 && options[0].OriginMinutes != nil {
		origin = options[0].OriginMinutes
	}
	result := Proposal{Original: baseline, Proposed: baseline, Order: slices.Clone(originalOrder)}
	best := baseline
	bestOrder := slices.Clone(originalOrder)
	feasible := len(best.Conflicts) == 0
	consider := func(order []int, evaluation Evaluation) {
		if len(evaluation.Conflicts) == 0 && (!feasible || evaluation.TravelMinutes < best.TravelMinutes) {
			best, bestOrder, feasible = evaluation, slices.Clone(order), true
		}
	}
	// Keep the baseline eligible for local improvement as well as greedy starts.
	seeds := [][]int{originalOrder}
	for first := range stops {
		if err := ctx.Err(); err != nil {
			return Proposal{}, err
		}
		order := []int{}
		used := make([]bool, n)
		clock := start
		for len(order) < n {
			next, nextEnd, distance := -1, 0, 1441
			for candidate, stop := range stops {
				if used[candidate] || (len(order) == 0 && candidate != first) {
					continue
				}
				leg := origin[candidate]
				if len(order) > 0 {
					leg = matrix[order[len(order)-1]][candidate]
				}
				if leg < 0 {
					continue
				}
				arrival := clock + leg
				if stop.Window != nil && arrival < stop.Window.Start {
					arrival = stop.Window.Start
				}
				if stop.FixedStart != nil {
					if arrival > *stop.FixedStart {
						continue
					}
					arrival = *stop.FixedStart
				}
				finish := arrival + stop.Duration
				if finish > end || (stop.Window != nil && finish > stop.Window.End) {
					continue
				}
				if leg < distance {
					next, nextEnd, distance = candidate, finish, leg
				}
			}
			if next < 0 {
				break
			}
			used[next] = true
			order = append(order, next)
			clock = nextEnd
		}
		if len(order) == n {
			seeds = append(seeds, order)
		}
	}
	for _, seed := range seeds {
		order := slices.Clone(seed)
		evaluated, _ := Evaluate(stops, matrix, order, start, end, options...)
		consider(order, evaluated)
		// Only feasible schedules are locally improved; greedy seeds can repair an
		// infeasible baseline without presenting partially scheduled proposals.
		if len(evaluated.Conflicts) != 0 {
			continue
		}
		for pass := 0; pass < 5; pass++ {
			improved := false
			for i := 0; i < n; i++ {
				for j := i + 1; j < n; j++ {
					if err := ctx.Err(); err != nil {
						return Proposal{}, err
					}
					candidate := slices.Clone(order)
					slices.Reverse(candidate[i : j+1])
					trial, _ := Evaluate(stops, matrix, candidate, start, end, options...)
					if len(trial.Conflicts) == 0 && trial.TravelMinutes < evaluated.TravelMinutes {
						order, evaluated, improved = candidate, trial, true
					}
				}
			}
			consider(order, evaluated)
			if !improved {
				break
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return Proposal{}, err
	}
	result.Proposed, result.Order = best, bestOrder
	result.Changed = !slices.Equal(originalOrder, bestOrder)
	switch {
	case !feasible:
		result.Explanation = "No feasible proposal found by this heuristic; review the original schedule conflicts."
	case len(baseline.Conflicts) > 0:
		result.Explanation = "A feasible order was found; the original schedule has conflicts."
	case result.Changed:
		result.Explanation = "A feasible order with lower estimated travel time was found."
	default:
		result.Explanation = "No lower-travel feasible order was found; the original order is retained."
	}
	if feasible && len(baseline.Conflicts) == 0 {
		savings := baseline.TravelMinutes - best.TravelMinutes
		result.Savings = &savings
	}
	return result, nil
}
