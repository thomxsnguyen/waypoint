# v2 milestone 6: itinerary comparison and approval UI

Add a day-scoped Optimize day dialog within the existing itinerary screen.
Collect walking/driving, day start/end, and explicit fixed-start selections.
Existing activity times are not fixed by default. Disable optimization for empty
or >20-activity days and explain the limit. Use existing modal/form styling.

Preview displays original-order versus proposed schedules, named activities,
estimated start/end and travel totals, savings where meaningful, conflicts, and
provider assumptions. Label original times as estimates (not saved times).
Changing inputs clears the preview. Cancel closes without saving; Apply sends
only the opaque approval ID. Explain that both order and times will be saved.
On success refetch the itinerary and clear selection. On failure disable the old
approval and require a fresh preview; server remains authoritative for expiry
and stale-plan checks. Block close/input changes while a request is in flight.

Test with the real API/PostgreSQL and test-only Routes provider: preview/cancel
preserves data, approval updates order/times and survives reload, stale approval
shows conflict without overwriting an edit. Verify mobile overflow, keyboard
labels, and capture the comparison for visual review. Run frontend checks,
production build, Go race tests, and full browser suite. No new provider modes,
origin/window model, or unrelated design changes.

Status: planned before implementation.

## Verification result

Implemented the day-scoped dialog and itinerary entry point using existing modal
and form conventions. Original/proposed times are explicitly labeled estimates;
Apply states that order and start times will be saved. Editing inputs invalidates
the displayed preview. Stale apply errors require regeneration.

Typecheck, lint, production build, full Go race suite with PostgreSQL, vet, and
compilation of the test-only API passed. All three browser scenarios passed with
the real API/database and external-provider doubles. The optimization scenario
also passed after adding input-change invalidation and lower-dialog screenshots.
It verifies unchanged data through preview/cancel, stale approval rejection,
approved order/times after reload, preservation of another-tab notes, and mobile
horizontal overflow. Mobile form and lower comparison/approval screenshots were
visually reviewed. No paid provider calls occurred.

Remaining v2 scope: origin inputs, visiting windows, transit semantics, controlled
benchmarking, and live-provider staging verification. This is not full v2 release
completion.
