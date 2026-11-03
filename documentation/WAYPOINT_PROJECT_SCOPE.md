# Waypoint — Project Scope & Engineering Specification

**Project:** Waypoint — Smart Personal Trip Planner  
**Type:** Full-stack web application  
**Architecture:** Modular monolith  
**Core stack:** Go · Next.js / React / TypeScript · PostgreSQL · Docker · Google Maps Platform  
**Delivery strategy:** MVP (v1.0) → Smart Planning (v2.0) → Travel Management (v3.0)

## 1. Project overview

Waypoint is a personal travel planning application that gives an individual traveler one place to organize destinations, discover and save locations, build day-by-day itineraries, and see activities on a map. A later release will use travel-time data and scheduling constraints to recommend a more efficient order of visits.

### Problem

Trip planning is fragmented across maps, saved lists, spreadsheets, booking confirmations, and calendars. Manually arranging activities is time-consuming, can produce inefficient routes or time conflicts, and is difficult to maintain when plans change.

### Product objective

Create a persistent, easy-to-edit travel planning workflow, starting with manual itinerary creation and expanding to *user-approved* itinerary optimization. Initial development targets a solo-developer-sized scope rather than a full travel booking platform.

### Target users

- Individuals organizing personal vacations
- Travelers planning multi-day or multi-city trips (with advanced multi-city optimization deferred)
- People who prefer a visual itinerary over spreadsheets
- Travelers who eventually want a central place for reservation references

## 2. Release scope

### v1.0 — Minimum Viable Product (committed scope)

| Area | Required behavior | Acceptance condition |
| --- | --- | --- |
| Accounts | Register, sign in, access personal trip data | Users can access only their own trips |
| Trip management | Create, list, open, update, delete trips | Trip name, destination, date range, time zone and optional description persist |
| Place discovery | Search destination attractions, restaurants, landmarks, and accommodations | Search results show permitted place details and can be saved to a trip |
| Saved places | Save and remove places from a trip-specific collection | Saved places persist across sessions |
| Itinerary editor | Assign saved places to days; set optional start time, duration, notes and order; edit/delete/reorder | Multi-day itinerary remains intact after reload |
| Map | Display trip place markers and synchronize selected markers with itinerary activities | Selecting either the map marker or activity identifies its counterpart |
| Persistence | Store trips, saved places, itinerary days and activities in PostgreSQL | User can sign out, return and reopen the plan |

**Definition of MVP complete:** A signed-in user can create a trip, find and save places, arrange them into a multi-day schedule, inspect them on a map, and reliably reopen the same itinerary later.

### v2.0 — Smart Planning (next major milestone)

- Obtain actual provider-supported travel-time estimates between candidate places.
- Build a weighted, directed travel-time representation for route evaluation.
- Suggest a more efficient visit order for a **single day** first.
- Account for travel mode (walking, driving, or transit when supported), activity duration, visiting windows and fixed reservations.
- Detect conflicts, unreachable places, and schedules that do not fit day boundaries.
- Compare original and proposed schedules, including estimated travel time and savings.
- Explain assumptions and infeasible activities; **never silently drop activities or overwrite the original plan**.
- Apply proposed changes only after explicit user approval.

**Definition of v2.0 complete:** On a controlled example, Waypoint proposes and explains a feasible schedule backed by route-time estimates and preserves the user's original itinerary until approval.

### v3.0 — Travel Management (future / optional)

- Personal trip budget and spending categories
- Organizer for flight, hotel and activity reservations (not in-app booking)
- Packing lists
- Exportable itinerary documents
- Optional AI-powered travel suggestions
- Broader multiple-destination support

These are later enhancements, not MVP dependencies; prioritize them according to feedback and actual usage.

### Explicitly outside v1.0

Group voting, real-time multiplayer editing, in-app flight/hotel booking, payment processing, automated parsing of booking emails, AI itinerary generation, prediction of exact travel conditions, and complex global multi-day/multi-city optimization.

## 3. System architecture

```text
Browser (Next.js / React / TypeScript)
                |
          HTTPS / JSON REST
                |
     Go API (modular monolith)
       |         |          |
    Auth/Trips  Places    Itinerary
       |         |          |
       +---------+----------+
                 |
             PostgreSQL
                 |
       [future] Routing module
                 |
         External map/routing APIs
```

*The diagram is conceptual: external places and routing integrations are accessed through server-side adapters where appropriate, while browser map rendering uses the mapping provider's client library.*

### Frontend

Next.js, React and TypeScript render the trip list, dashboard, discovery experience, itinerary editor, and interactive map. The frontend validates user input, handles loading/error states, and calls the Go API. It must not connect directly to PostgreSQL or reveal private server credentials.

### Backend

One Go HTTP application (standard `net/http` or a lightweight router) handles authentication/authorization, input validation, trip and itinerary business logic, database access, provider integration, and consistent JSON errors. Suggested modules:

