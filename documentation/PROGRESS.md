# Project progress

Initial scope: finish and verify the v1 manual trip planner. On 2026-11-17,
the user authorized v2 single-day smart planning under the project scope.
AI, collaboration, budgets, and booking remain outside this work.

## 2026-11-17

- Reviewed implementation against the MVP scope and implementation guide.
- Baseline TypeScript and lint checks passed. Go race tests and vet passed;
  database integration tests were skipped without a test database.
- Production compilation completed, but the build command exited with a
  Next.js security advisory. Dependency remediation remains a release blocker.
- Live Clerk/Google smoke tests and deployment remain unverified.
- Existing uncommitted `compose.yaml` change belongs to the user and is excluded
  from these commits.
- Fixed map authentication retry; added regression coverage to the existing
  itinerary journey. Confirmed the original loader fails the new assertion
  (zero markers after retry) and the corrected loader passes.
- Fixed a browser navigation race by waiting for the destination screen and
  scoping activity removal to its card.
- Updated Next.js and eslint-config-next from 16.4.4 to 16.4.5. Production build
  now exits successfully. The npm audit no longer flags Next.js itself.
- Verified Go race tests with real isolated PostgreSQL integration tests, Go vet,
  TypeScript, lint, production build, and both browser scenarios.

### Remaining release work

- npm audit still reports 15 affected dependency entries (6 moderate, 9 high)
  through development lint/glob dependencies, including minimatch and picomatch.
  Investigate compatible transitive patches separately; do not use a forced
  framework downgrade or describe this as a clean audit.
- Live Clerk/Google staging verification and deployment/operator configuration
  remain outstanding. Automated provider doubles do not prove those settings.
- No v2 features were introduced.

Commit convention: Conventional Commits, one focused change per commit.

## Development workflow

1. Compare each proposed change with committed MVP scope.
2. Write a change plan before implementation, including acceptance criteria.
3. Keep changes small and add regression coverage for concrete failures.
4. Use an isolated test database and provider doubles for automated checks.
5. Run relevant static checks, integration tests, and browser flows; investigate
   failures rather than hiding them with sleeps or retries.
6. Review the diff and secret handling, update this log with actual results,
   and commit using Conventional Commits. Preserve unrelated user changes.
7. Keep live-provider staging and deployment checks distinct from automated tests.

## v2 milestone 1 — schedule evaluation

- Documented the milestone before implementation in changes/004-v2-schedule-evaluation.md.
- Added a pure Go evaluator for directed travel matrices, fixed starts, waiting,
  unavailable routes, and day boundaries. Inputs and persisted plans are unchanged.
- Regression tests cover asymmetric legs, conflicts, preservation of activities,
  invalid orders/matrices, and duplicate activity IDs.
- Go race tests and vet passed. Database tests skip without TEST_DATABASE_URL;
  this milestone changes no persistence or HTTP behavior.
- Next: provider matrix adapter, proposal generation, preview/apply API with
  stale-plan protection, and comparison UI. v2 is not yet user-facing or complete.

## v2 milestone 2 — proposal generation

- Committed the plan before implementation (`428e782`), then implemented and
  tested the heuristic in a separate feature commit.
- Sequencing adjustment: verify proposal generation with controlled matrices
  before connecting the travel-time provider. Scope remains single-day v2.
- Try nearest-feasible-neighbor routes from each starting stop and up to five
  directed segment-reversal improvement passes. Maximum 20 stops; cancellation
  is supported. Every trial is evaluated against duration and fixed-start rules.
- Proposals retain all activities, keep equal-cost originals, explain search
  failure, and omit savings when the baseline is infeasible. No saved data changes.
- Go race tests and vet passed; routing statement coverage: 96.9%. Fixtures cover
  travel reduction, asymmetric/unavailable legs, fixed starts, impossible days,
  deterministic output, input preservation, cancellation, and limits.
- Database integration tests skipped without a test database in this milestone.
  No frontend changes or live provider claims.
- Remaining v2: provider-backed estimates and supported travel modes; visiting
  windows and origin inputs; preview/apply API with ownership and stale-plan
  checks; comparison UI; integration/browser validation and controlled benchmark.
- Existing v1 release caveats remain open. Full v2 is not complete.

## v2 milestone 3 — provider matrix adapter

- Plan committed first (`d74c7d6`); implemented adapter and contract tests in a
  separate feature commit. Checked Google's official Routes REST reference.
- Added WALK and traffic-unaware DRIVE estimates, preserving directed activity
  indices and rounding fractional seconds up to whole minutes.
- Enforced 20 stops, 10-second deadline, response-size bound, no redirects or
  automatic retries. Invalid/missing matrix elements fail safely; unavailable
  routes remain explicit. No credentials or provider details enter errors.
- Verified adapter-to-proposal integration with a controlled fixture. Go race
  tests and vet passed; database integration tests skipped without a test DB.
