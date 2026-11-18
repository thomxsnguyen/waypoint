# Waypoint MVP API

Base path: `/api/v1`. Protected requests require `Authorization: Bearer <Clerk session token>`. Bodies use `Content-Type: application/json`; unknown fields are rejected. Ownership always comes from the verified subject. A missing or not-owned resource returns `404`.

Success: `{"data": ...}`. Deletion: `204`, no body. Errors: `{"error":{"code":"VALIDATION_ERROR","message":"..."}}`.

| Method | Path | Request / response |
| --- | --- | --- |
| GET | `/health` (outside base path) | Database readiness; `data.status: "ok"` |
| GET | `/trips?offset=0` | Up to 50 owned trips, ordered by date then ID |
| POST | `/trips` | Trip fields below; returns trip, `201` |
| GET | `/trips/{id}` | Trip |
| PATCH | `/trips/{id}` | Any subset of trip fields; returns trip |
| DELETE | `/trips/{id}` | Delete trip and all dependent planning data |
| GET | `/places/search?trip_id=...&q=...` | 2–200 character query, destination added by API; up to 20 normalized results |
| GET | `/trips/{id}/places` | Saved places with freshly resolved `details`, or `unavailable: true` |
| GET | `/trips/{id}/places?details=false` | Saved records without provider calls (counts/metadata) |
| POST | `/trips/{id}/places` | `provider_place_id`, optional `notes`; duplicate returns existing record, `201` |
| PATCH | `/trips/{id}/places/{placeId}` | `notes`; returns updated notes |
| DELETE | `/trips/{id}/places/{placeId}` | Delete unused saved place; `409` if referenced |
| GET | `/trips/{id}/days` | Ordered days with ordered activity arrays |
| POST | `/days/{id}/activities` | Activity fields below; appends and returns activity, `201` |
| PATCH | `/activities/{id}` | Subset of `start_time`, `duration_minutes`, `notes`; returns activity |
| DELETE | `/activities/{id}` | Removes activity and compacts remaining positions |
| PUT | `/days/{id}/order` | `activity_ids`: exact permutation of that day's IDs; returns `saved: true` |

Trip example:

```json
{
  "name": "Tokyo autumn",
  "destination": "Tokyo, Japan",
  "start_date": "2026-11-12",
  "end_date": "2026-11-16",
  "timezone": "Asia/Tokyo",
  "description": "Food and museums"
}
```

Activity example:

```json
{
  "saved_place_id": "<UUID>",
  "start_time": "09:30",
  "duration_minutes": 90,
  "notes": "Buy tickets onsite"
}
```

`start_time` is optional or `null`, and `notes` defaults to empty. An activity's day/place association cannot be patched. Remove and add it to change days. The same saved place may be scheduled multiple times.

A saved record returns `id`, `trip_id`, `provider_place_id`, `notes`, `details`, and `unavailable`. Resolved place details include `provider`, `provider_place_id`, `name`, `address`, nullable `location: {lat,lng}`, `external_url`, and third-party `attributions: [{name,url}]`. Details are transient. Search and details have Google attribution in the UI.

A day returns `id`, `date`, `activities`. Each activity returns `id`, `day_id`, `saved_place_id`, nullable `start_time`, `duration_minutes`, `notes`, and zero-based `position`.

Responses use `Cache-Control: no-store`. `X-Request-ID` correlates structured logs. `429` includes `Retry-After: 60`. Common errors: `400 VALIDATION_ERROR`, `401 UNAUTHENTICATED`, `403 FORBIDDEN_ORIGIN`, `404 NOT_FOUND`, `409 CONFLICT`, `429 RATE_LIMITED/PROVIDER_RATE_LIMIT`, `502 PROVIDER_ERROR/PLACE_UNAVAILABLE`, `503 PLACES_UNAVAILABLE/DATABASE_UNAVAILABLE`, and `500 INTERNAL_ERROR`.

Concurrent writes are serialized per trip. The last successful edit wins; clients refetch after mutation. A failed reorder/date change rolls back the entire transaction. Authentication uses the Clerk SDK with signature, expiry/not-before, exact issuer, authorized browser origin, session identity, and optional audience checks.

## v2 optimization preview (backend milestone)

`POST /api/v1/trips/{trip}/days/{day}/optimization-preview` requires the trip
owner's session. Example body:

```json
{"travel_mode":"WALK","start_minute":540,"end_minute":1080,"fixed_activity_ids":[]}
```

Modes: WALK or DRIVE (traffic-unaware). Bounds are integer local wall-clock
minutes in the trip's IANA time zone, between 0 and 1440, with end after start.
Up to 20 activities. Fixed IDs must uniquely identify this day's activities with
saved start times. Other saved times are informational and can shift in preview.

The standard `data` envelope includes day_id, timezone, travel_mode, original,
proposed, changed, savings_minutes, explanation, and assumptions. Each schedule
contains visits (`activity_id`, `start_minute`, `end_minute`), conflicts
(`activity_id`, `reason`), and travel_minutes. Unknown arrival/travel estimates
are -1. Savings is null when the original schedule is infeasible.

No saved order, times, or notes change during preview. Feasible previews include
an approval_id and expires_at; infeasible previews return null for both.
Approval IDs expire after ten minutes and become invalid on API restart.

Errors include 400 for invalid input/over 20 activities, 404 for a foreign trip
or nonmember day, 429 for route quotas, and 503 for unconfigured/timed-out routing.
Provider response errors use 502. Budgets are 800 elements/user/minute and 4000
per API instance/minute, charged before requests (including failures); n stops
cost n*n elements for n>=2. These are separate from existing API/Places limits.


## v2 optimization approval

`POST /api/v1/trips/{trip}/days/{day}/optimization-apply` accepts:

```json
{"approval_id":"the-server-issued-preview-id"}
```

Only the owning account can approve, for the exact trip/day. No order or times
may be supplied by the client. The server applies the preview's complete order
and start times transactionally and returns `data: {"saved": true}`. Notes,
durations, and saved-place references are preserved. No provider calls occur.

Expired, unknown, replayed, or wrong-scope IDs return 409. A changed source
snapshot also returns 409; obtain and review a new preview. Foreign trips return
404. Concurrent applies of the same source serialize so only one succeeds.

Approvals are held in API memory (1000 maximum, ten-minute lifetime). Restart
invalidates them; this deployment model supports a single API instance. Capacity
exhaustion returns 503. The snapshot conservatively includes trip metadata,
selected-day activities, and saved places. Activity/trip timestamps detect
edit-and-revert; saved-place notes are compared by current value. No database
migration is needed. The itinerary comparison dialog uses this preview/approval flow.

### Optional visiting windows

Preview accepts `visiting_windows`, an array of
`{"activity_id":"…","start_minute":600,"end_minute":630}`. Each activity may
appear once and must belong to the selected day. Both integer bounds are required,
0 <= start < end <= 1440. The visit may wait until start and must finish by end;
finishing exactly at closing is allowed. Fixed starts and day bounds also apply.
Malformed/duplicate/foreign windows return 400 before any Routes call. A valid
window too short for the activity yields a conflict and no approval ID.

The response echoes supplied windows. These are user-entered constraints for the
current preview, not verified venue opening hours. They are not persisted as
reusable constraints: approval saves only the reviewed activity order/start times.
Changing a window in the UI discards its previous preview and approval.
