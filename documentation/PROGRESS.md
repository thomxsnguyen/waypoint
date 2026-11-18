# MVP progress

Scope: finish and verify the existing v1 manual trip planner. No routing,
optimization, AI, collaboration, budgets, or reservations are authorized here.

## 2026-11-17

- Reviewed implementation against the MVP scope and implementation guide.
- Baseline TypeScript and lint checks passed. Go race tests and vet passed;
  database integration tests were skipped without a test database.
- Production compilation completed, but the build command exited with a
  Next.js security advisory. Dependency remediation remains a release blocker.
- Live Clerk/Google smoke tests and deployment remain unverified.
- Existing uncommitted `compose.yaml` change belongs to the user and is excluded
  from these commits.
- Next: map authentication failure recovery; see
  [change plan](changes/001-map-retry.md).

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
