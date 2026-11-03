# Waypoint — MVP Implementation Guide

**Document type:** Engineering implementation plan  
**Product:** Waypoint, personal travel planning application  
**Release:** MVP v1.0  
**Architecture:** Next.js frontend + modular Go API + PostgreSQL  
**Status:** Implementation blueprint (not an assertion that features have been built)

---

## 1. Purpose

This document translates the Waypoint project scope into an actionable development plan for the **first usable release**. It establishes concrete implementation boundaries, recommended defaults, database design, API contracts, screen behavior, milestones, testing requirements, and a definition of done.

The goal is a complete **vertical slice**: a signed-in user creates a trip, discovers and saves places, schedules them by day, sees them on a map, and later returns to the same persisted itinerary.

**Source alignment:** The project proposal defines the MVP as user/trip management, place discovery, itinerary editing, interactive mapping, and persistent data. Route optimization belongs to v2.0; budgeting, reservations, packing, export, and optional AI belong to v3.0.

### 1.1 User story

> As a traveler planning a vacation, I want to create a trip, save interesting locations, assign them to dates, reorder my activities, and view them on a map so that I have one reliable itinerary to revisit.

### 1.2 MVP acceptance checklist

- [ ] A user can sign in and sign out.
- [ ] A user can create, list, open, edit, and delete their own trips.
- [ ] Each trip stores a destination, start/end dates, and an IANA time zone.
- [ ] A user can search for places near the trip destination.
- [ ] A user can save and remove places from a trip.
- [ ] A user can add a saved place to a trip day as an activity.
- [ ] A user can edit activity time, duration, notes, and position.
- [ ] A user can reorder activities within a day.
- [ ] The itinerary and map indicate the same selected location.
- [ ] All changes persist after refresh, sign-out/sign-in, and a new browser session.
- [ ] Users cannot read or modify each other's trips or activities.
- [ ] The deployed MVP passes its key end-to-end flow.

### 1.3 Explicit exclusions

**Do not implement in this release:** route optimization, travel-time matrices, AI planning, trip budgets, reservations management, flight/hotel booking, payments, packing lists, email parsing, multi-user collaboration, multi-city optimization, or background job infrastructure. The MVP displays markers; it does **not** have to compute an optimal driving or walking route.

---

## 2. Implementation decisions

The original scope allows more than one valid implementation. The choices below are **recommendations for a concrete build**, not requirements already settled by the source document.

| Area | Recommended MVP decision | Reason |
| --- | --- | --- |
| Frontend | Next.js App Router, React, TypeScript | Typed UI and straightforward page routing |
| Styling | Tailwind CSS | Efficient, consistent UI work |
| Backend | Go HTTP server, `chi` router | Lightweight REST implementation |
| Database | PostgreSQL with `pgx`/`pgxpool` | Explicit, performant SQL access |
| Migrations | Versioned SQL migrations (e.g., `golang-migrate`) | Reproducible schema evolution |
| Auth | Managed auth provider; use Clerk as a starting option | Avoid building password recovery and session security from scratch |
| External places | Google Places API (New) | Place search and identifiers |
| Interactive map | Google Maps JavaScript API | Map markers and selection interactions |
| Client data fetching | Typed API wrapper with TanStack Query (optional) | Cache invalidation, loading, mutations |
| Drag and drop | `dnd-kit` (optional) | Accessible activity ordering |
| Local infrastructure | Docker Compose: frontend, Go API, PostgreSQL | Reproducible environment |
| Tests | Go unit/integration tests, browser E2E via Playwright | Validate both domain logic and full user journeys |
| Deployment | One frontend service, one Go API service, managed PostgreSQL | Minimal operations burden |

**Keep provider interfaces replaceable.** Authentication and place search should have adapters so a future provider switch does not change core trip and itinerary logic. Check current vendor SDK documentation, billing, quota, storage, and attribution requirements when integrating; vendor-specific details may change.

### 2.1 Minimal architectural flow

```text
Browser (Next.js / React)
  ├─ managed auth UI/session
  ├─ trip & itinerary views
  ├─ place search & map
  └─ authenticated JSON fetches
           │
           ▼
Go API (single modular monolith)
  ├─ auth middleware (verify identity)
  ├─ trips handlers/service/repository
  ├─ places handlers/provider adapter/repository
  └─ itinerary handlers/service/repository
           │                     │
           ▼                     ▼
     PostgreSQL           Google Places API

Browser map component ── Google Maps JS API
```

