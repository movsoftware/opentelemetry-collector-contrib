// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package iperfreceiver

import (
	"context"
	"testing"
	"time"

	iperf "github.com/BGrewell/go-iperf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/scraper/scraperhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/iperfreceiver/internal/metadata"
)

func TestNewScraper(t *testing.T) {
	cfg := &Config{
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
	}

	settings := receivertest.NewNopSettings()
	scraper := newScraper(cfg, settings)

	assert.NotNil(t, scraper)
	assert.Equal(t, cfg, scraper.cfg)
	assert.NotNil(t, scraper.logger)
}

func TestScraperStart(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "client mode",
			cfg: &Config{
				ControllerConfig:     scraperhelper.NewDefaultControllerConfig(),
				MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
				Mode:                 "client",
				Targets: []TargetConfig{
					{
						Host:     "localhost",
						Port:     5201,
						Duration: 10 * time.Second,
					},
				},
			},
			wantErr: false,
		},
		// Note: Server mode test skipped as it would actually start a server
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := receivertest.NewNopSettings()
			scraper := newScraper(tt.cfg, settings)

			ctx := context.Background()
			host := componenttest.NewNopHost()
			err := scraper.start(ctx, host)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, scraper.mb)
			}
		})
	}
}

func TestRecordMetrics(t *testing.T) {
	cfg := &Config{
		ControllerConfig:     scraperhelper.NewDefaultControllerConfig(),
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		Mode:                 "client",
	}

	settings := receivertest.NewNopSettings()
	scraper := newScraper(cfg, settings)

	// Initialize metrics builder
	ctx := context.Background()
	host := componenttest.NewNopHost()
	err := scraper.start(ctx, host)
	require.NoError(t, err)

	// Create a mock report
	report := &iperf.Report{
		End: &iperf.End{
			SumSent: &iperf.Sum{
				Bytes:         1024000,
				BitsPerSecond: 8192000,
				Retransmits:   5,
			},
			SumReceived: &iperf.Sum{
				Bytes:         1024000,
				BitsPerSecond: 8192000,
				Jitter:        0.5,
				LostPercent:   0.1,
			},
			CPUUtilizationPercent: &iperf.CPUUtilizationPercent{
				HostTotal:   25.5,
				RemoteTotal: 30.2,
			},
		},
	}

	target := TargetConfig{
		Host:     "localhost",
		Port:     5201,
		Protocol: "tcp",
		Streams:  4,
	}

	timestamp := pcommon.NewTimestampFromTime(time.Now())
	testDuration := 10.5

	// Record metrics
	scraper.recordMetrics(report, target, timestamp, testDuration)

	// Get metrics
	metrics := scraper.mb.Emit()
	assert.NotNil(t, metrics)

	// Verify metrics were recorded
	assert.Greater(t, metrics.MetricCount(), 0)
	assert.Greater(t, metrics.DataPointCount(), 0)
}

func TestRecordMetricsWithNilReport(t *testing.T) {
	cfg := &Config{
		ControllerConfig:     scraperhelper.NewDefaultControllerConfig(),
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		Mode:                 "client",
	}

	settings := receivertest.NewNopSettings()
	scraper := newScraper(cfg, settings)

	// Initialize metrics builder
	ctx := context.Background()
	host := componenttest.NewNopHost()
	err := scraper.start(ctx, host)
	require.NoError(t, err)

	target := TargetConfig{
		Host:     "localhost",
		Port:     5201,
		Protocol: "tcp",
	}

	timestamp := pcommon.NewTimestampFromTime(time.Now())
	testDuration := 10.5

	// Test with nil End section
	report := &iperf.Report{
		End: nil,
	}

	// Should not panic
	scraper.recordMetrics(report, target, timestamp, testDuration)

	// Test with empty End section
	report = &iperf.Report{
		End: &iperf.End{},
	}

	// Should not panic
	scraper.recordMetrics(report, target, timestamp, testDuration)
}

func TestRecordMetricsUDP(t *testing.T) {
	cfg := &Config{
		ControllerConfig:     scraperhelper.NewDefaultControllerConfig(),
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		Mode:                 "client",
	}

	settings := receivertest.NewNopSettings()
	scraper := newScraper(cfg, settings)

	// Initialize metrics builder
	ctx := context.Background()
	host := componenttest.NewNopHost()
	err := scraper.start(ctx, host)
	require.NoError(t, err)

	// Create a UDP report
	report := &iperf.Report{
		End: &iperf.End{
			SumSent: &iperf.Sum{
				Bytes:         1024000,
				BitsPerSecond: 8192000,
			},
			SumReceived: &iperf.Sum{
				Bytes:         1024000,
				BitsPerSecond: 8192000,
				Jitter:        1.5,
				LostPercent:   0.5,
			},
		},
	}

	target := TargetConfig{
		Host:      "localhost",
		Port:      5201,
		Protocol:  "udp",
		Streams:   1,
		Bandwidth: "10M",
	}

	timestamp := pcommon.NewTimestampFromTime(time.Now())
	testDuration := 10.0

	// Record metrics
	scraper.recordMetrics(report, target, timestamp, testDuration)

	// Get metrics
	metrics := scraper.mb.Emit()
	assert.NotNil(t, metrics)

	// Verify UDP-specific metrics were recorded
	assert.Greater(t, metrics.MetricCount(), 0)
	assert.Greater(t, metrics.DataPointCount(), 0)
}