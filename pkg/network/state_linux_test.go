// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux_bpf

package network

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DataDog/datadog-agent/pkg/process/util"
)

func TestStatsOverflow(t *testing.T) {
	conn := ConnectionStats{
		Pid:       123,
		Type:      TCP,
		Family:    AFINET,
		Source:    util.AddressFromString("127.0.0.1"),
		Dest:      util.AddressFromString("127.0.0.1"),
		Monotonic: StatCounters{SentPackets: math.MaxUint32 - 1, RecvPackets: math.MaxUint32 - 2},
		IntraHost: true,
	}

	client := "client"

	state := newDefaultState()

	// Register the client
	state.RegisterClient(client)

	// Get the connections once to register stats
	delta := state.GetDelta(client, latestEpochTime(), []ConnectionStats{conn}, nil, nil)
	require.Equal(t, 1, delta.Conns.Len())

	// Expect Last.SentPackets to be math.MaxUint32-1
	conn.Last.SentPackets = math.MaxUint32 - 1
	// expect Last.RecvPackets to be math.MaxUint32-2
	conn.Last.RecvPackets = math.MaxUint32 - 2
	assert.Equal(t, conn, delta.Conns.Active[0])

	// Get the connections again but by simulating an overflow
	conn.Monotonic.SentPackets = 10
	conn.Monotonic.RecvPackets = 11

	delta = state.GetDelta(client, latestEpochTime(), []ConnectionStats{conn}, nil, nil)
	require.Equal(t, 1, delta.Conns.Len())
	assert.Equal(t, uint64(12), delta.Conns.Active[0].Last.SentPackets)
	assert.Equal(t, uint64(14), delta.Conns.Active[0].Last.RecvPackets)
}
