// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux_bpf
// +build linux_bpf

package kafka

import (
	"github.com/DataDog/datadog-agent/pkg/util/atomicstats"
	"time"

	"go.uber.org/atomic"

	"github.com/DataDog/datadog-agent/pkg/util/log"
)

type telemetry struct {
	then    *atomic.Int64
	elapsed *atomic.Int64

	totalHits    *atomic.Int64
	misses       *atomic.Int64 // this happens when we can't cope with the rate of events
	dropped      *atomic.Int64 // this happens when httpStatKeeper reaches capacity
	rejected     *atomic.Int64 // this happens when an user-defined reject-filter matches a request
	malformed    *atomic.Int64 // this happens when the request doesn't have the expected format
	aggregations *atomic.Int64
}

func newTelemetry() (*telemetry, error) {
	t := &telemetry{
		then:         atomic.NewInt64(time.Now().Unix()),
		elapsed:      atomic.NewInt64(0),
		totalHits:    atomic.NewInt64(0),
		misses:       atomic.NewInt64(0),
		dropped:      atomic.NewInt64(0),
		rejected:     atomic.NewInt64(0),
		malformed:    atomic.NewInt64(0),
		aggregations: atomic.NewInt64(0),
	}

	return t, nil
}

func (t *telemetry) aggregate(transactions []kafkaTX, err error) {
	t.totalHits.Add(int64(len(transactions)))

	if err == errLostBatch {
		t.misses.Add(int64(len(transactions)))
	}
}

func (t *telemetry) reset() telemetry {
	now := time.Now().Unix()
	then := t.then.Swap(now)

	delta, _ := newTelemetry()
	delta.misses.Store(t.misses.Swap(0))
	delta.dropped.Store(t.dropped.Swap(0))
	delta.rejected.Store(t.rejected.Swap(0))
	delta.malformed.Store(t.malformed.Swap(0))
	delta.aggregations.Store(t.aggregations.Swap(0))
	delta.elapsed.Store(now - then)

	log.Debugf(
		"http stats summary: requests_processed=%d(%.2f/s) requests_missed=%d(%.2f/s) requests_dropped=%d(%.2f/s) requests_rejected=%d(%.2f/s) requests_malformed=%d(%.2f/s) aggregations=%d",
		delta.totalHits.Load(),
		float64(delta.totalHits.Load())/float64(delta.elapsed.Load()),
		delta.misses.Load(),
		float64(delta.misses.Load())/float64(delta.elapsed.Load()),
		delta.dropped.Load(),
		float64(delta.dropped.Load())/float64(delta.elapsed.Load()),
		delta.rejected.Load(),
		float64(delta.rejected.Load())/float64(delta.elapsed.Load()),
		delta.malformed.Load(),
		float64(delta.malformed.Load())/float64(delta.elapsed.Load()),
		delta.aggregations.Load(),
	)

	return *delta
}

func (t *telemetry) report() map[string]interface{} {
	return atomicstats.Report(t)
}
