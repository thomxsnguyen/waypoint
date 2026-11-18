# v2 milestone 7: user-specified visiting windows

Before implementation: allow optional per-activity earliest-start/latest-finish
windows in preview requests. Minutes use the trip's local day (0–1440), with
start < end. Both bounds are required. One window per selected-day activity.
Wait until opening; the entire activity must finish by closing (equality allowed).
Fixed starts remain hard constraints: a fixed start before opening or a finish
after closing is infeasible, not silently shifted into an approved schedule.

Add checks to evaluation, greedy construction, and candidate reversal evaluation.
Malformed/foreign/duplicate windows fail before provider calls. Impossible but
well-formed windows return conflicts and no approval, retaining every activity.

UI adds optional activity windows with clear labels, inside a collapsible section.
Changing fields clears preview. Inputs apply to this preview only and are not
stored as reusable opening hours; Apply still stores reviewed order/times only.
No automatic Google opening-hours lookup, schema migration, multi-day scheduling,
or new provider mode. Explain this distinction in UI and API docs.

Validate opening wait, exact closing, overrun, fixed-time/window conflict,
constraint-driven reorder, invalid inputs/no provider call, API approval behavior,
and browser preview conflict then successful constrained apply. Run appropriate
Go/PostgreSQL/frontend/browser checks; document results and commit in stages.

## Backend validation

Evaluator and heuristic now enforce waiting and whole-visit closing boundaries.
Preview validates optional visiting_windows before provider calls and returns the
supplied limits with the comparison. Infeasible windows yield conflicts and no
approval ID. Go race suite with PostgreSQL and vet passed, covering opening wait,
exact closing, overrun, fixed/opening conflict, constraint-driven reorder, invalid
bounds/foreign IDs, and suppression of approval for impossible windows.
