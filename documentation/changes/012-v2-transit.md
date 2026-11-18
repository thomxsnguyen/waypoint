# v2 transit: time-specific estimates and revalidation

Add TRANSIT with a required departure timestamp derived from the selected trip
day/start and IANA zone. Google supports at most 100 transit matrix elements and
requests within seven days past / 100 days future. Limit transit to ten locations
(including optional origin); reject out-of-range dates before provider calls.
DST-transition days are explicitly unsupported in this initial wall-clock model,
so ambiguous/skipped clock times are never silently resolved.

Initial matrix is a departure-time sample used only to propose an order. Recheck
original and proposed legs at their actual departures, advancing through travel,
activity durations, waiting windows, and fixed starts. Each leg uses a two-point
matrix (four elements). Reserve n*n + 8*activities elements up front for worst-case
two-route rechecking. Rechecks run under the existing request deadline, with no
retries. Unavailable/late legs prevent approval. If the proposed route becomes
infeasible or no longer improves a feasible original, retain the rechecked original.
Explain that this heuristic may miss another feasible order; never silently drop.

Provider references checked 2026-11-17:
https://developers.google.com/maps/documentation/routes/transit-route
https://developers.google.com/maps/documentation/routes/reference/rest/v2/TopLevel/computeRouteMatrix

Test transit payload/time limits, per-leg departures after waiting, route costs,
recheck fallback, no feasible approval, DST rejection, API budget, and browser
mode selection. No transit booking, fare estimation, or route geometry.