**Trust boundary:** The browser never connects directly to PostgreSQL. The API owns all authorization decisions. Provider keys are scoped and restricted according to where they run; never embed a server credential in browser code.

---

## 3. Suggested repository structure

```text
waypoint/
├── README.md
├── .gitignore
├── .env.example
├── compose.yaml
├── docs/
│   ├── PROJECT_SCOPE.md
│   ├── MVP_IMPLEMENTATION.md
│   └── api.md
├── frontend/
│   ├── app/
│   │   ├── (auth)/sign-in/
│   │   ├── (app)/trips/
│   │   │   ├── page.tsx                 # all trips
│   │   │   └── [tripId]/
│   │   │       ├── page.tsx             # dashboard
│   │   │       ├── places/page.tsx      # discovery
│   │   │       └── itinerary/page.tsx   # builder + map
│   │   └── layout.tsx
│   ├── components/
│   │   ├── trips/
│   │   ├── places/
│   │   ├── itinerary/
│   │   ├── maps/
│   │   └── ui/
│   ├── lib/
│   │   ├── api/
│   │   ├── auth/
│   │   └── dates/
│   ├── types/
│   └── tests/
├── backend/
│   ├── cmd/api/main.go
│   ├── internal/
│   │   ├── config/
│   │   ├── httpapi/                 # router, middleware, response helpers
│   │   ├── auth/
│   │   ├── trips/
│   │   ├── places/
│   │   ├── itinerary/
│   │   └── db/
│   ├── migrations/
│   ├── testdata/
│   ├── go.mod
│   └── Dockerfile
├── infra/
└── .github/workflows/ci.yml
```

The precise names are flexible; the key boundary is **handlers → services → repositories/provider adapters**. Avoid one giant `main.go`, but also avoid unnecessary internal frameworks.

---

## 4. Phase 0 — Project bootstrap

**Outcome:** The entire local stack starts reliably and the web app can reach the API.

### 4.1 Work items

1. Initialize Git repository and root documentation.
2. Scaffold Next.js with TypeScript.
3. Initialize the Go module and create `cmd/api/main.go`.
4. Start PostgreSQL through Compose with persistent local volume.
5. Add an HTTP server with graceful shutdown and sensible read/write timeouts.
6. Expose `GET /health` returning HTTP 200 and a minimal JSON response.
7. Connect the frontend to the health endpoint and render connectivity status during development.
8. Add `.env.example` (dummy values only), ignore real `.env*` secrets, and document required variables.
9. Set up format/lint/build/test CI for both codebases.
10. Add structured request logs with request ID and duration; redact tokens and secrets.

### 4.2 Environment configuration

Illustrative environment names; adjust to selected hosting and auth tooling:

```dotenv
# backend
PORT=8080
DATABASE_URL=postgres://waypoint:localpass@db:5432/waypoint?sslmode=disable
AUTH_ISSUER=<provider-issuer>
AUTH_AUDIENCE=<expected-audience>
GOOGLE_MAPS_SERVER_API_KEY=<server-key>
CORS_ALLOWED_ORIGIN=http://localhost:3000

# frontend
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
NEXT_PUBLIC_MAPS_BROWSER_API_KEY=<restricted-browser-key>
NEXT_PUBLIC_AUTH_PUBLISHABLE_KEY=<provider-public-key>
```

Never commit populated secrets. The browser Maps key is necessarily public and should be **API- and referrer-restricted**; the server key should be **API- and environment-restricted**. Authentication SDKs may require additional environment values.

### 4.3 Exit criteria

- `docker compose up --build` starts all required local services.
- The frontend shows a successful response from `GET /health`.
- A clean checkout follows documented setup without manual database edits.
- CI passes formatting, compilation, and starter tests.

---

## 5. Phase 1 — Authentication, database, and trip CRUD

**Outcome:** A user can securely create a trip and reopen it after a restart.

### 5.1 Authentication workflow

Recommended flow with a managed provider:

