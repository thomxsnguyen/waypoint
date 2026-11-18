# v2 milestone 2: deterministic single-day proposals

## Scope and sequencing

Implement the proposal heuristic before the provider adapter so it can be
verified with controlled directed matrices. This reorders the planned work,
without expanding v2 scope. No endpoint, persistence mutation, or UI yet.

## Behavior

The current input order is the baseline. Try a nearest-feasible-neighbor route
from each possible first stop, then improve complete feasible candidates with
bounded segment reversals. Re-evaluate the entire directed route after every
reversal; symmetric 2-opt cost shortcuts are invalid here. Fixed starts are hard
constraints. Waiting is permitted. First-stop travel is not included because
there is no origin input in this milestone.

Only return a changed route if it is fully feasible and either improves travel
time over a feasible baseline or repairs an infeasible baseline. Never drop a
stop. Equal-cost candidates retain the baseline or first deterministic candidate.
If the heuristic finds no feasible route, return the original with its conflicts;
this is not proof that no feasible order exists. Savings are absent when the
baseline is infeasible. Proposals are estimates, not globally optimal solutions.

Bound computation to 20 stops, at most five improvement passes, and honor context
cancellation. Reject larger inputs explicitly rather than truncating activities.
The evaluator's independent 100-stop validation limit remains unchanged.

## Acceptance and validation

Fixtures cover reduced travel, asymmetric routes, fixed-time feasibility,
unavailable routes, impossible days, unchanged equal-cost orders, cancellation,
size limits, and preservation of inputs/activity IDs. Run race tests and vet.
Record results and limitations in PROGRESS.md before the implementation commit.

Status: implemented and verified.

Results: Go race tests and vet passed; routing package statement coverage is
96.9%. The directed fixture improves from 100 to 10 travel minutes. Tests verify
fixed-start repair, unavailable-leg avoidance, infeasible/equal-cost fallback,
input preservation, deterministic results, invalid inputs, and cancellation.
Database integration tests skip without TEST_DATABASE_URL; this change adds no
HTTP or persistence behavior. No live provider calls or browser changes occurred.

Limitations: static matrix estimates; no origin/return leg, opening windows, or
traffic-dependent re-estimation yet. Greedy construction may miss feasible
solutions, and bounded reversal search may miss lower-cost solutions. The result
explains search failure without declaring mathematical infeasibility.
