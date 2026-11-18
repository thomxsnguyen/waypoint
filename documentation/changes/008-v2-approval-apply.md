# v2 milestone 5: explicit approval and stale-safe apply

Before implementation: extend feasible previews with an opaque random approval
ID, expiring after ten minutes. Store its owner/trip/day, exact proposed visits,
and source snapshot digest in bounded API memory (1000 outstanding previews).
Expired entries are pruned. Restart invalidates previews; one API instance is
supported. No client-supplied order/times are accepted by apply.

POST /api/v1/trips/{trip}/days/{day}/optimization-apply accepts only approval_id.
Reauthorize trip ownership. Reject unknown/expired/wrong-scope approvals with
409. Infeasible previews receive no approval ID. Preview issuance checks a source
snapshot before and after computation so mixed/stale reads cannot be approved.

Snapshot hashes include trip metadata, selected day, activities (including
updated_at), and saved places. Apply locks the trip row, compares the current
snapshot, validates the exact activity permutation and times, and updates order
and start times in one transaction. Notes/durations/place associations stay intact.
Any intervening edit, including edit-and-revert, invalidates the preview. Concurrent
apply requests serialize at the trip lock; completed approvals are removed.
Conservative invalidation on unrelated trip/saved-place edits is acceptable.

Validation: PostgreSQL tests cover apply persistence, original preserved before
approval, wrong account/day, unknown/expired IDs, edited/reverted inputs, replay,
and concurrent application. No provider call on apply. Document restart/expiry
and single-instance limits. UI, visiting windows, and origin inputs remain pending.