- No paid provider calls or frontend changes. The adapter is not yet exposed by
  the API. Ownership and per-user/global element quotas must precede that step.
- Next: authorized preview integration and provider configuration, followed by
  stale-safe approval/apply and comparison UI. Transit/departure-time behavior,
  visiting windows, origin inputs, and live-provider validation remain pending.

## v2 milestone 4 — authorized preview API

- Plan committed before implementation (`633d8a6`). Added trip/day preview route
  with ownership and day-membership checks, explicit fixed activity selection,
  day bounds, and WALK/DRIVE selection.
- Runtime uses a separate server Routes key. Added optional compose.routing.yaml
  without modifying the user's existing compose.yaml change.
- Separate route budgets enforce 800 elements/user/minute and 4000/global/minute
  for one API instance. A preview consumes n*n elements for n>=2, even on provider
  failure. Project-level quotas remain necessary across restarts/instances.
- Returned original/proposed schedules use activity IDs and explain assumptions;
  unknown estimates use -1 and infeasible-baseline savings are null. No writes.
- Real PostgreSQL integration tests plus full Go race suite and vet passed.
  Tests cover authorization, validation, quotas, timeout, fixed-time behavior,
  response mapping, and unchanged persisted activities.
- API/setup documentation updated. No paid Google requests or frontend changes.
- Next: approval/apply with stale-plan protection and comparison UI. Visiting
  windows, origin inputs, transit semantics, and live-provider checks remain open.

## v2 milestone 5 — approval and apply

- Plan committed first (`f297137`), followed by the implementation and tests.
- Feasible previews issue opaque approval IDs valid for ten minutes. The bounded
  in-memory cache supports one API instance and invalidates approvals on restart.
- Snapshot checks bracket preview computation. Apply reauthorizes ownership,
  checks approval scope, locks the trip, compares the source snapshot, and saves
  the approved activity order/start times in one transaction. Notes/durations stay.
- Unknown, expired, replayed, wrong-scope, and stale approvals are rejected.
  Client-supplied schedules are rejected. No provider calls occur during apply.
- Full Go race suite with isolated PostgreSQL and vet passed. Additional targeted
  race tests passed after expanding coverage for infeasible previews and mutation
  during provider computation. Concurrent apply test confirms one commit/one 409.
- API/setup docs now describe approval expiry, restart behavior, and snapshot
  semantics. No schema migration, UI, or live provider request was introduced.
- Next: comparison/approval UI and browser coverage. Origin inputs, visiting
  windows, transit semantics, and live-provider staging verification remain open;
  v2 is not complete.

## v2 milestone 6 — comparison and approval UI

- Plan committed first (`d8302a4`); UI and browser coverage committed separately.
- Added Optimize day with WALK/DRIVE, local day bounds, and explicit fixed-start
  checkboxes. Comparison shows named visits, estimated times/travel, savings,
  conflicts, and limitations. Cancel preserves saved data; Apply sends approval ID.
- Changed inputs discard the preview. Stale/failed apply clears the old approval
  and prompts regeneration. Successful apply reloads the itinerary and map state.
- Added a Routes double only to the build-tagged browser-test API. The new real
  API/PostgreSQL browser scenario covers preview/cancel preservation, input-change
  invalidation, another-tab edit rejection, explicit apply, and reload persistence.
- Typecheck, lint, production build, Go race/integration suite, vet, and test-only
  API compilation passed. Full browser suite: 3 passed; expanded optimization
  scenario passed separately. Mobile form and comparison screenshots visually
  reviewed, and overflow assertion passed.
- README/API docs updated for the user-visible flow. No paid provider requests.
- Remaining scope: origin inputs, visiting windows, transit semantics, controlled
  benchmarks, and live-provider staging verification. Existing release caveats
  remain open; full v2 is not complete.

## v2 milestone 7 — visiting windows

- Plan committed first (`fbef2fd`); backend constraints committed separately
  (`13047a2`) from UI and browser coverage.
- Added optional per-activity earliest-start/latest-finish windows. Evaluation
  waits for opening and requires the full duration to fit; exact closing is valid.
  Greedy/reversal candidates honor windows together with fixed starts/day bounds.
- Preview rejects malformed/duplicate/foreign windows before provider work;
  impossible valid windows retain activities, show conflicts, and issue no approval.
- UI collects checked-by-user windows for the current preview only. Constraints
  are not saved as reusable venue hours; Apply saves reviewed order/start times.
- Go race suite with PostgreSQL and vet passed. Frontend typecheck, lint,
  production build, and all three browser scenarios passed. New browser coverage
  verifies partial input, infeasible approval disabled, exact-closing apply, and
  time persistence after reload. Mobile controls inspected and overflow checked.
- Remaining v2: starting location, transit/departure semantics, controlled
  benchmarks, live-provider staging verification, and existing release caveats.
