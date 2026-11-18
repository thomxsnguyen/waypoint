# v2 milestone 1: directed travel-time schedule evaluation

Authorized 2026-11-17: begin v2 under the existing project scope.

Before proposing improved orders, implement a deterministic evaluator that can
compare schedules using a supplied directed travel-time matrix. Matrix entries
are minutes; -1 means unavailable. The first stop starts at the day boundary
(no inferred hotel/origin leg). Each stop has a positive duration and optional
fixed start. Waiting is allowed; missed fixed starts, unavailable legs, and day
overruns produce explicit conflicts. Every activity remains in the result.

Reject invalid matrices, duplicate IDs, invalid bounds, and malformed orders.
Never mutate input or persistence. This foundation has no HTTP endpoint and no
provider calls; supplied matrices in tests are fixtures, not real estimates.

Acceptance: asymmetric travel, waiting, fixed-time conflict, unreachable leg,
invalid permutation, impossible day, and input preservation tests pass.

Next milestones: provider-backed matrix adapter with quotas/timeouts; feasible
single-day proposal heuristic; authorized preview/apply API with stale-plan
protection; comparison UI and integration/browser tests. No multi-day optimizer,
AI, booking, budgets, collaboration, or workers.
