// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux

package servicediscovery

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/procfs"

	"github.com/DataDog/datadog-agent/pkg/collector/corechecks/servicediscovery/model"
	ddconfig "github.com/DataDog/datadog-agent/pkg/config"
	processnet "github.com/DataDog/datadog-agent/pkg/process/net"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

//go:generate mockgen -source=$GOFILE -package=$GOPACKAGE -destination=impl_linux_mock.go

func init() {
	newOSImpl = newLinuxImpl
}

var ignoreCfgLinux = []string{
	"sshd",
	"dhclient",
	"systemd",
	"systemd-resolved",
	"systemd-networkd",
	"datadog-agent",
	"livenessprobe",
	"docker-proxy", // remove when we have docker support in place
}

type linuxImpl struct {
	procfs            procFS
	getSysProbeClient func() (systemProbeClient, error)
	time              timer
	bootTime          uint64

	serviceDetector *serviceDetector
	ignoreCfg       map[string]bool

	ignoreProcs       map[int]bool
	aliveServices     map[int]*serviceInfo
	potentialServices map[int]*serviceInfo
}

func newLinuxImpl(ignoreCfg map[string]bool) (osImpl, error) {
	for _, i := range ignoreCfgLinux {
		ignoreCfg[i] = true
	}
	pfs, err := procfs.NewDefaultFS()
	if err != nil {
		return nil, err
	}
	stat, err := pfs.Stat()
	if err != nil {
		return nil, err
	}
	return &linuxImpl{
		procfs:            wProcFS{pfs},
		bootTime:          stat.BootTime,
		getSysProbeClient: getSysProbeClient,
		time:              realTime{},
		serviceDetector:   newServiceDetector(),
		ignoreCfg:         ignoreCfg,
		ignoreProcs:       make(map[int]bool),
		aliveServices:     make(map[int]*serviceInfo),
		potentialServices: make(map[int]*serviceInfo),
	}, nil
}

