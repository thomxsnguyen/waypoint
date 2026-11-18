# v2 milestone 3: Google Routes matrix adapter

## Contract and scope

Add a server-side provider interface accepting ordered Google place IDs and WALK
or DRIVE. Use one Compute Route Matrix request for 2–20 stops (at most 400
billable elements); preserve activity indices even when place IDs repeat. Zero
or one stop needs no provider request. DRIVE explicitly uses TRAFFIC_UNAWARE so
these are static estimates. Transit requires departure-time semantics and remains
pending; reject unsupported modes rather than substituting driving.

Reference checked 2026-11-17:
https://developers.google.com/maps/documentation/routes/reference/rest/v2/TopLevel/computeRouteMatrix

Use a narrow field mask including status, condition, indices, and duration.
Decode unordered elements into a directed matrix. Round seconds upward to whole
minutes. Missing/duplicate/out-of-range elements and malformed durations fail the
whole response; explicit missing routes or element errors become unavailable
legs (-1). Diagonal is zero. Durations longer than a day are unavailable for this
single-day planner. No fabricated straight-line travel estimates.

Use a 10-second deadline, bounded response size, cancellation, no automatic
retries, and safe provider errors. Credentials remain server-side. Redirects must
not forward the key. No HTTP endpoint, cache, database table, or UI in this commit.
Before exposing this adapter, the API must enforce per-user/global element quotas
and ownership; a per-request cap alone is not a production spending control.

## Validation

Mock transport tests assert endpoint, payload, mode, mask, deadline, indexed
mapping, fractional rounding, unavailable routes, malformed responses, quota
errors, missing credentials, validation, and cancellation. Exercise the adapter
with Propose on a controlled matrix. No paid requests or live-key claims.

Status: planned before implementation.
