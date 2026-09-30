**Title:** `[BUG]: Replica stuck on "Unknown node" after a node loss; recovery never completes`

### Bug Description

After a node loss, the replacement pods can be introduced only to each other. At that point every well-connected node is in `cluster_state:fail`. The replacements form a sub-cluster that is isolated from the shard's primary. `replicateToShardPrimary` treats `CLUSTER REPLICATE`'s `Unknown node` error as transient and waits for gossip, but the isolated replica never hears about the primary through gossip.

### Steps to Reproduce

1. Create a multi-shard ValkeyCluster with replicas.
2. Lose a Kubernetes node that hosts several of its pods, so the cluster goes to `cluster_state:fail`.
3. Let the replacement pods start.

### Expected Behaviour

The replacement replicas join their shard primaries and the cluster recovers.

### Actual Behaviour

`CLUSTER REPLICATE` fails with `Unknown node` on every reconcile, and the cluster never becomes Ready.
