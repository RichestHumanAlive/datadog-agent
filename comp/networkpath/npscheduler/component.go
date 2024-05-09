// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024-present Datadog, Inc.

// Package npscheduler used to manage network paths
package npscheduler

import (
	model "github.com/DataDog/agent-payload/v5/process"
)

// team: network-device-monitoring

// Component is the component type.
type Component interface {
	ScheduleConns(conns []*model.Connection)
	Enabled() bool
}