```text
auth/         user identity and authorization
trips/        trip CRUD and ownership
places/       external discovery and saved place references
itinerary/    days, activities, reordering
routing/      travel-time evaluation and optimization (v2)
jobs/         durable asynchronous operations (only if needed)
```

### Database

PostgreSQL provides relational integrity, transactions and persistent storage. Use schema migrations. Start with a single Go API instance and do not add microservices, Redis, WebSockets or worker infrastructure without a demonstrated need.

### External integrations

- Google Places API — place discovery
- Google Maps JavaScript API — interactive map
- Google Routes API — route durations and travel mode estimates in v2
- Managed authentication provider, or a properly secured server-side auth implementation

Provider-dependent fields, route coverage, retention, attribution and costs must be reviewed when integrating the service.

## 4. Initial data model

| Table | Core fields | Purpose |
| --- | --- | --- |
| `users` | `id`, `auth_provider_id`, `created_at` | User identity reference |
| `trips` | `id`, `user_id`, `name`, `destination`, `start_date`, `end_date`, `timezone` | Trip ownership and metadata |
| `saved_places` | `id`, `trip_id`, `provider_place_id`, `notes`, `created_at` | Places a user saved for a trip |
| `itinerary_days` | `id`, `trip_id`, `date` | Individual dates within a trip |
| `activities` | `id`, `day_id`, `saved_place_id`, `start_time`, `duration_minutes`, `position` | Ordered activities on a day |

### Invariants

- A user can own many trips; each trip has many saved places and itinerary days.
- Each day has multiple ordered activities; an activity references a saved place.
- Every protected operation verifies trip ownership, including operations through day/activity IDs.
- `end_date >= start_date`; activities belong to dates within their trip; durations are positive.
- Explicit activity position determines order; multi-row reorder operations are transactional.
- Store the trip's time zone as an IANA zone (e.g. `Asia/Tokyo`); distinguish local itinerary clock times from UTC audit timestamps.
- Persist provider identifiers and user-created content only alongside third-party fields permitted by the provider's terms.

## 5. REST API surface (v1)

All APIs return JSON under `/api/v1`.

| Method | Path | Action |
| --- | --- | --- |
| `GET` | `/trips` | List owned trips |
| `POST` | `/trips` | Create trip |
| `GET` | `/trips/{id}` | Read trip |
| `PATCH` | `/trips/{id}` | Update trip |
| `DELETE` | `/trips/{id}` | Delete trip |
| `GET` | `/places/search` | Search external provider |
| `POST` | `/trips/{id}/places` | Save place |
| `GET` | `/trips/{id}/days` | List itinerary days |
| `POST` | `/days/{id}/activities` | Add activity |
| `PATCH` | `/activities/{id}` | Update activity |
| `DELETE` | `/activities/{id}` | Remove activity |
| `PUT` | `/days/{id}/order` | Reorder activities |

**Implementation note:** The source proposal outlines the core routes but does not specify all read/remove endpoints for saved places. Add those routes to the actual API contract as the place collection UI is implemented.

API standards: validate bodies/query parameters, return appropriate status codes and consistent errors, paginate large collections, propagate timeouts/cancellation, enforce authorization for every resource, and never allow the client to override ownership.

## 6. Optimization approach (v2)

### Inputs

Places and starting point; travel mode; activity duration; daily start/end bounds; optional fixed reservations; optional known opening hours.

### Outputs

A proposed ordered activity list with start/end estimates, travel durations, estimated total travel time, feasibility/conflict information, and clearly stated assumptions.

### Algorithm plan

1. **Travel-time matrix:** Request estimates for relevant pairs of locations; model them as directed, weighted edges.
2. **Initial schedule:** Use nearest-neighbor selection of the nearest *feasible* unvisited location, updating the current place and arrival time.
3. **Local improvement:** Apply a route-improvement heuristic such as 2-opt while re-evaluating schedule feasibility. For asymmetric or time-dependent durations, recompute the affected route rather than relying on symmetric shortcuts.
4. **Constraint checks:** Honor opening windows, waiting time, activity duration, fixed events, day bounds and unavailable routes. Explain failures rather than pretending every requested activity fits.
5. **Review and approval:** Show before/after itinerary, estimated savings and unscheduled activities; mutate persisted itinerary only after confirmation.

Nearest-neighbor and 2-opt are practical heuristics, **not guarantees of globally optimal itineraries**. Start with single-day optimization to manage complexity.

## 7. Go concurrency, reliability and security

### Concurrency

Use goroutines where independent API calls justify parallelism, with a **bounded worker pool**, request context cancellation, deadlines, rate-limit compliance, partial-failure collection, and controlled retries. Do not introduce concurrency solely to demonstrate it.

For computationally expensive future optimizations, use a **durable database-backed job queue** with job status and client polling. In-memory channels alone do not provide durable jobs.

### Reliability and observability

Structured logging; p50/p95 request latency; failure and external API request metrics; graceful shutdown; bounded exponential backoff with jitter; retries only for safe-to-repeat or idempotent requests; and monitoring after deployment.

