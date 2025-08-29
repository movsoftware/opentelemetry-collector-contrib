// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package iperfreceiver

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/scraper/scraperhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/iperfreceiver/internal/metadata"
)

func TestType(t *testing.T) {
	factory := NewFactory()
	assert.Equal(t, metadata.Type, factory.Type())
}

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg)
	
	iperfCfg, ok := cfg.(*Config)
	require.True(t, ok)
	
	assert.Equal(t, "client", iperfCfg.Mode)
	assert.Equal(t, 5201, iperfCfg.ServerPort)
	assert.Equal(t, 60*time.Second, iperfCfg.ControllerConfig.CollectionInterval)
	assert.Empty(t, iperfCfg.Targets)
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))
}

func TestCreateMetricsReceiver(t *testing.T) {
	tests := []struct {
		name    string
		cfg     component.Config
		wantErr bool
	}{
		{
			name: "valid client config",
			cfg: &Config{
				ControllerConfig:     scraperhelper.NewDefaultControllerConfig(),
				MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
				Mode:                 "client",
				Targets: []TargetConfig{
					{
						Host:     "localhost",
						Port:     5201,
						Duration: 10 * time.Second,
						Streams:  1,
						Protocol: "tcp",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid server config",
			cfg: &Config{
				ControllerConfig:     scraperhelper.NewDefaultControllerConfig(),
				MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
				Mode:                 "server",
				ServerPort:           5201,
			},
			wantErr: false,
		},
		{
			name:    "invalid config type",
			cfg:     &struct{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewFactory()
			ctx := context.Background()
			params := receivertest.NewNopSettings()
			consumer := consumertest.NewNop()

			receiver, err := factory.CreateMetrics(ctx, params, tt.cfg, consumer)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, receiver)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, receiver)
			}
		})
	}
}

func TestFactoryMetricsReceiverCapabilities(t *testing.T) {
	factory := NewFactory()
	assert.Equal(t, metadata.MetricsStability, factory.MetricsStability())
}