1. User signs in through the provider's frontend integration.
2. Frontend obtains an appropriate short-lived credential for calls to the Go API.
3. Every protected request carries credentials over HTTPS (for example, an `Authorization: Bearer ...` header).
4. Go middleware verifies the credential using the selected provider's **current server SDK or published validation procedure**: signature, issuer, audience, expiry, and relevant key rotation rules.
5. The middleware places the validated provider subject in request context.
6. The API resolves (or provisionally creates, using a safe upsert) its local `users` row.
7. All trip queries use the authenticated local user ID, **never an owner ID submitted by the client**.

Do not implement custom password storage or trust an unverified user ID/header. Define behavior for expired credentials (401 and a frontend re-auth path).

### 5.2 Core schema

This SQL is a **design sketch** to turn into numbered migrations and test before deploying. Use `UUID` IDs generated by the application or database; the example assumes `gen_random_uuid()` is available on the target PostgreSQL setup.

```sql
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  auth_provider_id TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE trips (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  destination TEXT NOT NULL,
  start_date DATE NOT NULL,
  end_date DATE NOT NULL,
  timezone TEXT NOT NULL,
  description TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT trip_date_order CHECK (end_date >= start_date)
);

CREATE INDEX idx_trips_user_id ON trips(user_id);

CREATE TABLE saved_places (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  provider TEXT NOT NULL DEFAULT 'google',
  provider_place_id TEXT NOT NULL,
  notes TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (trip_id, provider, provider_place_id),
  UNIQUE (id, trip_id)
);

CREATE TABLE itinerary_days (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  date DATE NOT NULL,
  UNIQUE (trip_id, date),
  UNIQUE (id, trip_id)
);

CREATE TABLE activities (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id UUID NOT NULL,
  day_id UUID NOT NULL,
  saved_place_id UUID NOT NULL,
  start_time TIME,
  duration_minutes INTEGER NOT NULL DEFAULT 60,
  notes TEXT,
  position INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT positive_duration CHECK (duration_minutes > 0),
  CONSTRAINT nonnegative_position CHECK (position >= 0),
  FOREIGN KEY (day_id, trip_id)
    REFERENCES itinerary_days(id, trip_id) ON DELETE CASCADE,
  FOREIGN KEY (saved_place_id, trip_id)
    REFERENCES saved_places(id, trip_id)
);

CREATE INDEX idx_saved_places_trip_id ON saved_places(trip_id);
CREATE INDEX idx_activities_day_position ON activities(day_id, position);
```

**Deliberate design details:**

- `activities.trip_id` is redundant for lookup but enables **composite foreign keys** that prevent a place from trip A from being scheduled into trip B.
- `position` is maintained by application transactions; the ordered sequence should remain contiguous (`0..n-1`). A unique `(day_id, position)` constraint can be added with a reorder strategy that handles temporary collisions (e.g., a deferrable constraint or temporary values).
- `start_time` is a **local wall-clock time** for that trip day. `created_at`/`updated_at` are absolute UTC-capable timestamps. Validate `timezone` against the IANA database in application logic.
- The proposed five core tables correspond to the scope document. Provider-specific transient details should **not** be stored beyond what provider terms permit.

### 5.3 Trip operations

| Method | Path | Request intent | Result |
| --- | --- | --- | --- |
| `GET` | `/api/v1/trips` | List current user's trips | `200` |
| `POST` | `/api/v1/trips` | Create trip | `201` |
| `GET` | `/api/v1/trips/{tripId}` | Retrieve owned trip | `200` |
| `PATCH` | `/api/v1/trips/{tripId}` | Edit name/destination/dates/time zone/description | `200` |
| `DELETE` | `/api/v1/trips/{tripId}` | Delete owned trip and dependent data | `204` |

**Create example:**

```http
POST /api/v1/trips
Content-Type: application/json
Authorization: Bearer <token>

{
  "name": "Tokyo Autumn Trip",
  "destination": "Tokyo, Japan",
  "start_date": "2026-11-12",
  "end_date": "2026-11-16",
  "timezone": "Asia/Tokyo",
  "description": "Food and sightseeing"
}
```

**Representative success response:**

```json
{
  "data": {
    "id": "<generated-trip-uuid>",
    "name": "Tokyo Autumn Trip",
    "destination": "Tokyo, Japan",
    "start_date": "2026-11-12",
    "end_date": "2026-11-16",
    "timezone": "Asia/Tokyo",
    "description": "Food and sightseeing"
  }
}
```

**Validation rules:**

