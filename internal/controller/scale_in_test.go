/*
Copyright 2025 Valkey Contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	valkeyiov1alpha1 "github.com/valkey-io/valkey-operator/api/v1alpha1"
	"github.com/valkey-io/valkey-operator/internal/valkey"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func primaryValkeyNode(shard int, ip string) valkeyiov1alpha1.ValkeyNode {
	return valkeyiov1alpha1.ValkeyNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("c-%d-0", shard),
			Labels: map[string]string{
				LabelShardIndex: fmt.Sprint(shard),
				LabelNodeIndex:  "0",
			},
		},
		Status: valkeyiov1alpha1.ValkeyNodeStatus{PodIP: ip},
	}
}

func primaryShard(id, ip string, slots ...valkey.SlotsRange) *valkey.ShardState {
	return &valkey.ShardState{
		Id:        "shard-" + id,
		PrimaryId: id,
		Slots:     slots,
		Nodes:     []*valkey.NodeState{{Address: ip, Id: id, Flags: []string{"master"}}},
	}
}

// Spec shrinks 3 -> 5 -> 4 while scaling out: shard 4 has joined and already
// received slots, shard 3 has not joined yet. The live topology then has 4
// shards (0,1,2,4) and spec.Shards is 4, so a count-based check sees nothing
// to drain while shard 4 (index >= spec) owns slots.
func TestScaleInDuringScaleOutOwnedSlots(t *testing.T) {
	nodes := &valkeyiov1alpha1.ValkeyNodeList{Items: []valkeyiov1alpha1.ValkeyNode{
		primaryValkeyNode(0, "10.0.0.0"),
		primaryValkeyNode(1, "10.0.0.1"),
		primaryValkeyNode(2, "10.0.0.2"),
		primaryValkeyNode(3, "10.0.0.3"),
		primaryValkeyNode(4, "10.0.0.4"),
	}}
	state := &valkey.ClusterState{Shards: []*valkey.ShardState{
		primaryShard("n0", "10.0.0.0", valkey.SlotsRange{Start: 400, End: 5461}),
		primaryShard("n1", "10.0.0.1", valkey.SlotsRange{Start: 5462, End: 10922}),
		primaryShard("n2", "10.0.0.2", valkey.SlotsRange{Start: 10923, End: 16383}),
		primaryShard("n4", "10.0.0.4", valkey.SlotsRange{Start: 0, End: 399}),
	}}

	assert.Len(t, state.Shards, 4, "count-based scale-in check sees no excess")
	assert.True(t, excessShardOwnsSlots(state, nodes, 4), "shard 4 owns slots and must be drained")
	assert.True(t, isSlotOwningPrimary(state, "10.0.0.4"), "shard 4 primary must not be deleted")
	assert.False(t, isSlotOwningPrimary(state, "10.0.0.3"), "shard 3 has not joined: safe to delete")

	// Once shard 4 is drained it no longer blocks deletion.
	state.Shards[3].Slots = nil
	assert.False(t, excessShardOwnsSlots(state, nodes, 4))
	assert.False(t, isSlotOwningPrimary(state, "10.0.0.4"))
}

func TestIsSlotOwningPrimaryIgnoresReplicasAndNil(t *testing.T) {
	shard := primaryShard("n0", "10.0.0.0", valkey.SlotsRange{Start: 0, End: 16383})
	shard.Nodes = append(shard.Nodes, &valkey.NodeState{Address: "10.0.0.9", Id: "r0", Flags: []string{"slave"}})
	state := &valkey.ClusterState{Shards: []*valkey.ShardState{shard}}

	assert.True(t, isSlotOwningPrimary(state, "10.0.0.0"))
	assert.False(t, isSlotOwningPrimary(state, "10.0.0.9"), "replicas never own slots")
	assert.False(t, isSlotOwningPrimary(state, ""))
	assert.False(t, isSlotOwningPrimary(nil, "10.0.0.0"))
}
