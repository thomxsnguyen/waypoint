# Waypoint

A personal trip planner built from the [MVP implementation guide](documentation/implementation/WAYPOINT_MVP_IMPLEMENTATION.md). Next.js / React / TypeScript, a Go API, and PostgreSQL.

Implemented: Clerk accounts, owned trip CRUD, Google Places search, saved places and notes, day-by-day activity editing, accessible ordering, and synchronized Google Maps markers. No route optimization, budgets, reservations, AI, collaboration, or background workers.

**Release status:** The MVP code and automated checks are implemented. Production release still requires your Clerk and Google credentials, a deployment target, and a live-provider staging smoke test. Automated browser tests use external-service doubles and a real Go API/PostgreSQL database; they do not prove a live Clerk login or Google billing/key configuration.

## Run locally

Prerequisites: Docker Compose. For native development: Go 1.26+, Node.js 24+, npm, and PostgreSQL 17.

1. Copy `.env.example` to `.env`.
2. Create a Clerk application with **personal accounts enabled**. Set `CLERK_SECRET_KEY`, `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`, and `AUTH_ISSUER` (the exact `https://…clerk.accounts.dev` issuer for development, without a trailing slash). Allow `http://localhost:3000` as your frontend origin. Clerk provides registration, sign-in, and sign-out. Default Clerk session tokens need no custom JWT template. Set `AUTH_AUDIENCE` only if you also add the matching `aud` claim in Clerk.
3. Enable **Places API (New)** and **Maps JavaScript API** in your Google Cloud project with billing. Set `GOOGLE_MAPS_SERVER_API_KEY` and `NEXT_PUBLIC_MAPS_BROWSER_API_KEY`. Use separate keys: restrict the server key to Places and your server environment; restrict the browser key to Maps JavaScript and your frontend referrers. Set provider quotas/budget alerts. `DEMO_MAP_ID` works for development; supply your own map ID for release.
4. Run:

```sh
docker compose up --build
```

Open **http://localhost:3000**. API health: **http://localhost:8080/health**. Sign in, create a trip, save some places, and add them to itinerary days. The PostgreSQL volume survives normal `docker compose down` and restarts. `docker compose down -v` destroys local data.

Keys are intentionally not supplied. The API refuses to start without authentication configuration. Missing Google credentials produce an unavailable state while existing itinerary data remains editable. The browser uses only publishable keys; server secrets are never public environment variables.

### Native development

Start PostgreSQL yourself or use the isolated container pattern below (use a separate `waypoint` database for development). Export the root `.env` for the backend, and copy the needed `NEXT_PUBLIC_*` values plus `CLERK_SECRET_KEY` into `frontend/.env.local`.

```sh
# Shell 1, from repository root (after configuring .env)
set -a
. ./.env
set +a
cd backend
go run ./cmd/api

# Shell 2
cd frontend
npm ci
npm run dev
```

The API applies numbered migrations transactionally at startup, with a database advisory lock to serialize runners. No manual schema setup is required.

## Tests

Use a **separate test database**. Tests create and drop unique schemas; they never reset your development schema. The browser harness additionally requires a database name ending in `_test`.

```sh
docker run --name waypoint-test-db \
  -e POSTGRES_USER=waypoint -e POSTGRES_PASSWORD=waypoint_test \
  -e POSTGRES_DB=waypoint_test \
  -p 127.0.0.1:55432:5432 -d postgres:17-alpine

export TEST_DATABASE_URL='postgres://waypoint:waypoint_test@localhost:55432/waypoint_test?sslmode=disable'

cd backend
go test -race ./...
go vet ./...

cd ../frontend
npm ci
npm run typecheck
npm run lint
npm run build
npx playwright install chromium
npm run test:e2e
```

Without `TEST_DATABASE_URL`, Go integration tests explicitly skip; unit tests still run. CI supplies PostgreSQL and runs all checks, including the browser journey. The browser test starts its own servers at ports **3100** and **18080**, so keep those ports free. It tests persistence, multi-day editing, map/card selection in both directions, reorder, sign-out/sign-in, ownership isolation, mobile width, and deletion. No paid provider calls are made.

Test authentication lives in a separate Go executable behind the `e2e` build tag and a development-only frontend alias. The production API has no test-token branch. `WAYPOINT_E2E=1` is rejected for production builds, and CI checks that test login strings are absent from production bundles.

Go tests also exercise real RSA-signed session tokens through Clerk's official verifier, including signature, expiry, issuer, audience, origin, and signing-key rotation. Provider contract tests assert outbound Google requests and field mapping without calling Google.

