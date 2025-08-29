// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package iperfreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/iperfreceiver"

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scraperhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/iperfreceiver/internal/metadata"
)

var errConfigNotIperf = errors.New("config was not an iperf receiver config")

// NewFactory creates a new iperf receiver factory
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability),
	)
}

// createDefaultConfig creates the default configuration for the receiver
func createDefaultConfig() component.Config {
	cfg := scraperhelper.NewDefaultControllerConfig()
	cfg.CollectionInterval = 60 * time.Second

	return &Config{
		ControllerConfig:     cfg,
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		Mode:                 "client",
		ServerPort:           5201, // Default iperf3 port
		Targets:              []TargetConfig{},
	}
}

// createMetricsReceiver creates a metrics receiver based on the provided config
func createMetricsReceiver(
	_ context.Context,
	params receiver.Settings,
	rConf component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	cfg, ok := rConf.(*Config)
	if !ok {
		return nil, errConfigNotIperf
	}

	iperfScraper := newScraper(cfg, params)
	s, err := scraper.NewMetrics(
		iperfScraper.scrape,
		scraper.WithStart(iperfScraper.start),
		scraper.WithShutdown(iperfScraper.shutdown),
	)
	if err != nil {
		return nil, err
	}

	return scraperhelper.NewMetricsController(
		&cfg.ControllerConfig,
		params,
		consumer,
		scraperhelper.AddScraper(metadata.Type, s),
	)
}