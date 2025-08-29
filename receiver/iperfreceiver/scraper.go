// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package iperfreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/iperfreceiver"

import (
	"context"
	"fmt"
	"sync"
	"time"

	iperf "github.com/BGrewell/go-iperf"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/iperfreceiver/internal/metadata"
)

type scraper struct {
	cfg      *Config
	logger   *zap.Logger
	settings receiver.Settings
	mb       *metadata.MetricsBuilder
	server   *iperf.Server
	mu       sync.Mutex
}

func newScraper(cfg *Config, settings receiver.Settings) *scraper {
	return &scraper{
		cfg:      cfg,
		logger:   settings.Logger,
		settings: settings,
	}
}

func (s *scraper) start(ctx context.Context, host component.Host) error {
	s.mb = metadata.NewMetricsBuilder(s.cfg.MetricsBuilderConfig, s.settings)

	// If running in server mode, start the iperf3 server
	if s.cfg.Mode == "server" {
		s.server = iperf.NewServer()
		s.server.SetPort(s.cfg.ServerPort)
		s.server.SetJSON(true)

		s.logger.Info("Starting iperf3 server", zap.Int("port", s.cfg.ServerPort))
		
		go func() {
			if err := s.server.Start(); err != nil {
				s.logger.Error("Failed to start iperf3 server", zap.Error(err))
			}
		}()
		
		// Give the server time to start
		time.Sleep(2 * time.Second)
	}

	return nil
}

func (s *scraper) shutdown(ctx context.Context) error {
	if s.server != nil {
		s.logger.Info("Stopping iperf3 server")
		if err := s.server.Stop(); err != nil {
			s.logger.Error("Failed to stop iperf3 server", zap.Error(err))
			return err
		}
	}
	return nil
}

func (s *scraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := pcommon.NewTimestampFromTime(time.Now())

	// Server mode: collect server-side metrics if available
	if s.cfg.Mode == "server" {
		// In server mode, we would collect metrics from the running server
		// This would require implementing a way to get metrics from the server
		s.logger.Debug("Server mode metrics collection not fully implemented")
		return s.mb.Emit(), nil
	}

	// Client mode: run tests against configured targets
	var wg sync.WaitGroup
	for _, target := range s.cfg.Targets {
		wg.Add(1)
		go func(t TargetConfig) {
			defer wg.Done()
			s.runClientTest(ctx, t, now)
		}(target)
	}
	wg.Wait()

	return s.mb.Emit(), nil
}

func (s *scraper) runClientTest(ctx context.Context, target TargetConfig, timestamp pcommon.Timestamp) {
	client := iperf.NewClient(target.Host)
	client.SetPort(target.Port)
	client.SetJSON(true)
	client.SetStreams(target.Streams)
	client.SetTimeSec(int(target.Duration.Seconds()))
	client.SetOmitSec(target.OmitSec)
	client.SetReverse(target.Reverse)

	// Set protocol-specific options
	switch target.Protocol {
	case "udp":
		client.SetProto(iperf.PROTO_UDP)
		if target.Bandwidth != "" {
			client.SetBandwidth(target.Bandwidth)
		}
	case "sctp":
		client.SetProto(iperf.PROTO_SCTP)
	default:
		client.SetProto(iperf.PROTO_TCP)
		if target.ZeroCopy {
			client.SetZerocopy(true)
		}
		if target.NoDelay {
			client.SetNoDelay(true)
		}
		if target.MSS > 0 {
			client.SetMSS(target.MSS)
		}
		if target.Window != "" {
			client.SetWindow(target.Window) 
		}
		if target.Congestion != "" {
			client.SetCongestionAlgorithm(target.Congestion)
		}
	}

	// Run the test
	startTime := time.Now()
	err := client.Start()
	testDuration := time.Since(startTime).Seconds()

	if err != nil {
		s.logger.Error("Failed to run iperf test", 
			zap.String("host", target.Host),
			zap.Int("port", target.Port),
			zap.Error(err))
		
		// Record error metric
		s.mb.RecordIperfTestErrorDataPoint(timestamp, 1, err.Error())
		return
	}

	// Get test report
	report := client.Report()
	if report == nil {
		s.logger.Error("Failed to get iperf report",
			zap.String("host", target.Host),
			zap.Int("port", target.Port))
		return
	}

	// Set resource attributes
	rb := s.mb.NewResourceBuilder()
	rb.SetIperfTargetHost(target.Host)
	rb.SetIperfTargetPort(int64(target.Port))
	resource := rb.Emit()
	s.mb.ResourceOption(resource)

	// Record metrics from the report
	s.recordMetrics(report, target, timestamp, testDuration)
}

func (s *scraper) recordMetrics(report *iperf.Report, target TargetConfig, timestamp pcommon.Timestamp, testDuration float64) {
	if report.End == nil {
		s.logger.Warn("Report has no end section", 
			zap.String("host", target.Host),
			zap.Int("port", target.Port))
		return
	}

	// Record test duration
	s.mb.RecordIperfTestDurationDataPoint(timestamp, testDuration, target.Protocol)

	// Process sum stats
	if report.End.SumSent != nil {
		// Bandwidth (bits per second)
		s.mb.RecordIperfBandwidthDataPoint(timestamp,
			report.End.SumSent.BitsPerSecond,
			target.Protocol,
			"send",
			int64(target.Streams))

		// Transfer (bytes)
		s.mb.RecordIperfTransferDataPoint(timestamp,
			int64(report.End.SumSent.Bytes),
			target.Protocol,
			"send")
	}

	if report.End.SumReceived != nil {
		// Bandwidth (bits per second) 
		s.mb.RecordIperfBandwidthDataPoint(timestamp,
			report.End.SumReceived.BitsPerSecond,
			target.Protocol,
			"receive",
			int64(target.Streams))

		// Transfer (bytes)
		s.mb.RecordIperfTransferDataPoint(timestamp,
			int64(report.End.SumReceived.Bytes),
			target.Protocol,
			"receive")
	}

	// TCP-specific metrics
	if target.Protocol == "tcp" && report.End.SumSent != nil {
		// Retransmits
		if report.End.SumSent.Retransmits > 0 {
			s.mb.RecordIperfRetransmitsDataPoint(timestamp,
				int64(report.End.SumSent.Retransmits),
				target.Protocol)
		}
	}

	// UDP-specific metrics
	if target.Protocol == "udp" {
		if report.End.SumReceived != nil {
			// Jitter
			if report.End.SumReceived.Jitter > 0 {
				s.mb.RecordIperfJitterDataPoint(timestamp,
					report.End.SumReceived.Jitter,
					target.Protocol,
					"receive")
			}

			// Packet loss
			if report.End.SumReceived.LostPercent > 0 {
				s.mb.RecordIperfPacketLossDataPoint(timestamp,
					report.End.SumReceived.LostPercent,
					target.Protocol,
					"receive")
			}
		}
	}

	// CPU utilization (if available)
	if report.End.CPUUtilizationPercent != nil {
		if report.End.CPUUtilizationPercent.HostTotal > 0 {
			s.mb.RecordIperfCpuUtilizationDataPoint(timestamp,
				report.End.CPUUtilizationPercent.HostTotal,
				target.Protocol,
				"send")
		}
		if report.End.CPUUtilizationPercent.RemoteTotal > 0 {
			s.mb.RecordIperfCpuUtilizationDataPoint(timestamp,
				report.End.CPUUtilizationPercent.RemoteTotal,
				target.Protocol,
				"receive")
		}
	}
}