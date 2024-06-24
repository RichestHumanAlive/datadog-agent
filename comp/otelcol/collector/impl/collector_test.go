// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build otlp

// Package collectorimpl provides the implementation of the collector component for OTel Agent
package collectorimpl

import (
	"os"
	"path/filepath"
	"testing"

	compdef "github.com/DataDog/datadog-agent/comp/def"
	collectorcontribimpl "github.com/DataDog/datadog-agent/comp/otelcol/collector-contrib/impl"
	converterimpl "github.com/DataDog/datadog-agent/comp/otelcol/converter/impl"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/confmap"
	"gopkg.in/yaml.v3"
)

type lifecycle struct{}

func (l *lifecycle) Append(h compdef.Hook) {
	return
}

func uriFromFile(filename string) []string {
	return []string{filepath.Join("testdata", filename)}
}

func TestGetConfDump(t *testing.T) {
	provider, err := converterimpl.NewConverter()
	assert.NoError(t, err)

	reqs := Requires{
		CollectorContrib: collectorcontribimpl.NewComponent(),
		URIs:             uriFromFile("simple-dd/config.yaml"),
		Provider:         provider,
		Lc:               &lifecycle{},
	}
	provides, err := NewComponent(reqs)
	assert.NoError(t, err)

	t.Run("provided", func(t *testing.T) {
		// string
		actualString, _ := provides.Comp.GetProvidedConfAsString()
		actualStringMap, err := yamlBytesToMap([]byte(actualString))
		assert.NoError(t, err)

		expectedBytes, err := os.ReadFile(filepath.Join("testdata", "simple-dd", "config-provided-result.yaml"))
		assert.NoError(t, err)
		expectedMap, err := yamlBytesToMap(expectedBytes)
		assert.NoError(t, err)

		assert.Equal(t, expectedMap, actualStringMap)
	})

	t.Run("enhanced", func(t *testing.T) {
		// string
		actualString, _ := provides.Comp.GetEnhancedConfAsString()
		actualStringMap, err := yamlBytesToMap([]byte(actualString))
		assert.NoError(t, err)

		expectedBytes, err := os.ReadFile(filepath.Join("testdata", "simple-dd", "config-enhanced-result.yaml"))
		assert.NoError(t, err)
		expectedMap, err := yamlBytesToMap(expectedBytes)
		assert.NoError(t, err)

		assert.Equal(t, expectedMap, actualStringMap)
	})
}

func yamlBytesToMap(bytesConfig []byte) (map[string]any, error) {
	var configMap = map[string]interface{}{}
	err := yaml.Unmarshal(bytesConfig, configMap)
	if err != nil {
		return nil, err
	}
	return configMap, nil
}
