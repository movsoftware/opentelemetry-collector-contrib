// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ztracereceiver

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))

	zCfg := cfg.(*Config)
	assert.Equal(t, "0.0.0.0:8888", zCfg.Endpoint)
	assert.Equal(t, 60*time.Second, zCfg.CollectionInterval)
	assert.Equal(t, 10*time.Second, zCfg.Timeout)
	assert.Equal(t, "udp", zCfg.Protocol)
	assert.Equal(t, 30, zCfg.MaxHops)
	assert.Equal(t, 56, zCfg.PacketSize)
	assert.Equal(t, 3, zCfg.Retries)
	assert.True(t, zCfg.EnableGeolocation)
	assert.True(t, zCfg.EnableASNLookup)
}

func TestCreateMetricsReceiver(t *testing.T) {
	cfg := &Config{
		ServerConfig: confighttp.ServerConfig{
			Endpoint: "localhost:8080",
		},
		Targets: []TargetConfig{
			{
				Endpoint: "example.com",
				Port:     80,
			},
		},
		CollectionInterval: 30 * time.Second,
		Timeout:            10 * time.Second,
		Protocol:           "udp",
		MaxHops:            30,
		PacketSize:         56,
		Retries:            3,
	}

	factory := NewFactory()
	set := receivertest.NewNopSettings()
	mReceiver, err := factory.CreateMetrics(context.Background(), set, cfg, consumertest.NewNop())
	assert.NoError(t, err)
	assert.NotNil(t, mReceiver)
}

func TestCreateTracesReceiver(t *testing.T) {
	cfg := &Config{
		ServerConfig: confighttp.ServerConfig{
			Endpoint: "localhost:8080",
		},
		Targets: []TargetConfig{
			{
				Endpoint: "example.com",
				Port:     80,
			},
		},
		CollectionInterval: 30 * time.Second,
		Timeout:            10 * time.Second,
		Protocol:           "udp",
		MaxHops:            30,
		PacketSize:         56,
		Retries:            3,
	}

	factory := NewFactory()
	set := receivertest.NewNopSettings()
	tReceiver, err := factory.CreateTraces(context.Background(), set, cfg, consumertest.NewNop())
	assert.NoError(t, err)
	assert.NotNil(t, tReceiver)
}

func TestCreateReceiverWithInvalidConfig(t *testing.T) {
	cfg := &Config{
		ServerConfig: confighttp.ServerConfig{
			Endpoint: "localhost:8080",
		},
		// Missing targets
		CollectionInterval: 30 * time.Second,
		Timeout:            10 * time.Second,
		Protocol:           "udp",
		MaxHops:            30,
		PacketSize:         56,
		Retries:            3,
	}

	// Validate should fail
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one target must be specified")
}