func (li *linuxImpl) DiscoverServices() (*discoveredServices, error) {
	procs, err := li.aliveProcs()
	if err != nil {
		return nil, errWithCode{
			err:  err,
			code: errorCodeProcfs,
			svc:  nil,
		}
	}

	sysProbe, err := li.getSysProbeClient()
	if err != nil {
		return nil, errWithCode{
			err:  err,
			code: errorCodeSystemProbeConn,
			svc:  nil,
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ports, err := sysProbe.GetDiscoveryOpenPorts(ctx)
	if err != nil {
		return nil, errWithCode{
			err:  err,
			code: errorCodeSystemProbeOpenPorts,
			svc:  nil,
		}
	}

	portsByPID := map[int][]*model.Port{}
	for _, p := range ports.Ports {
		if p.PID == 0 {
			log.Debugf("port:%s:%d | could not find port pid, skipping (insufficient permissions?)", p.Proto, p.Port)
			continue
		}
		portsByPID[p.PID] = append(portsByPID[p.PID], p)
	}

	events := serviceEvents{}

	now := li.time.Now()

	// potentialServices contains processes that we scanned in the previous iteration and had open ports.
	// we check if they are still alive in this iteration, and if so, we send a start-service telemetry event.
	for pid, svc := range li.potentialServices {
		if _, ok := procs[pid]; ok {
			svc.LastHeartbeat = now
			li.aliveServices[pid] = svc
			events.start = append(events.start, *svc)
		}
	}
	clear(li.potentialServices)

	// check open ports - these will be potential new services if they are still alive in the next iteration.
	for pid := range portsByPID {
		if li.ignoreProcs[pid] {
			continue
		}
		if _, ok := li.aliveServices[pid]; !ok {
			log.Debugf("[pid: %d] found new process with open ports", pid)

			p, ok := procs[pid]
			if !ok {
				log.Debugf("[pid: %d] process with open ports was not found in alive procs", pid)
				continue
			}

			svc, err := li.getServiceInfo(p, sysProbe, portsByPID)
			if err != nil {
				telemetryFromError(errWithCode{
					err:  err,
					code: errorCodeProcfs,
					svc:  nil,
				})
				log.Errorf("[pid: %d] failed to get process info: %v", pid, err)
				li.ignoreProcs[pid] = true
				continue
			}
			if li.ignoreCfg[svc.meta.Name] {
				log.Debugf("[pid: %d] process ignored from config: %s", pid, svc.meta.Name)
				li.ignoreProcs[pid] = true
				continue
			}
			log.Debugf("[pid: %d] adding process to potential: %s", pid, svc.meta.Name)
			li.potentialServices[pid] = svc
		}
	}

	// check if services previously marked as alive still are.
	for pid, svc := range li.aliveServices {
		if _, ok := procs[pid]; !ok {
			delete(li.aliveServices, pid)
			events.stop = append(events.stop, *svc)
		} else if now.Sub(svc.LastHeartbeat).Truncate(time.Minute) >= heartbeatTime {
			svc.LastHeartbeat = now
			events.heartbeat = append(events.heartbeat, *svc)
		}
	}

	// check if services previously marked as ignore are still alive.
	for pid := range li.ignoreProcs {
		if _, ok := procs[pid]; !ok {
			delete(li.ignoreProcs, pid)
		}
	}

	return &discoveredServices{
		aliveProcsCount: len(procs),
		openPorts:       ports.Ports,
		ignoreProcs:     li.ignoreProcs,
		potentials:      li.potentialServices,
		runningServices: li.aliveServices,
		events:          events,
	}, nil
}

func (li *linuxImpl) aliveProcs() (map[int]proc, error) {
	procs, err := li.procfs.AllProcs()
	if err != nil {
		return nil, err
	}
	procMap := map[int]proc{}
	for _, v := range procs {
		procMap[v.PID()] = v
	}
	return procMap, nil
}

func (li *linuxImpl) getServiceInfo(p proc, sysProbe systemProbeClient, openPorts map[int][]*model.Port) (*serviceInfo, error) {
	cmdline, err := p.CmdLine()
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/{pid}/cmdline: %w", err)
	}

	stat, err := p.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/{pid}/stat: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	procResp, err := sysProbe.GetDiscoveryProc(ctx, p.PID())
	if err != nil {
		return nil, errWithCode{
			err:  fmt.Errorf("failed to get proc from system-probe: %w", err),
			code: errorCodeSystemProbeGetProc,
			svc:  nil,
		}
	}

	var ports []int
	for _, port := range openPorts[p.PID()] {
		ports = append(ports, int(port.Port))
	}

	// if the process name is docker-proxy, we should talk to docker to get the process command line and env vars
	// have to see how far this can go but not for the initial release

	// for now, docker-proxy is going on the ignore list

	// calculate the start time
	// divide Starttime by 100 to go from clicks since boot to seconds since boot
	startTimeSecs := li.bootTime + (stat.Starttime / 100)

	pInfo := processInfo{
		PID:     p.PID(),
		CmdLine: cmdline,
		Env:     procResp.Proc.Environ,
		Cwd:     procResp.Proc.CWD,
		Stat: procStat{
			StartTime: startTimeSecs,
		},
		Ports: ports,
	}

	meta := li.serviceDetector.Detect(pInfo)

	return &serviceInfo{
		process:       pInfo,
		meta:          meta,
		LastHeartbeat: li.time.Now(),
	}, nil
}

type proc interface {
	PID() int
	CmdLine() ([]string, error)
	Stat() (procfs.ProcStat, error)
}

type wProc struct {
	procfs.Proc
}

func (w wProc) PID() int {
	return w.Proc.PID
}

type procFS interface {
	AllProcs() ([]proc, error)
}

type wProcFS struct {
	procfs.FS
}

func (w wProcFS) AllProcs() ([]proc, error) {
	procs, err := w.FS.AllProcs()
	if err != nil {
		return nil, err
	}
	var res []proc
	for _, p := range procs {
		res = append(res, wProc{p})
	}
	return res, nil
}

type systemProbeClient interface {
	GetDiscoveryOpenPorts(ctx context.Context) (*model.OpenPortsResponse, error)
	GetDiscoveryProc(ctx context.Context, pid int) (*model.GetProcResponse, error)
}

func getSysProbeClient() (systemProbeClient, error) {
	return processnet.GetRemoteSystemProbeUtil(
		ddconfig.SystemProbe.GetString("system_probe_config.sysprobe_socket"),
	)
}
