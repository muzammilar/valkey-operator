**Title:** `fix: CLUSTER MEET the shard primary when REPLICATE reports Unknown node`

This PR closes #<issue>

### Summary

A replica isolated from its shard primary never learns it through gossip, so `CLUSTER REPLICATE` failed with `Unknown node` forever. The replica now sends `CLUSTER MEET` to the primary, and the next reconcile retries.

### Implementation

- `replicateToShardPrimary`: on `Unknown node`, send `CLUSTER MEET <primary IP> 6379` from the replica before returning `errPrimaryNotReady`. A failed MEET is only logged.

### Testing

- `TestReplicateToShardPrimaryMeetsUnknownPrimary` checks that `Unknown node` sends REPLICATE and then MEET, and that a successful REPLICATE sends no MEET. It fails without the fix.
- Commands go through the real valkey-go client to a small in-process RESP server (`fake_valkey_test.go`).

### Checklist

- [x] This Pull Request is related to one issue.
- [x] Commit message explains what changed and why
- [x] Tests are added or updated.
- [ ] Documentation files are updated. (no user-facing change)
- [x] I have run pre-commit locally (`pre-commit run --all-files` or hooks on commit)
