**Title:** `fix: keep slot-owning primaries when spec.shards drops mid scale-out`

This PR closes #<issue>

### Summary

Lowering `spec.shards` during a scale-out could delete a primary that still owned slots, losing its keys.

### Features / Behaviour Changes

- Scale-in also drains when a shard at or above `spec.shards` owns slots, not only when there are too many shards.
- An excess ValkeyNode whose pod is a slot-owning primary is never deleted. A `ScaleInBlocked` Warning event is emitted instead.

### Implementation

- `handleScaleIn`: new `excessShardOwnsSlots` check next to the shard-count check.
- `deleteExcessValkeyNodes`: takes the cluster state and skips slot-owning primaries.

### Testing

- `TestExcessShardOwnsSlots` covers the 2 → 4 → 3 case, where the shard count matches the spec but shard 3 owns slots.
- `TestDeleteExcessValkeyNodesKeepsSlotOwners` uses the fake client. It fails without the guard.

### Checklist

- [x] This Pull Request is related to one issue.
- [x] Commit message explains what changed and why
- [x] Tests are added or updated.
- [x] Documentation files are updated.
- [x] I have run pre-commit locally (`pre-commit run --all-files` or hooks on commit)
