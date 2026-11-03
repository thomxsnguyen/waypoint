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