- Name/destination required with reasonable length limits.
- Use `YYYY-MM-DD` calendar dates; reject invalid formats and end-before-start.
- Verify time zone is an IANA identifier (e.g., `Asia/Tokyo`).
- Enforce a documented maximum trip length if needed to prevent abuse.
- Respond with `404` for a missing or not-owned trip to avoid leaking its existence.
- Updating dates must reconcile itinerary days **in a transaction**. Recommended MVP policy: adding dates creates new days; shortening dates is rejected with `409` if it would delete days containing activities. Provide a clear message explaining the required user action. Empty out-of-range days may be deleted.

### 5.4 Day creation strategy

When a trip is created, generate one `itinerary_days` row for each calendar date in the **inclusive** date range (e.g., Nov 12–16 gives five days). Iterate by calendar dates rather than adding 24-hour timestamp durations, which can break around daylight saving transitions. The trip's IANA time zone determines interpretation of local schedule times.

### 5.5 Phase 1 tests

- Valid and invalid trip creation.
- Start/end date boundary and same-day trips.
- Trip list filtered to authenticated owner.
- Unauthorized/no-token access gets 401.
- User B accessing user A's trip receives 404.
- Updating dates creates/removes eligible empty itinerary days.
- Deletion cascades to owned dependent records.
- Schema migrations run on a fresh test DB.

**Exit criteria:** A user signs in, creates a trip, restarts the application, and retrieves the same trip without access to another user's records.

---

## 6. Phase 2 — Place discovery and saved places

**Outcome:** A user can search a destination and create a reusable trip-specific list of locations.

### 6.1 Place search flow

```text
Search text + trip context
  → frontend GET /places/search
  → Go validates query & request identity
  → Places provider adapter performs lookup
  → Go returns normalized result DTOs
  → frontend renders cards and optional markers
  → user saves a selected provider_place_id
```

Place lookup may use a destination/location bias when a supported place identifier or coordinates are available. MVP searches need not understand arbitrary travel intent.

### 6.2 Public API contracts

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/places/search?q=...&trip_id=...` | Search provider-backed places in a trip context |
| `GET` | `/api/v1/trips/{tripId}/places` | List saved places |
| `POST` | `/api/v1/trips/{tripId}/places` | Save a provider place identifier |
| `PATCH` | `/api/v1/trips/{tripId}/places/{savedPlaceId}` | Edit user-authored notes |
| `DELETE` | `/api/v1/trips/{tripId}/places/{savedPlaceId}` | Remove an unused saved place |

The GET/PATCH/DELETE routes for saved places **extend** the scope document's minimal API table to make the described user-facing actions implementable.

**Example normalized search result (illustrative):**

```json
{
  "data": [
    {
      "provider": "google",
      "provider_place_id": "<provider-place-id>",
      "name": "Example Museum",
      "address": "Example address",
      "location": { "lat": 35.0, "lng": 139.0 },
      "external_url": "https://maps.google.com/"
    }
  ]
}
```

The actual URL must be a legitimate returned/constructed link permitted by the provider; the sample is not a real place record. Only request provider fields needed for the MVP.

**Save request:**

```json
{
  "provider": "google",
  "provider_place_id": "<provider-place-id>",
  "notes": "Visit in the morning"
}
```

**Save behavior:** Server verifies ownership and resolves the place through the provider when required; never assume a browser-supplied name, address, or coordinate is authoritative. Reject duplicate saved provider IDs for a trip with a documented `409` response or return the existing saved record (pick and test one behavior; **recommended: return existing record**).

### 6.3 Provider compliance and failure behavior

- Use narrow field selection and monitor quota/cost.
- Display provider-required attribution, trademarks, and links.
- Respect rules about persisting place attributes; `provider_place_id` and user-written notes are the primary long-term records in this design.
- Fetch current display details when needed, rather than relying on indefinitely cached third-party data.
- Ensure the frontend can show a place as **temporarily unavailable** if the provider fails; do not delete its saved identifier automatically.
- Handle rate limits, invalid IDs, no results, and network errors distinctly.
- For the MVP, removing a place referenced by an activity should return `409` with guidance to remove the activities first. This avoids accidentally deleting itinerary entries.

### 6.4 Frontend requirements

- Search box with debounce and explicit loading state.
- Place cards with name, address (when available), and “Save to trip.”
- Clear feedback for saved / already-saved results.
- Saved list with notes and remove action.
- Empty and provider-error states.
- Map search markers where desired; do not confuse these with saved itinerary markers.

### 6.5 Phase 2 tests

- Mock provider response mapping into normalized DTOs.
- Empty query and very long query validation.
- No-results and transient error cases.
- Save same place twice; verify agreed duplicate behavior.
- User cannot save into/read from another user's trip.
- A referenced saved place cannot be removed without an explicit activity cleanup.
- Test provider request failures without live paid requests.

**Exit criteria:** Search for a Tokyo attraction, save it, refresh the trip, and find the saved place again.

---

## 7. Phase 3 — Itinerary builder and map

**Outcome:** Users can build an editable multi-day itinerary with synchronized location markers.

### 7.1 Activity model and rules

| Field | Description | Example |
| --- | --- | --- |
| `id` | Internal activity ID | UUID |
| `day_id` | Itinerary day | UUID |
| `saved_place_id` | Place already saved to this trip | UUID |
| `start_time` | Optional local time (`HH:MM`) | `09:30` |
| `duration_minutes` | Required positive estimate | `90` |
| `notes` | User-authored note | `Buy tickets onsite` |
| `position` | Zero-based list order | `2` |

For MVP, **activity order is authoritative**. Start times are informational user inputs, not an automatically optimized schedule. A saved place can be used in more than one activity/day (for repeat visits). Time overlaps may be flagged visually but **do not silently reorder** the itinerary.

### 7.2 API contracts

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/trips/{tripId}/days` | Days and activities for owned trip |
| `POST` | `/api/v1/days/{dayId}/activities` | Append an activity from a saved place |
| `PATCH` | `/api/v1/activities/{activityId}` | Edit start time, duration, notes |
| `DELETE` | `/api/v1/activities/{activityId}` | Remove activity |
| `PUT` | `/api/v1/days/{dayId}/order` | Replace full activity order for a day |

