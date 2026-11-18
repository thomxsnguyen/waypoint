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
