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
Any changed snapshot invalidates the preview. Activity/trip timestamps also detect
edit-and-revert; saved-place notes have no timestamp and are compared by value. Concurrent
apply requests serialize at the trip lock; completed approvals are removed.
Conservative invalidation on unrelated trip/saved-place edits is acceptable.

Validation: PostgreSQL tests cover apply persistence, original preserved before
approval, wrong account/day, unknown/expired IDs, edited/reverted inputs, replay,
and concurrent application. No provider call on apply. Document restart/expiry
and single-instance limits. UI, visiting windows, and origin inputs remain pending.

## Verification result

Implemented the bounded approval cache, source snapshot hashing, explicit apply
route, and transactional store operation. Full Go race suite with isolated
PostgreSQL and vet passed. Targeted PostgreSQL race tests and vet passed again
after adding mutation-during-preview and infeasible-preview coverage.

Tests confirm edit-and-revert detection for activities, expiry, wrong owner/day,
unknown IDs, rejection of client-supplied schedule fields, persisted approved
order/times, preserved notes/durations, replay rejection, and exactly one success
for concurrent applies. Apply does not call the Routes provider. Snapshot checks
reject a trip mutation during provider computation before issuing approval.

No schema migration, UI, or paid provider call. Saved-place notes have no update
timestamp, so their transient edit-and-revert is not detected; their current
value is included and they do not affect schedule computation. Approval storage
is ephemeral and bounded, consistent with the single-instance deployment model.