**Create request example:**

```json
{
  "saved_place_id": "<saved-place-uuid>",
  "start_time": "09:30",
  "duration_minutes": 90,
  "notes": "Arrive before opening crowds"
}
```

**Order request example:**

```json
{
  "activity_ids": [
    "<first-activity-uuid>",
    "<second-activity-uuid>",
    "<third-activity-uuid>"
  ]
}
```

The server must verify that the submitted array is an **exact permutation** of the day's current activity IDs: no missing, foreign, duplicated, or invented IDs. Execute all position updates in one DB transaction. To avoid lost updates from simultaneous tabs, a later improvement can use a day revision/optimistic concurrency check; for the MVP, document that last successful write wins and refetch after mutations.

### 7.3 Activity security and database integrity

- Resolve the parent trip for every day/activity request and verify ownership.
- Enforce `saved_place_id` belongs to the same trip (composite FK and service check).
- Validate `duration_minutes > 0` and apply a reasonable maximum.
- Validate optional time strictly, e.g., `HH:MM`; store local wall-clock time.
- Append new activity at the current end position inside a transaction.
- After deletion, compact positions to `0..n-1` transactionally.
- Prevent updates from writing a mismatched trip/day/place association.
- Return unambiguous JSON error codes for validation and conflicts.

### 7.4 Screen behaviors

#### Screen A — My Trips (`/trips`)

- Cards for upcoming, active, and past trips, derived from date range.
- Create-trip modal/page, update/delete actions, clear empty state.
- Open a trip dashboard.

#### Screen B — Trip Dashboard (`/trips/[tripId]`)

- Trip name, destination, date range, optional description.
- Count of saved places and scheduled activities.
- Navigation to place discovery and itinerary builder.
- Links/actions to edit details.

#### Screen C — Place Discovery (`/trips/[tripId]/places`)

- Search, result cards, saved collection, save/remove, optional markers.
- Provider attribution and empty/error/loading states.

#### Screen D — Itinerary Editor (`/trips/[tripId]/itinerary`)

- Day selector with human-readable dates.
- Ordered activity cards with name, notes, start time, duration.
- Add place from trip's saved collection; edit/delete actions.
- Drag-and-drop reorder or accessible up/down controls.
- Map panel showing markers for **the selected day** (optionally all saved places with a distinct visual state).
- Clicking card focuses/highlights map marker; clicking marker selects corresponding card.
- Mobile layout: stack the list and map instead of requiring a wide desktop split.
- Screen exposes autosave/explicit save feedback depending on the selected mutation UX.

