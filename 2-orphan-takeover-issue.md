**Title:** `[BUG]: With persistence, a cluster without failover quorum never promotes the replica of a dead primary`

### Bug Description

When a primary dies and the cluster has no failover quorum (a single-shard cluster, or 2 of 4 pods), Valkey cannot promote the replica, so the operator has to issue `CLUSTER FAILOVER TAKEOVER`. `promoteOrphanedReplicas` returns early whenever `spec.persistence` is set, on the assumption that the primary will come back with its data. If the primary's node or volume is gone it never comes back, and the shard stays down.

### Steps to Reproduce

1. Create a ValkeyCluster with `shards: 1`, `replicas: 1` and `persistence` set.
2. Make the primary unrecoverable, e.g. delete its PVC and pod, or cordon and drain its node.

### Expected Behaviour

After a reasonable wait, the replica is promoted.

### Actual Behaviour

The replica stays a replica and the cluster reports `cluster_state:fail` indefinitely.
