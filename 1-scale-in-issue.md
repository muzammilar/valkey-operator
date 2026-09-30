**Title:** `[BUG]: Lowering spec.shards during a scale-out deletes a primary that still owns slots`

### Bug Description

`handleScaleIn` only drains when `len(state.Shards) > spec.shards`. If `spec.shards` is lowered while a scale-out is still running, a new high-index shard can already own slots while a lower-index new shard has not joined. The shard count then matches the spec, nothing is drained, and `deleteExcessValkeyNodes` deletes the high-index shard's primary with its slots. Those keys are lost.

### Steps to Reproduce

1. Create a ValkeyCluster with `shards: 2`, `replicas: 0` and write some keys.
2. Set `shards: 4`.
3. As soon as shard 3 has slots but before shard 2 joins, set `shards: 3`.

### Expected Behaviour

Shard 3's slots are drained to shards 0–2 before its ValkeyNode is deleted.

### Actual Behaviour

Shard 3's ValkeyNode is deleted while it still owns slots. The slots are later reassigned empty and their keys are gone.