### 7.5 Map component contract

A map needs no routing engine in v1.0. Given an array of selected-day activities with resolved coordinates:

```ts
type MapActivity = {
  activityId: string;
  name: string;
  lat: number;
  lng: number;
  position: number;
};
```

Maintain one `selectedActivityId` state shared by the itinerary list and map component. Use stable IDs instead of array indices for selection, because reordering changes indices. Fit bounds to current day markers; handle zero-marker and single-marker cases. If place details are unavailable, show the activity card with a notice and omit its marker rather than crashing.

### 7.6 Timeline and time-zone decisions

- `DATE` fields identify travel calendar days, not UTC midnight instants.
- `TIME` fields represent the local time in the trip's IANA time zone.
- Display `YYYY-MM-DD` dates in a way that does not accidentally shift the day when the user browses from another time zone.
- Do not infer travel time between activities during MVP.
- If start time + duration crosses midnight, show a warning; do not auto-move the activity into the next day.

### 7.7 Phase 3 tests

- Add activity to a trip day and persist it.
- Reject place from another trip.
- Edit time, duration, and notes.
- Reorder three activities and reload to verify order.
- Reject an order payload with duplicate/missing/foreign IDs.
- Delete middle activity and verify contiguous positions.
- Click an activity and confirm the correct map marker is selected; repeat from marker to card.
- Test empty days and unavailable coordinates.
- Test dates near DST changes and travel time zones different from browser time zone.

**Exit criteria:** The user builds a multi-day itinerary, edits and reorders activities, refreshes, and sees exactly the same itinerary and matching map markers.

---

## 8. Error responses and backend conventions

### 8.1 API response envelope

