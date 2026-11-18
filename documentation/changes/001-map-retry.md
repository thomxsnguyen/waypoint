# MVP fix: recover map loading after authentication failure

## Problem

The existing Retry map action can reuse a rejected loader promise or a Google
Maps namespace left behind by an authentication failure. A retry must load the
script again rather than treating that namespace as proof of success.

## Scope

Track loader authentication failure, clear failed loader state, and retry the
existing script integration. Preserve itinerary editing during map failure.
No new provider, map features, or itinerary behavior.

## Acceptance

- A simulated Google authentication failure displays an error and Retry map.
- Retrying fetches the script again and renders the selected day's markers.
- Activity data remains visible throughout; marker/card selection still works.
- Existing journey and provider-outage tests continue to pass.

## Validation plan

Add a browser regression with Google Maps responses stubbed, using the existing
real API and isolated PostgreSQL test harness. Run typecheck, lint, Go tests,
and the browser suite. Record actual results in the progress log.

Status: implemented. Authentication failure invalidates the cached loader and
removes its script. Retry bypasses the namespace left by the failed script.

The regression was run against the original loader and failed at the expected
assertion: zero markers after retry instead of two. With the fix, both browser
scenarios passed. Typecheck and lint also passed. The journey confirms activities
can be added during map failure and card/marker selection works after recovery.