### Security

HTTPS, secure sessions, per-resource authorization, server-side validation, parameterized SQL, restricted CORS, protected secrets, endpoint rate limits, safe error messages, SSRF protection for server-fetched URLs, proper migrations, backup/restore documentation, and third-party licensing compliance. Do not log secrets or sensitive reservation contents.

## 8. Main application screens

1. **My Trips:** Create, list and reopen trips.
2. **Trip Dashboard:** Review destination, dates, saved places and itinerary days.
3. **Place Discovery:** Search, inspect map/results and save locations.
4. **Itinerary Editor:** Select days, adjust ordered activity cards and times, drag to reorder, and see synchronized map markers.
5. **Optimization Comparison (v2):** Review original versus suggested schedules and choose whether to apply.

## 9. Implementation roadmap

| Phase | Deliverables | Exit condition |
| --- | --- | --- |
| **0 — Foundation** | Git repo, Go API, Next.js app, Docker Compose, PostgreSQL, env templates, CI, `/health` endpoint | Frontend successfully requests API health endpoint locally |
| **1 — Trips & data** | Schema/migrations, auth, trip CRUD, frontend integration, tests | Authenticated user creates, restarts and securely retrieves a trip |
| **2 — Places** | Places provider adapter, search/results/map UI, save/remove, quotas and errors | Search and save a place and see it later in the trip |
| **3 — Itinerary** | Trip days, activity CRUD, start/duration fields, ordering/drag-and-drop, map sync, E2E tests | Multi-day itinerary persists across sessions — **MVP complete** |
| **4 — Optimization** | Route matrix, nearest-neighbor, local improvement, feasibility, comparison, benchmarking | Explains a feasible suggested visit order on controlled data |
| **5 — Production hardening** | Bounded concurrency, resilience, logging/metrics, load tests, optional durable jobs, deployment monitoring | Deployed system meets documented reliability/test goals |
| **6 — Expansion** | Optional budget, reservations, export, recommendations, multiple destinations | Features chosen based on usage and feedback |

No fixed dates or team estimates are specified in the source proposal; this roadmap is ordered by dependency, not calendar.

## 10. Testing and quality gates

**Unit:** Dates and durations; authorization decisions; activity ordering; route scoring; feasibility; heuristic behavior.

**Integration:** PostgreSQL migrations, foreign keys, transactions and rollback, cascading deletes, CRUD and cross-user authorization. Mock paid third-party APIs where possible.

**Algorithm cases (v2):** Two places, clustered attractions, fixed midpoint reservation, closed venue, impossible day, unavailable route, equal-duration alternatives and cross-midnight itinerary.

**End-to-end:** Sign in → create trip → search/save place → schedule → reorder → reload → verify persistence.

**Performance (after v2):** Benchmark computation by place count, sequential vs bounded-concurrent data fetching, p50/p95 API latency, calls per optimization, resource utilization and concurrent DB usage. Measure before claiming improvements.

## 11. Risks and boundaries

| Risk | Mitigation |
| --- | --- |
| API cost and usage limits | Provider quotas, narrow field selection, mocked integration tests |
| Inaccurate or incomplete travel estimates | Label estimates, limitations, and assumptions |
| Combinatorial optimization complexity | Single-day scope and explainable heuristics first |
| Cross-resource data inconsistency | Relational constraints, transactions and tests |
| External outage | Timeouts, useful errors and manual itinerary fallback |
| Unauthorized access | Ownership checks at every resource boundary |
| Feature creep | Maintain v1 exclusions and gated milestones |
| Unsupported performance claims | Reproducible benchmarks and recorded measurements |

## 12. Expected engineering deliverables

- Deployed, functional web application and demo itinerary
- Reproducible Docker-based development environment
- Go REST API documentation
- PostgreSQL schema and migrations
- Trip, search, map and itinerary UI
- Unit, integration and browser end-to-end tests
- Architecture documentation and setup README
- **v2 additions:** route-optimization implementation, constraints/limitations documentation, heuristic comparisons and benchmark results

## 13. Non-goals and design principles

- **Functionality before complexity:** Ship manual planning before optimization.
- **User control:** Suggestions do not silently replace plans.
- **Engineering clarity:** Favor a modular monolith over premature microservices.
- **Correctness first:** Persist reliably, authorize consistently and test edge cases.
- **Measure results:** Use benchmarks to justify optimizations and portfolio claims.
- **Scope discipline:** Booking, collaboration and AI features are not MVP blockers.

## 14. Project completion statement

Waypoint's **v1** is a persistent, map-enabled, personal itinerary builder. Its **v2** adds travel-time-aware, constraint-sensitive schedule recommendations that users can inspect and approve. Its longer-term direction is a unified personal travel organizer, developed incrementally with measurable backend, algorithmic, systems and product-engineering quality.

---

*Derived from the supplied “Waypoint — Software Engineering Project Proposal.” This document organizes the existing proposal into a working project-scope specification; it does not imply that any feature has already been implemented.*