Use one convention consistently:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "End date must not precede start date.",
    "details": { "field": "end_date" }
  }
}
```

| Status | When used |
| --- | --- |
| `200` | Successful reads and updates |
| `201` | New resource created |
| `204` | Successful deletion with no body |
| `400` | Malformed input or invalid field values |
| `401` | Missing/invalid/expired authentication |
| `403` | A forbidden operation not covered by owner-resource hiding |
| `404` | Missing or not-owned resource |
| `409` | Conflict, e.g., saved place still referenced by activities |
| `429` | Rate-limited request |
| `502` / `503` | External provider failure or temporary unavailability |
| `500` | Unexpected server error; no internal details leaked |

### 8.2 Handler/service/repository boundaries

**Handler:** decode/validate shape, read authenticated subject, call service, serialize response.  
**Service:** implement ownership checks, domain invariants, transactions, and policies.  
**Repository:** parameterized SQL, database row mapping.  
**Provider adapter:** external Places requests, field mapping, errors, timeouts.

Every external HTTP call and database operation should receive a propagated Go `context.Context`. Never log raw authorization headers, private secrets, or full sensitive user input.

### 8.3 Security controls

- Enforce authentication on all trip-specific API endpoints.
- Enforce **ownership at data access**, not just in the frontend.
- Use prepared/parameterized SQL, strict JSON validation, request size limits, and timeouts.
- Set a specific production CORS allowlist.
- Use HTTPS in production and secure authentication/session settings.
- Use provider key restrictions and backend rate limiting on costly external API routes.
- Restrict outbound network destinations for server requests; no arbitrary fetch URL endpoint.
- Document deletion behavior and database backups.

---

## 9. Testing and quality gates

### 9.1 Unit tests (Go)

- Date range / time-zone validation.
- Activity duration and local time parsing.
- Exact-permutation reorder validation.
- Reorder index generation and deletion compaction.
- Trip ownership service policies.
- Provider response normalization.

### 9.2 Integration tests (Go + test PostgreSQL)

- Migrations and constraints from scratch.
- All trip/places/day/activity database operations.
- Cross-trip composite FK violations.
- Reorder and deletion transactions (including rollback on errors).
- Concurrent mutations when relevant.
- No data leakage across two test identities.

Use isolated test DBs or cleanup between tests. Do not use the development database as a test fixture.

### 9.3 Frontend tests

- Form validation and displayed server errors.
- Loading, empty, unavailable, and retry states.
- Activity list and map selection state.
- Correct query invalidation after mutations.
- Keyboard-accessible reorder behavior.

### 9.4 Browser end-to-end smoke test

```text
1. Sign in as test user A
2. Create a trip (Tokyo, Nov 12–16)
3. Open place discovery and search for an attraction
4. Save a place
5. Open Nov 13 in the itinerary
6. Add saved place as an activity
7. Set duration and start time
8. Add another place and reorder both activities
9. Select activity and verify map marker highlights
10. Refresh page and verify persisted order/details
11. Sign out, sign back in, and verify trip persists
12. Sign in as user B and confirm user A's trip is inaccessible
```

For deterministic automated tests, mock/stub external provider requests and use dedicated testing credentials or auth test setup. Run a small manual staging smoke test against the real mapping provider before release.

### 9.5 CI minimum

| Target | Required CI check |
| --- | --- |
| Go | `go test ./...`, formatting, vet/static analysis |
| TypeScript | Typecheck, lint, production build |
| Database | Run migrations against test PostgreSQL |
| E2E | Key flow on pull requests or before deployment |
| Secrets | No real credentials committed |

---

## 10. Delivery sequence and issue breakdown

Build in the following order. Each issue should be small enough to have a testable output. Do **not** start an entire new subsystem before completing the preceding end-to-end slice.

| ID | Phase | Issue | Depends on | Proof of completion |
| --- | --- | --- | --- | --- |
| M0-01 | Setup | Scaffold mono-repo / Next.js / Go | — | Both apps compile |
| M0-02 | Setup | Docker Compose PostgreSQL + API + web | M0-01 | Fresh local startup succeeds |
| M0-03 | Setup | Health endpoint and web connectivity | M0-02 | Browser renders 200 response |
| M0-04 | Setup | CI, env examples, README | M0-01 | CI runs on a pull request |
| M1-01 | Trips | SQL migrations for 5 tables | M0-02 | Migrations + rollback tested |
| M1-02 | Trips | Managed auth integration / Go verification | M0-03 | Protected endpoint rejects anonymous requests |
| M1-03 | Trips | Trip repository + service validations | M1-01, M1-02 | Unit/integration tests pass |
| M1-04 | Trips | Trip REST CRUD endpoints | M1-03 | API tests pass |
| M1-05 | Trips | My Trips and trip form | M1-04 | Signed-in create/list/edit/delete works |
| M1-06 | Trips | Dashboard + generated days | M1-05 | Trip shows correct day range |
| M2-01 | Places | Google Places provider adapter | M0-03 | Mock provider contract test passes |
| M2-02 | Places | Search API with quota/error handling | M2-01, M1-02 | Search endpoint returns normalized data |
| M2-03 | Places | Saved places API with ownership checks | M1-01, M2-02 | Save/list/remove tests pass |
| M2-04 | Places | Search and saved-places screen | M2-02, M2-03 | Search, save, refresh works |
| M3-01 | Itinerary | Days/activities backend | M1-06, M2-03 | Create/update/delete activity works |
| M3-02 | Itinerary | Transactional reorder endpoint | M3-01 | Exact order survives reload |
| M3-03 | Itinerary | Day selector and activity editor | M3-01 | Edit all activity fields in UI |
| M3-04 | Itinerary | Reorder UI + error rollback | M3-02, M3-03 | Drag/drop or controls persist order |
| M3-05 | Map | Map renderer and marker selection | M2-04, M3-03 | Cards and markers sync both directions |
| M3-06 | QA | Cross-user tests, E2E, staging smoke | M3-04, M3-05 | Key journey green |
| M3-07 | Release | Deploy, monitor, document | M3-06 | Public app + setup docs available |

### 10.1 Suggested sprint grouping (sequence, not fixed dates)

**Sprint A — Foundation:** M0-01 through M0-04.  
**Sprint B — First persistent trip:** M1-01 through M1-06.  
**Sprint C — Search and collect:** M2-01 through M2-04.  
**Sprint D — Build an itinerary:** M3-01 through M3-04.  
**Sprint E — Map, polish, deploy:** M3-05 through M3-07.

Timeboxes can be adjusted to your availability; do not treat these labels as a promised schedule.

---

## 11. Deployment checklist

- [ ] Production database exists and credentials have least privilege.
- [ ] Migrations run via documented release process.
- [ ] API and frontend use HTTPS.
- [ ] Correct frontend origin is allowed by the API CORS policy.
- [ ] Auth provider's production issuer/audience/origins are configured.
- [ ] Browser and server Maps/Places credentials are appropriately restricted.
- [ ] External API usage quotas/alerts are configured.
- [ ] Request and application logs are available without exposing private data.
- [ ] Database backups and recovery procedure are documented.
- [ ] Secrets are supplied by deployment configuration, not committed files.
- [ ] `/health` succeeds and automated smoke flow passes.
- [ ] `README.md` contains setup, environment, commands, architecture, and test instructions.

### 11.1 Observability starter metrics

Log and/or record:

- HTTP request count, status, and latency.
- Failed authentication and authorization attempts (without logging tokens).
- Place provider failures and quota/rate-limit responses.
- Database error counts and slow queries.
- Activity reorder failures and transaction rollbacks.

No advanced telemetry platform or Redis instance is required to finish v1.0.

---

## 12. MVP definition of done

The MVP may be labeled **complete** only when all of the following are true:

1. A new developer can follow the README, run the stack, and run the test suite.
2. Users can register/sign in (through the managed provider), manage their own trips, and sign out.
3. Users can search, save, list, and remove locations (subject to activity references).
4. Users can assign saved places to specific itinerary days, edit fields, and reorder activities.
5. Itinerary and map selections are synchronized in both directions.
6. All user changes remain after restart or reload.
7. Cross-user authorization tests pass and secret-handling basics are in place.
8. Automated unit, integration, and E2E checks pass.
9. A deployed environment supports the same end-to-end path as local development.
10. Documentation records implemented features, known limitations, and next steps.

**Do not conflate “a frontend demo that looks correct” with an MVP.** The application needs a real authenticated API, transactional persistence, useful error handling, and a repeatable test/deploy process.

---

## 13. Known design questions to decide before implementing each part

These are **decisions to record**, not blockers to building the initial foundation:

| Decision | Suggested initial policy |
| --- | --- |
| Authentication provider | Use one managed provider and its official supported server validation path |
| Trip destination location | Begin with user-visible destination text; add canonical place ID/coordinates when enabling reliable search bias |
| Place data storage | Store provider place ID and user notes; resolve display details as permitted by provider terms |
| Duplicate saved places | Return existing saved row; don't create duplicates |
| Reusing places in itinerary | Allow multiple activity rows referencing one saved place |
| Place deletion when referenced | Return 409; require activity cleanup first |
| Resizing trip dates | Expand automatically; reject shortening that would remove nonempty days |
| Conflicting activity times | Warn rather than automatically solve overlaps in v1 |
| Concurrent editing | Last successful write wins for MVP; refetch UI state; consider revisions later |
| Travel modes and routing | Postpone until v2 optimization |
| Offline mode | Not part of MVP |

These policies can evolve, but the backend, UI, tests, and documentation must agree on each one.

---

## 14. What comes immediately after MVP

The v2.0 route optimizer can build directly on the completed data model. It will resolve coordinates and routing estimates for scheduled places, construct weighted travel-time matrices, generate a nearest-neighbor proposal, improve candidates with local search, validate fixed-time and duration constraints, and let users compare the proposed itinerary before applying it.

**Do not create routing endpoints, background workers, travel-time caches, or optimization tables now.** Keep a small `routing` module placeholder only if helpful for future architecture, and prioritize shipping the complete MVP user journey first.

---

## Appendix A — Useful developer commands (illustrative)

```bash
# Start local dependencies/apps when Compose services are configured
docker compose up --build

# Run backend tests
cd backend && go test ./...

# Run frontend checks
cd frontend && npm run lint && npm run build
```

These commands assume the corresponding scripts and Compose services have been configured. Migration and E2E commands should be documented in `README.md` once tooling is selected.

## Appendix B — Source and deviations

**Primary basis:** `Waypoint — Software Engineering Project Proposal` supplied with the project. Its intended MVP phases are setup (Phase 0), database and trip management (Phase 1), place discovery (Phase 2), and itinerary builder (Phase 3).

**Implementation-specific proposals added here:** recommended libraries, authentication provider candidate, SQL sketch and composite foreign keys, endpoint extensions, response envelopes, deletion/reorder policies, issue IDs, and test cases. These are practical implementation recommendations, not features claimed to exist in the original proposal.
