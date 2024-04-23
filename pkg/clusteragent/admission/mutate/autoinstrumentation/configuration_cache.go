// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver

// Package autoinstrumentation implements the webhook that injects APM libraries
// into pods
package autoinstrumentation

import (
	"fmt"
	"sync"

	"github.com/DataDog/datadog-agent/pkg/remoteconfig/state"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

type instrumentationConfiguration struct {
	enabled            bool
	enabledNamespaces  []string
	disabledNamespaces []string
}

type enablementConfig struct {
	configID          string
	rcVersion         int
	rcAction          string
	env               *string
	enabled           *bool
	enabledNamespaces *[]string
}

type instrumentationConfigurationCache struct {
	localConfiguration           *instrumentationConfiguration
	currentConfiguration         *instrumentationConfiguration
	configurationUpdatesQueue    chan Request
	configurationUpdatesResponse chan Response
	clusterName                  string
	namespaceToConfigIdMap       map[string]string // maps the namespace with enabled instrumentation to Remote Enablement rule

	mu                  sync.RWMutex
	lastAppliedRevision int64
	orderedRevisions    []int64
	enabledRevisions    map[int64]enablementConfig
}

func newInstrumentationConfigurationCache(
	localEnabled *bool,
	localEnabledNamespaces *[]string,
	localDisabledNamespaces *[]string,
	clusterName string,
) *instrumentationConfigurationCache {
	localConfig := newInstrumentationConfiguration(localEnabled, localEnabledNamespaces, localDisabledNamespaces)
	currentConfig := newInstrumentationConfiguration(localEnabled, localEnabledNamespaces, localDisabledNamespaces)
	reqChannel := make(chan Request, 10)
	respChannel := make(chan Response, 10)

	nsToRules := make(map[string]string)
	if *localEnabled {
		for _, ns := range *localEnabledNamespaces {
			nsToRules[ns] = "local"
		}
	}

	return &instrumentationConfigurationCache{
		localConfiguration:           localConfig,
		currentConfiguration:         currentConfig,
		configurationUpdatesQueue:    reqChannel,
		configurationUpdatesResponse: respChannel,
		clusterName:                  clusterName,
		namespaceToConfigIdMap:       nsToRules,

		orderedRevisions: make([]int64, 0),
		enabledRevisions: map[int64]enablementConfig{},
	}
}

func (c *instrumentationConfigurationCache) start(stopCh <-chan struct{}) {
	for {
		select {
		case req := <-c.configurationUpdatesQueue:
			// if err := c.updateConfiguration(nil, nil); err != nil {
			// 	log.Error(err.Error())
			// }
			resp := c.update(req)
			c.configurationUpdatesResponse <- resp
		case <-stopCh:
			log.Info("Shutting down patcher")
			return
		}
	}
}

func (c *instrumentationConfigurationCache) update(req Request) Response {

	k8sClusterTargets := req.K8sTargetV2.ClusterTargets
	var resp Response

	switch req.Action {
	case EnableConfig:
		for _, target := range k8sClusterTargets {
			clusterName := target.ClusterName

			if c.clusterName == clusterName {
				log.Infof("Current configuration: %v, %v, %v",
					c.currentConfiguration.enabled, c.currentConfiguration.enabledNamespaces, c.currentConfiguration.disabledNamespaces)
				newEnabled := target.Enabled
				newEnabledNamespaces := target.EnabledNamespaces

				c.mu.Lock()
				resp = c.updateConfiguration(*newEnabled, newEnabledNamespaces, req.ID, int(req.RcVersion))
				log.Infof("Updated configuration: %v, %v, %v",
					c.currentConfiguration.enabled, c.currentConfiguration.enabledNamespaces, c.currentConfiguration.disabledNamespaces)

				c.orderedRevisions = append(c.orderedRevisions, req.Revision)
				c.enabledRevisions[req.Revision] = enablementConfig{
					configID:          req.ID,
					rcVersion:         int(req.RcVersion),
					rcAction:          string(req.Action),
					env:               req.LibConfig.Env,
					enabled:           target.Enabled,
					enabledNamespaces: target.EnabledNamespaces,
				}
				c.mu.Unlock()
			}
		}
	default:
		log.Errorf("unknown action %q", req.Action)
	}

	return resp
}

func (c *instrumentationConfigurationCache) readConfiguration() *instrumentationConfiguration {
	return c.currentConfiguration
}

func (c *instrumentationConfigurationCache) readLocalConfiguration() *instrumentationConfiguration {
	return c.localConfiguration
}

func (c *instrumentationConfigurationCache) delete(rcConfigID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, rev := range c.orderedRevisions {
		confId, ok := c.enabledRevisions[rev]
		if !ok {
			log.Error("Revision was not found")
		}
		if confId.configID == rcConfigID {
			delete(c.enabledRevisions, rev)
			c.orderedRevisions = append(c.orderedRevisions[:i], c.orderedRevisions[i+1:]...)
			break
		}
	}
	c.resetConfiguration()
	return nil
}

func (c *instrumentationConfigurationCache) resetConfiguration() {
	c.currentConfiguration = c.localConfiguration
	for _, rev := range c.orderedRevisions {
		conf := c.enabledRevisions[rev]
		c.updateConfiguration(*conf.enabled, conf.enabledNamespaces, conf.configID, conf.rcVersion)
	}
}

func (c *instrumentationConfigurationCache) updateConfiguration(enabled bool, enabledNamespaces *[]string, rcID string, rcVersion int) Response {
	log.Debugf("Updating current APM Instrumentation configuration")
	log.Debugf("Old APM Instrumentation configuration [enabled=%t enabledNamespaces=%v disabledNamespaces=%v]",
		c.currentConfiguration.enabled,
		c.currentConfiguration.enabledNamespaces,
		c.currentConfiguration.disabledNamespaces,
	)
	resp := Response{
		ID:        rcID,
		RcVersion: uint64(rcVersion),
		Status:    state.ApplyStatus{State: state.ApplyStateAcknowledged},
	}

	if c.currentConfiguration.enabled && !enabled {
		log.Errorf("Remote Enablement: disabling APM instrumentation is not supported")
		resp.Status.State = state.ApplyStateError
		resp.Status.Error = "Disabling APM instrumentation is not supported"
		return resp
	}

	if c.currentConfiguration.enabled {
		if len(c.currentConfiguration.enabledNamespaces) == 0 &&
			len(c.currentConfiguration.disabledNamespaces) == 0 &&
			enabledNamespaces != nil && len(*enabledNamespaces) > 0 {
			// current configuration - APM Instrumentation enabled in the whole cluster
			// remote configuration - APM Instrumentation enabled in specific namespaces
			// result - error
			log.Errorf("Remote Enablement: APM Insrtumentation is enabled in the whole cluster via agent configuration")
			resp.Status.State = state.ApplyStateError
			resp.Status.Error = "Remote Enablement: APM Insrtumentation is enabled in the whole cluster via agent configuration"
			return resp
		} else if len(c.currentConfiguration.enabledNamespaces) > 0 && (enabledNamespaces == nil || len(*enabledNamespaces) == 0) {
			// current configuration - APM Instrumentation enabled in specific namespaces
			// remote configuration - APM Instrumentation enabled in the whole cluster
			// result - enable APM instrumentation in the whole cluster
			c.currentConfiguration.enabledNamespaces = []string{}
			c.namespaceToConfigIdMap["cluster"] = fmt.Sprintf("%s-%d", rcID, rcVersion)
		} else if len(c.currentConfiguration.enabledNamespaces) > 0 {
			// current configuration - APM Instrumentation enabled in specific namespaces
			// remote configuration - APM Instrumentation enabled in specific namespaces
			// result - enable APM instrumentation in namespaces specified by current + remote configuration
			alreadyEnabledNamespaces := []string{}
			for _, ns := range *enabledNamespaces {
				if _, ok := c.namespaceToConfigIdMap[ns]; ok {
					alreadyEnabledNamespaces = append(alreadyEnabledNamespaces, ns)
				} else {
					c.currentConfiguration.enabledNamespaces = append(c.currentConfiguration.enabledNamespaces, ns)
					c.namespaceToConfigIdMap[ns] = fmt.Sprintf("%s-%d", rcID, rcVersion)
				}
			}
			if len(alreadyEnabledNamespaces) > 0 {
				resp.Status.State = state.ApplyStateError
				resp.Status.Error = fmt.Sprintf("Remote Enablement: failing instrumentation because APM is already enabled in namespaces %v", alreadyEnabledNamespaces)
				return resp
			}
		} else if len(c.currentConfiguration.disabledNamespaces) > 0 && (enabledNamespaces == nil || len(*enabledNamespaces) == 0) {
			// current configuration - APM Instrumentation disabled in specific namespaces
			// remote configuration - APM Instrumentation enabled in the whole cluster
			// result - enable APM instrumentation in the whole cluster
			c.currentConfiguration.disabledNamespaces = []string{}
			c.namespaceToConfigIdMap["cluster"] = fmt.Sprintf("%s-%d", rcID, rcVersion)
		} else if len(c.currentConfiguration.disabledNamespaces) > 0 {
			// current configuration - APM Instrumentation disabled in specific namespaces
			// remote configuration - APM Instrumentation enabled in specific namespaces
			// result - enable APM instrumentation in disabled namespaces
			disabledNsMap := make(map[string]struct{})
			for _, ns := range c.currentConfiguration.disabledNamespaces {
				disabledNsMap[ns] = struct{}{}
			}
			for _, ns := range *enabledNamespaces {
				if _, ok := disabledNsMap[ns]; ok {
					delete(disabledNsMap, ns)
				}
				c.namespaceToConfigIdMap[ns] = fmt.Sprintf("%s-%d", rcID, rcVersion)
			}
			disabledNs := make([]string, 0, len(disabledNsMap))
			for k := range disabledNsMap {
				disabledNs = append(disabledNs, k)
			}
			c.currentConfiguration.disabledNamespaces = disabledNs
		}
	} else {
		if enabled {
			c.currentConfiguration.enabled = enabled
			if enabledNamespaces != nil && len(*enabledNamespaces) > 0 {
				for _, ns := range *enabledNamespaces {
					c.currentConfiguration.enabledNamespaces = append(c.currentConfiguration.enabledNamespaces, ns)
					c.namespaceToConfigIdMap[ns] = fmt.Sprintf("%s-%d", rcID, rcVersion)
				}
			} else {
				c.namespaceToConfigIdMap["cluster"] = fmt.Sprintf("%s-%d", rcID, rcVersion)
			}
		} else {
			log.Errorf("Noop: APM Instrumentation is off")
			resp.Status.State = state.ApplyStateError
			resp.Status.Error = "Noop: APM Instrumentation is off"
			return resp
		}
	}

	log.Debugf("New APM Instrumentation configuration [enabled=%t enabledNamespaces=%v disabledNamespaces=%v]",
		c.currentConfiguration.enabled,
		c.currentConfiguration.enabledNamespaces,
		c.currentConfiguration.disabledNamespaces,
	)
	return resp
}

func newInstrumentationConfiguration(
	enabled *bool,
	enabledNamespaces *[]string,
	disabledNamespaces *[]string,
) *instrumentationConfiguration {
	config := instrumentationConfiguration{
		enabled:            false,
		enabledNamespaces:  []string{},
		disabledNamespaces: []string{},
	}
	if enabled != nil {
		config.enabled = *enabled
	}
	if enabledNamespaces != nil {
		config.enabledNamespaces = *enabledNamespaces
	}
	if disabledNamespaces != nil {
		config.disabledNamespaces = *disabledNamespaces
	}

	return &config
}
