// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ztracereceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/ztracereceiver"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/ztracereceiver/internal/metadata"
)

// NewFactory creates a factory for ztrace receiver.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability),
		receiver.WithTraces(createTracesReceiver, metadata.TracesStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		ServerConfig: confighttp.ServerConfig{
			Endpoint: "0.0.0.0:8888",
		},
		CollectionInterval: 60 * time.Second,
		Timeout:            10 * time.Second,
		Protocol:           "udp",
		MaxHops:            30,
		PacketSize:         56,
		Retries:            3,
		EnableGeolocation:  true,
		EnableASNLookup:    true,
	}
}

func createMetricsReceiver(
	ctx context.Context,
	params receiver.Settings,
	cfg component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	zCfg := cfg.(*Config)
	r := &ztraceReceiver{
		config:   zCfg,
		settings: params,
		consumer: consumer,
	}
	return r, nil
}

func createTracesReceiver(
	ctx context.Context,
	params receiver.Settings,
	cfg component.Config,
	consumer consumer.Traces,
) (receiver.Traces, error) {
	zCfg := cfg.(*Config)
	r := &ztraceReceiver{
		config:        zCfg,
		settings:      params,
		traceConsumer: consumer,
	}
	return r, nil
}