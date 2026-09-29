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
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vclient "github.com/valkey-io/valkey-go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"

	valkeyiov1alpha1 "github.com/valkey-io/valkey-operator/api/v1alpha1"
	"github.com/valkey-io/valkey-operator/internal/valkey"
)

// fakeValkey is a minimal RESP server that answers the client handshake,
// replies "Unknown node" to CLUSTER REPLICATE and records every command.
type fakeValkey struct {
	mu   sync.Mutex
	cmds []string
}

func (f *fakeValkey) commands() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.cmds...)
}

func (f *fakeValkey) serve(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	r := bufio.NewReader(conn)
	for {
		args, err := readRESPArray(r)
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.Join(args, " "))
		f.mu.Lock()
		f.cmds = append(f.cmds, cmd)
		f.mu.Unlock()

		var reply string
		switch {
		case strings.HasPrefix(cmd, "HELLO"):
			reply = "%1\r\n+proto\r\n:3\r\n"
		case strings.HasPrefix(cmd, "PING"):
			reply = "+PONG\r\n"
		case strings.HasPrefix(cmd, "CLUSTER REPLICATE"):
			reply = "-ERR Unknown node " + args[2] + "\r\n"
		default:
			reply = "+OK\r\n"
		}
		if _, err := conn.Write([]byte(reply)); err != nil {
			return
		}
	}
}

func readRESPArray(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, "*") {
		return nil, fmt.Errorf("unexpected RESP line %q", line)
	}
	n, err := strconv.Atoi(strings.TrimSpace(line[1:]))
	if err != nil {
		return nil, err
	}
	args := make([]string, 0, n)
	for range n {
		if _, err := r.ReadString('\n'); err != nil { // $<len>
			return nil, err
		}
		arg, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		args = append(args, strings.TrimSuffix(arg, "\r\n"))
	}
	return args, nil
}

// A replacement node that was never introduced to its shard's primary keeps
// failing CLUSTER REPLICATE with "Unknown node"; gossip alone may never fix
// it. The operator must MEET the primary from the replica before retrying.
func TestReplicateToShardPrimaryMeetsPrimaryOnUnknownNode(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()
	fake := &fakeValkey{}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go fake.serve(conn)
		}
	}()

	c, err := vclient.NewClient(vclient.ClientOption{
		InitAddress:       []string{ln.Addr().String()},
		ForceSingleClient: true,
		DisableCache:      true,
	})
	require.NoError(t, err)
	defer c.Close()

	nodes := &valkeyiov1alpha1.ValkeyNodeList{Items: []valkeyiov1alpha1.ValkeyNode{{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "c-0-0",
			Labels: map[string]string{LabelShardIndex: "0", LabelNodeIndex: "0"},
		},
		Status: valkeyiov1alpha1.ValkeyNodeStatus{PodIP: "10.0.0.1"},
	}}}
	state := &valkey.ClusterState{Shards: []*valkey.ShardState{{
		Id:        "shard-0",
		PrimaryId: "primary0",
		Slots:     []valkey.SlotsRange{{Start: 0, End: 16383}},
		Nodes:     []*valkey.NodeState{{Address: "10.0.0.1", Port: DefaultPort, Id: "primary0", Flags: []string{"master"}}},
	}}}
	replica := &valkey.NodeState{Client: c, Address: "10.0.0.2", Port: DefaultPort, Id: "replica0"}

	r := &ValkeyClusterReconciler{Recorder: events.NewFakeRecorder(10)}
	cluster := &valkeyiov1alpha1.ValkeyCluster{ObjectMeta: metav1.ObjectMeta{Name: "c", Namespace: "default"}}

	err = r.replicateToShardPrimary(context.Background(), cluster, state, replica, 0, nodes)
	assert.True(t, errors.Is(err, errPrimaryNotReady), "expected errPrimaryNotReady, got %v", err)

	cmds := fake.commands()
	assert.Contains(t, cmds, "CLUSTER REPLICATE PRIMARY0")
	assert.Contains(t, cmds, fmt.Sprintf("CLUSTER MEET 10.0.0.1 %d", DefaultPort))
}
