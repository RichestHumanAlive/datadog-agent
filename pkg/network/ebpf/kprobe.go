// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux

package ebpf

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"unsafe"

	"github.com/DataDog/datadog-agent/pkg/process/util"
)

// Family returns whether a tuple is IPv4 or IPv6
func (t ConnTuple) Family() ConnFamily {
	if t.Metadata&uint32(IPv6) != 0 {
		return IPv6
	}
	return IPv4
}

// Type returns whether a tuple is TCP or UDP
func (t ConnTuple) Type() ConnType {
	if t.Metadata&uint32(TCP) != 0 {
		return TCP
	}
	return UDP
}

// SourceAddress returns the source address
func (t ConnTuple) SourceAddress() util.Address {
	if t.Family() == IPv4 {
		return util.V4Address(uint32(t.Saddr_l))
	}
	return util.V6Address(t.Saddr_l, t.Saddr_h)
}

// SourceEndpoint returns the source address and source port joined
func (t ConnTuple) SourceEndpoint() string {
	return net.JoinHostPort(t.SourceAddress().String(), strconv.Itoa(int(t.Sport)))
}

// DestAddress returns the destination address
func (t ConnTuple) DestAddress() util.Address {
	if t.Family() == IPv4 {
		return util.V4Address(uint32(t.Daddr_l))
	}
	return util.V6Address(t.Daddr_l, t.Daddr_h)
}

// DestEndpoint returns the destination address and source port joined
func (t ConnTuple) DestEndpoint() string {
	return net.JoinHostPort(t.DestAddress().String(), strconv.Itoa(int(t.Dport)))
}

// SetFamily sets the family (IPv4 or IPv6) for a tuple.
func (t *ConnTuple) SetFamily(family ConnFamily) {
	if family == IPv6 {
		t.Metadata |= uint32(IPv6) // Set the IPv6 bit
	} else {
		t.Metadata &^= uint32(IPv6) // Clear the IPv6 bit, assuming IPv4 is default
	}
}

// SetType sets the type (TCP or UDP) for a tuple.
func (t *ConnTuple) SetType(connType ConnType) {
	if connType == TCP {
		t.Metadata |= uint32(TCP) // Set the TCP bit
	} else {
		t.Metadata &^= uint32(TCP) // Clear the TCP bit, assuming UDP is default
	}
}

// SetSourceAddress sets the source address.
func (t *ConnTuple) SetSourceAddress(addr util.Address) {
	if t.Family() == IPv4 {
		ip := addr.As4()
		t.Saddr_l = uint64(ip[0]) | uint64(ip[1])<<8 | uint64(ip[2])<<16 | uint64(ip[3])<<24
	} else {
		ip := addr.As16()
		t.Saddr_h = binary.LittleEndian.Uint64(ip[:8])
		t.Saddr_l = binary.LittleEndian.Uint64(ip[8:])
	}
}

// SetDestAddress sets the destination address.
func (t *ConnTuple) SetDestAddress(addr util.Address) {
	if t.Family() == IPv4 {
		ip := addr.As4()
		t.Daddr_l = uint64(ip[0]) | uint64(ip[1])<<8 | uint64(ip[2])<<16 | uint64(ip[3])<<24
	} else {
		ip := addr.As16()
		t.Daddr_h = binary.LittleEndian.Uint64(ip[:8])
		t.Daddr_l = binary.LittleEndian.Uint64(ip[8:])
	}
}

func (t ConnTuple) String() string {
	return fmt.Sprintf(
		"[%s%s] [PID: %d] [%s ⇄ %s] (ns: %d)",
		t.Type(),
		t.Family(),
		t.Pid,
		t.SourceEndpoint(),
		t.DestEndpoint(),
		t.Netns,
	)
}

// ConnectionDirection returns the direction of the connection (incoming vs outgoing).
func (cs ConnStats) ConnectionDirection() ConnDirection {
	return ConnDirection(cs.Direction)
}

// IsAssured returns whether the connection has seen traffic in both directions.
func (cs ConnStats) IsAssured() bool {
	return cs.Flags&uint8(Assured) != 0
}

// ToBatch converts a byte slice to a Batch pointer.
func ToBatch(data []byte) *Batch {
	return (*Batch)(unsafe.Pointer(&data[0]))
}