## Architecture and behavior

- `backend/internal/domain`: DTOs and validation.
- `backend/internal/store`: parameterized SQL, ownership, transactions, and persistence policies.
- `backend/internal/auth`: Clerk session verification adapter.
- `backend/internal/places`: Google Places adapter with narrow field masks and timeouts.
- `backend/internal/httpapi`: REST handlers, JSON errors, authentication, CORS, rate limits, and request logs.
- `backend/migrations`: embedded up migration, explicit destructive down migration.
- `frontend/app`: trip list, dashboard, discovery, and itinerary routes.
- `frontend/components`: forms, shared session, and Google map integration.

The API verifies every resource's owner, including day/activity IDs. Composite foreign keys prevent cross-trip activity references. Mutations lock the trip row so date edits, place removal, appends, reorders, and deletions cannot race across tabs. A reorder must be an exact permutation; positions remain contiguous. Concurrent edits use last successful write wins, followed by a frontend refetch.

Dates are calendar dates; activity times are local wall-clock values in the trip's IANA time zone. Trip dates expand automatically. Shortening a range that contains activities returns `409`. Activity ordering is authoritative; entering a time does not reorder anything. Activities ending after midnight show a warning without moving days. Conflict solving is not implemented.

Saved places store Google place IDs and user-created notes only. Display details and coordinates are resolved on reads and are not persisted. Missing provider details preserve saved places/activities and omit unavailable markers. Repeated saves return the existing row without replacing notes. Removing a referenced saved place returns `409` until its activities are removed. Removing a trip deletes all its dependent planning data.

Limits: 366 days per trip, 100 saved places per trip, 100 activities per day, 1–1,440 minute durations, 5,000-byte notes/descriptions, 20 search results per query, and 50 trips per list page. HTTP bodies are capped at 32 KiB. Lookup concurrency is bounded at five; provider requests have timeouts. In-process limits allow 240 API requests per user/minute and 240 place lookups per user/minute, with a 2,000/minute global provider cap. Configure lower Google project quotas as appropriate; this MVP targets one API instance.

API contract: [documentation/api.md](documentation/api.md).

## Deployment and recovery

Deploy the two Dockerfiles as separate services with PostgreSQL 17. Frontend `NEXT_PUBLIC_*` values are **build-time** values; rebuild when they change. Supply the Clerk secret to both runtime services. The API needs `DATABASE_URL`, `AUTH_ISSUER`, `CORS_ALLOWED_ORIGIN`, and the server Places key. Use TLS for the browser/API and database connection, set the exact HTTPS frontend origin, and keep database access private. Compose binds services to loopback for local development; configure ingress for your deployment.

Before release, complete the deployment checklist in the implementation guide: configure production auth origins, restricted keys, quotas/alerts, log access, backups, and a real sign-in → search → save → schedule → reload → sign-out/sign-in staging smoke test. Verify a second account cannot open the first account's trip. Check Google attribution and map selection with live data. Review the public `/terms` and `/privacy` pages for your deployment's operator and retention policy.

The API emits structured JSON request count/status/duration records and provider-failure logs; it does not log tokens or note bodies. `/health` checks PostgreSQL. Shutdown drains requests for up to ten seconds. Deployment monitoring and backup retention are operator configuration, not additional application services.

Back up before migrations and on a regular schedule. Example with separately configured database URLs:

```sh
pg_dump --format=custom --file=waypoint.dump "$DATABASE_URL"
# Restore only into a NEW, empty recovery database:
pg_restore --no-owner --dbname="$RECOVERY_DATABASE_URL" waypoint.dump
```

Test restores before relying on backups. The down migration drops all planning tables and is intended only for disposable databases; for production, restore a tested backup or apply a forward repair migration. If deliberately rolling back a disposable database, run `001_initial.down.sql` in a transaction and delete version `1` from `schema_migrations` in that same transaction.

## Provider references

Implementation follows [Clerk's Next.js integration](https://clerk.com/docs/nextjs/getting-started/quickstart) and [official Go session verification](https://clerk.com/docs/guides/sessions/verifying), plus [Google Places field masks](https://developers.google.com/maps/documentation/places/web-service/choose-fields), [storage/attribution policies](https://developers.google.com/maps/documentation/places/web-service/policies), and [advanced map markers](https://developers.google.com/maps/documentation/javascript/advanced-markers/basic-customization). Review provider terms for your billing region before deployment.
