**Title:** `fix: promote orphaned replicas with persistence after a grace period`

This PR closes #<issue>

### Summary

With persistence, a cluster without failover quorum never promoted the replica of a dead primary. It now does so once the primary has been FAIL for a grace period.

### Features / Behaviour Changes

- With persistence, `CLUSTER FAILOVER TAKEOVER` is issued once the primary has been FAIL for 60s. That gives a crashed primary time to restart with its data.
- `VALKEY_OPERATOR_ORPHAN_TAKEOVER_GRACE` (a Go duration) overrides the 60s.
- A primary that is still loading its dataset is never taken over. While loading it refuses cluster-bus connections, so it looks FAIL.
- A primary that is running but not Ready is never taken over. It still answers the cluster bus, so it is not flagged.
- Without persistence nothing changes.

### Implementation

- `promoteOrphanedReplicas` records when each primary is first seen FAIL, on the reconciler. The entry is cleared when the primary recovers or the takeover succeeds.
- TAKEOVER is held while any scraped node reports `loading:1` in INFO. That check is cluster-wide, so a node loading in another shard can delay it, but it can never force one.
- The state is in memory, so an operator restart starts the wait again.

### Limitations

- Recovery takes about `cluster-node-timeout` (15s), plus the grace period, plus the wait for the next reconcile.
- The grace period is measured from the first time a reconcile sees the primary flagged failed. The operator cannot tell whether it stayed failed between reconciles.
- A primary that is partitioned rather than dead, for longer than the grace period, is taken over. When it returns it rejoins at the lower epoch as a replica, and writes it accepted while partitioned are lost.
- This touches the same early return as #477. Whichever merges second will need a rebase.
- The grace period is an env var, not a CRD field. Happy to switch if a field is preferred.

### Testing

- `TestPromoteOrphanedReplicas` covers:
  - without persistence, immediate takeover;
  - with persistence: the first FAIL, after the grace period, while a node is loading, and a recovered primary.
- The loading case fails without the guard.
- `TestOrphanTakeoverGrace` covers the env parsing.
- Uses valkey-go's own mock package, `github.com/valkey-io/valkey-go/mock`. It adds `go.uber.org/mock` as a test dependency.

### Checklist

- [x] This Pull Request is related to one issue.
- [x] Commit message explains what changed and why
- [x] Tests are added or updated.
- [x] Documentation files are updated.
- [x] I have run pre-commit locally (`pre-commit run --all-files` or hooks on commit)
