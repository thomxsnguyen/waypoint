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
