# v2 milestone 4: authorized optimization preview

POST /api/v1/trips/{trip}/days/{day}/optimization-preview accepts travel_mode
(WALK/DRIVE), start_minute/end_minute (0–1440, increasing), and optional
fixed_activity_ids. Bounds are local wall-clock minutes in the trip time zone.
Only explicitly selected activities use their existing start_time as a fixed
constraint; other MVP start times remain informational. Fixed IDs must be unique,
belong to this day, and have start times.

Verify trip ownership and day membership before provider use. Read saved place
IDs without resolving display details. Limit preview to 20 activities. Budget
matrix elements (n*n for n>=2) at 800/user/minute and 4000/global/minute, using
separate route buckets. These conservative in-process controls target one API
instance; deployments still need Google project quotas. Failed provider calls
consume budget; invalid/unauthorized requests never call the provider.

Return activity-ID-based original/proposed visits, conflicts, travel minutes,
nullable savings, explanation, time zone, mode, and explicit assumptions. No
persisted changes, apply endpoint, or claim of stale-safe approval yet. Preview
reads may become stale while requests run; approval integration must revalidate.

Configure GOOGLE_ROUTES_SERVER_API_KEY separately; absent key/provider gives 503.
Add runtime wiring and a separate optional Compose override to preserve the
user's existing compose.yaml edit. Provider timeout returns a safe 503.

Validation: real PostgreSQL integration tests with fake route provider cover
ownership, foreign day, invalid inputs, quota rejection, provider failure,
explicit fixed starts, response mapping, and unchanged stored itinerary.
Document API/setup and results, commit feature separately from progress.

Status: planned before implementation.

## Verification result

Implemented runtime wiring, the preview endpoint, JSON response mapping, and
optional Compose configuration. Go race tests passed with TEST_DATABASE_URL set
against isolated PostgreSQL schemas; vet passed. Integration coverage verifies
anonymous/foreign-account rejection, nonmember day rejection, invalid mode/bounds/
fixed IDs, ID-based reordered output, explicit fixed-time behavior, provider
timeout, global/user quota rejection before provider calls, absent provider,
and equality of stored days/activities before and after previews.

No schema change or browser UI was introduced. Preview data can become stale;
this milestone provides no apply operation. Live Google calls remain unverified.
