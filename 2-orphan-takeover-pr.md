**Title:** `fix: promote orphaned replicas with persistence after a grace period`

This PR closes #<issue>

### Summary

With persistence, a cluster without failover quorum never promoted the replica of a dead primary. It now does so once the primary has been FAIL for a grace period.

### Features / Behaviour Changes

- With persistence, `CLUSTER FAILOVER TAKEOVER` is issued once the primary has been FAIL for 60s. That gives a crashed primary time to restart with its data.
- `VALKEY_OPERATOR_ORPHAN_TAKEOVER_GRACE` (a Go duration) overrides the 60s.
- Without persistence nothing changes.

### Implementation

- `promoteOrphanedReplicas` records when each primary is first seen FAIL, on the reconciler. The entry is cleared when the primary recovers or the takeover succeeds.
- The state is in memory, so an operator restart starts the wait again.

### Limitations

- A primary that is partitioned rather than dead, for longer than the grace period, is taken over. When it returns it rejoins at the lower epoch as a replica, and writes it accepted while partitioned are lost.
- This touches the same early return as #477. Whichever merges second will need a rebase.
- The grace period is an env var, not a CRD field. Happy to switch if a field is preferred.

### Testing

- `TestPromoteOrphanedReplicas` covers:
  - without persistence, immediate takeover;
  - with persistence: the first FAIL, after the grace period, and a recovered primary.
- `TestOrphanTakeoverGrace` covers the env parsing.
- Commands go through the real valkey-go client to a small in-process RESP server (`fake_valkey_test.go`).

### Checklist

- [x] This Pull Request is related to one issue.
- [x] Commit message explains what changed and why
- [x] Tests are added or updated.
- [x] Documentation files are updated.
- [x] I have run pre-commit locally (`pre-commit run --all-files` or hooks on commit)
