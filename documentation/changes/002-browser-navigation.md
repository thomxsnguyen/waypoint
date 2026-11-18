# MVP verification fix: wait for destination screen before removal

## Problem

The provider-outage browser test clicks a generic Remove button immediately
after requesting navigation. Next.js can still display the outgoing discovery
screen, so the test sends a saved-place deletion instead of deleting an activity.
The API correctly returns 409 for that referenced place.

## Scope and acceptance

Wait for the itinerary heading and target the activity card's Remove button.
On the return navigation, wait for discovery before removing the saved place.
Do not change application behavior. The existing outage scenario must complete
against the real test API/database without sleeps or forced clicks.

Status: planned before implementation.
