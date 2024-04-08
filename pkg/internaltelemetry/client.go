// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

// Package internaltelemetry full description in README.md
package internaltelemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	metadatautils "github.com/DataDog/datadog-agent/comp/metadata/host/hostimpl/utils"
	pb "github.com/DataDog/datadog-agent/pkg/proto/pbgo/trace"
	"github.com/DataDog/datadog-agent/pkg/trace/config"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/datadog-agent/pkg/version"
)

const (
	telemetryEndpoint = "/api/v2/apmtelemetry"
)

// Client defines the interface for a telemetry client
type Client interface {
	SendLog(level string, message string)
	SendTraces(traces *pb.Traces)
}

type client struct {
	m         sync.Mutex
	client    httpClient
	endpoints []*config.Endpoint

	// we can pre-calculate the host payload structure at init time
	baseEvent Event
}

// RequestType defines the request type
type RequestType string

const (
	// RequestTypeLogs defines the logs request type
	RequestTypeLogs RequestType = "logs"
	// RequestTypeTraces defines the traces request type
	RequestTypeTraces RequestType = "traces"
)

// Event defines the event object
type Event struct {
	APIVersion  string      `json:"api_version"`
	RequestType RequestType `json:"request_type"`
	TracerTime  int64       `json:"tracer_time"` // unix timestamp (in seconds)
	RuntimeID   string      `json:"runtime_id"`
	SequenceID  uint64      `json:"seq_id"`
	DebugFlag   bool        `json:"debug"`
	Host        Host        `json:"host"`
	Application Application `json:"application"`
	Payload     interface{} `json:"payload"`
}

// Host defines the host object
type Host struct {
	Hostname      string `json:"hostname"`
	OS            string `json:"os"`
	Arch          string `json:"architecture"`
	KernelName    string `json:"kernel_name"`
	KernelRelease string `json:"kernel_release"`
	KernelVersion string `json:"kernel_version"`
}

// Application defines the application object
type Application struct {
	ServiceName     string `json:"service_name"`
	LanguageName    string `json:"language_name"`
	LanguageVersion string `json:"language_version"`
	TracerVersion   string `json:"tracer_version"`
}

type TracePayload struct {
	Traces []pb.Trace `json:"traces"`
}

// LogPayload defines the log payload object
type LogPayload struct {
	Logs []LogMessage `json:"logs"`
}

// LogMessage defines the log message object
type LogMessage struct {
	Message string `json:"message"`
	Level   string `json:"level"`
}

var sequenceID atomic.Uint64

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewClient creates a new telemetry client
func NewClient(httpClient httpClient, endpoints []*config.Endpoint, service string) Client {
	info := metadatautils.GetInformation()
	return &client{
		client:    httpClient,
		endpoints: endpoints,
		baseEvent: Event{
			APIVersion: "v2",
			DebugFlag:  true,
			RuntimeID:  info.HostID,
			Host: Host{
				Hostname:      info.Hostname,
				OS:            info.OS,
				Arch:          info.KernelArch,
				KernelName:    info.Platform,
				KernelRelease: info.PlatformVersion,
				KernelVersion: info.PlatformVersion,
			},
			Application: Application{
				ServiceName:     service,
				LanguageName:    "go",
				LanguageVersion: runtime.Version(),
				TracerVersion:   version.AgentVersion,
			},
		},
	}
}

func (c *client) SendLog(level, message string) {
	c.m.Lock()
	defer c.m.Unlock()
	payload := LogPayload{
		Logs: []LogMessage{{Message: message, Level: level}},
	}
	c.sendPayload(RequestTypeLogs, payload)
}

func (c *client) SendTraces(traces *pb.Traces) {
	c.m.Lock()
	defer c.m.Unlock()
	payload := TracePayload{
		Traces: *traces,
	}
	c.sendPayload(RequestTypeTraces, payload)
}

func (c *client) sendPayload(requestType RequestType, payload interface{}) {
	event := c.baseEvent
	event.RequestType = requestType
	event.SequenceID = sequenceID.Add(1)
	event.TracerTime = time.Now().Unix()

	serializedPayload, err := json.Marshal(payload)
	if err != nil {
		log.Errorf("failed to serialize payload: %v", err)
		return
	}
	for _, endpoint := range c.endpoints {
		url := fmt.Sprintf("%s%s", endpoint.Host, telemetryEndpoint)
		req, err := http.NewRequest("POST", url, bytes.NewReader(serializedPayload))
		if err != nil {
			log.Errorf("failed to create request for endpoint %s: %v", url, err)
			continue
		}
		req.Header.Add("DD-Api-Key", endpoint.APIKey)
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("DD-Telemetry-api-version", "v2")
		req.Header.Add("DD-Telemetry-request-type", "v2") // todo
		req.Header.Add("dd-client-library-language", "agent")
		req.Header.Add("dd-client-library-version", "1.5") // should this be agent version?
		req.Header.Add("datadog-container-id", "1")        // todo is this necessary?  likely not a container
		resp, err := c.client.Do(req)
		if err != nil {
			log.Errorf("failed to send payload to endpoint %s: %v", url, err)
			continue
		}
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body)
	}
}
