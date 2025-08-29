// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ztracereceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/ztracereceiver"

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

type ztraceReceiver struct {
	config        *Config
	settings      receiver.Settings
	consumer      consumer.Metrics
	traceConsumer consumer.Traces
	stopCh        chan struct{}
	stopOnce      sync.Once
	wg            sync.WaitGroup
	tracer        *tracer
}

func (r *ztraceReceiver) Start(ctx context.Context, host component.Host) error {
	r.stopCh = make(chan struct{})
	
	// Initialize the tracer with the configured protocol
	var err error
	r.tracer, err = newTracer(r.config.Protocol, r.settings.Logger)
	if err != nil {
		return fmt.Errorf("failed to create tracer: %w", err)
	}

	// Start collection goroutines for each target
	for _, target := range r.config.Targets {
		r.wg.Add(1)
		go r.collect(target)
	}

	r.settings.Logger.Info("ztrace receiver started",
		zap.Int("targets", len(r.config.Targets)),
		zap.String("protocol", r.config.Protocol))

	return nil
}

func (r *ztraceReceiver) Shutdown(ctx context.Context) error {
	r.stopOnce.Do(func() {
		close(r.stopCh)
	})
	r.wg.Wait()
	
	if r.tracer != nil {
		r.tracer.close()
	}
	
	r.settings.Logger.Info("ztrace receiver stopped")
	return nil
}

func (r *ztraceReceiver) collect(target TargetConfig) {
	defer r.wg.Done()

	ticker := time.NewTicker(r.config.CollectionInterval)
	defer ticker.Stop()

	// Run immediately on start
	r.runTrace(target)

	for {
		select {
		case <-ticker.C:
			r.runTrace(target)
		case <-r.stopCh:
			return
		}
	}
}

func (r *ztraceReceiver) runTrace(target TargetConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), r.config.Timeout)
	defer cancel()

	r.settings.Logger.Debug("Running trace", zap.String("target", target.Endpoint))

	result, err := r.tracer.trace(ctx, target, r.config)
	if err != nil {
		r.settings.Logger.Error("Failed to trace target",
			zap.String("target", target.Endpoint),
			zap.Error(err))
		return
	}

	// Convert trace result to metrics
	if r.consumer != nil {
		metrics := r.convertToMetrics(result, target)
		if err := r.consumer.ConsumeMetrics(ctx, metrics); err != nil {
			r.settings.Logger.Error("Failed to consume metrics", zap.Error(err))
		}
	}

	// Convert trace result to traces
	if r.traceConsumer != nil {
		traces := r.convertToTraces(result, target)
		if err := r.traceConsumer.ConsumeTraces(ctx, traces); err != nil {
			r.settings.Logger.Error("Failed to consume traces", zap.Error(err))
		}
	}
}

func (r *ztraceReceiver) convertToMetrics(result *traceResult, target TargetConfig) pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	
	// Set resource attributes
	resource := rm.Resource()
	resource.Attributes().PutStr("ztrace.target", target.Endpoint)
	resource.Attributes().PutStr("ztrace.protocol", r.config.Protocol)
	if target.Port > 0 {
		resource.Attributes().PutInt("ztrace.port", int64(target.Port))
	}
	
	// Add custom tags
	for k, v := range target.Tags {
		resource.Attributes().PutStr(k, v)
	}

	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("ztrace")
	sm.Scope().SetVersion("1.0.0")

	timestamp := pcommon.NewTimestampFromTime(time.Now())

	// Create metrics for each hop
	for _, hop := range result.hops {
		// Latency metric
		latencyMetric := sm.Metrics().AppendEmpty()
		latencyMetric.SetName("ztrace.hop.latency")
		latencyMetric.SetDescription("Latency for each hop in the trace")
		latencyMetric.SetUnit("ms")
		
		gauge := latencyMetric.SetEmptyGauge()
		dp := gauge.DataPoints().AppendEmpty()
		dp.SetTimestamp(timestamp)
		dp.SetDoubleValue(hop.latency)
		dp.Attributes().PutInt("ttl", int64(hop.ttl))
		dp.Attributes().PutStr("ip", hop.ip)
		if hop.hostname != "" {
			dp.Attributes().PutStr("hostname", hop.hostname)
		}
		if r.config.EnableGeolocation && hop.city != "" {
			dp.Attributes().PutStr("city", hop.city)
			dp.Attributes().PutStr("country", hop.country)
		}
		if r.config.EnableASNLookup && hop.asn != "" {
			dp.Attributes().PutStr("asn", hop.asn)
			dp.Attributes().PutStr("provider", hop.provider)
		}

		// Packet loss metric
		if hop.packetLoss > 0 {
			lossMetric := sm.Metrics().AppendEmpty()
			lossMetric.SetName("ztrace.hop.packet_loss")
			lossMetric.SetDescription("Packet loss percentage for each hop")
			lossMetric.SetUnit("%")
			
			lossGauge := lossMetric.SetEmptyGauge()
			lossDp := lossGauge.DataPoints().AppendEmpty()
			lossDp.SetTimestamp(timestamp)
			lossDp.SetDoubleValue(hop.packetLoss)
			lossDp.Attributes().PutInt("ttl", int64(hop.ttl))
			lossDp.Attributes().PutStr("ip", hop.ip)
		}

		// Jitter metric
		if hop.jitter > 0 {
			jitterMetric := sm.Metrics().AppendEmpty()
			jitterMetric.SetName("ztrace.hop.jitter")
			jitterMetric.SetDescription("Jitter for each hop in the trace")
			jitterMetric.SetUnit("ms")
			
			jitterGauge := jitterMetric.SetEmptyGauge()
			jitterDp := jitterGauge.DataPoints().AppendEmpty()
			jitterDp.SetTimestamp(timestamp)
			jitterDp.SetDoubleValue(hop.jitter)
			jitterDp.Attributes().PutInt("ttl", int64(hop.ttl))
			jitterDp.Attributes().PutStr("ip", hop.ip)
		}
	}

	// Overall trace metrics
	if result.totalLatency > 0 {
		totalLatencyMetric := sm.Metrics().AppendEmpty()
		totalLatencyMetric.SetName("ztrace.total_latency")
		totalLatencyMetric.SetDescription("Total latency to reach the target")
		totalLatencyMetric.SetUnit("ms")
		
		totalGauge := totalLatencyMetric.SetEmptyGauge()
		totalDp := totalGauge.DataPoints().AppendEmpty()
		totalDp.SetTimestamp(timestamp)
		totalDp.SetDoubleValue(result.totalLatency)
	}

	hopCountMetric := sm.Metrics().AppendEmpty()
	hopCountMetric.SetName("ztrace.hop_count")
	hopCountMetric.SetDescription("Number of hops to reach the target")
	hopCountMetric.SetUnit("1")
	
	hopGauge := hopCountMetric.SetEmptyGauge()
	hopDp := hopGauge.DataPoints().AppendEmpty()
	hopDp.SetTimestamp(timestamp)
	hopDp.SetIntValue(int64(len(result.hops)))

	return md
}

func (r *ztraceReceiver) convertToTraces(result *traceResult, target TargetConfig) ptrace.Traces {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	
	// Set resource attributes
	resource := rs.Resource()
	resource.Attributes().PutStr("ztrace.target", target.Endpoint)
	resource.Attributes().PutStr("ztrace.protocol", r.config.Protocol)
	resource.Attributes().PutStr("service.name", "ztrace")
	if target.Port > 0 {
		resource.Attributes().PutInt("ztrace.port", int64(target.Port))
	}
	
	// Add custom tags
	for k, v := range target.Tags {
		resource.Attributes().PutStr(k, v)
	}

	ss := rs.ScopeSpans().AppendEmpty()
	ss.Scope().SetName("ztrace")
	ss.Scope().SetVersion("1.0.0")

	// Create a root span for the entire trace
	rootSpan := ss.Spans().AppendEmpty()
	rootSpan.SetName(fmt.Sprintf("traceroute to %s", target.Endpoint))
	rootSpan.SetKind(ptrace.SpanKindClient)
	
	traceID := pcommon.TraceID([16]byte{}) // Generate proper trace ID
	rootSpanID := pcommon.SpanID([8]byte{}) // Generate proper span ID
	rootSpan.SetTraceID(traceID)
	rootSpan.SetSpanID(rootSpanID)
	
	startTime := pcommon.NewTimestampFromTime(time.Now().Add(-time.Duration(result.totalLatency) * time.Millisecond))
	endTime := pcommon.NewTimestampFromTime(time.Now())
	rootSpan.SetStartTimestamp(startTime)
	rootSpan.SetEndTimestamp(endTime)
	
	rootSpan.Attributes().PutInt("hop.count", int64(len(result.hops)))
	rootSpan.Attributes().PutDouble("total.latency.ms", result.totalLatency)

	// Create child spans for each hop
	for _, hop := range result.hops {
		hopSpan := ss.Spans().AppendEmpty()
		hopSpan.SetName(fmt.Sprintf("hop %d: %s", hop.ttl, hop.ip))
		hopSpan.SetKind(ptrace.SpanKindClient)
		hopSpan.SetTraceID(traceID)
		
		hopSpanID := pcommon.SpanID([8]byte{byte(hop.ttl)}) // Generate proper span ID
		hopSpan.SetSpanID(hopSpanID)
		hopSpan.SetParentSpanID(rootSpanID)
		
		hopStartTime := startTime
		hopEndTime := pcommon.NewTimestampFromTime(startTime.AsTime().Add(time.Duration(hop.latency) * time.Millisecond))
		hopSpan.SetStartTimestamp(hopStartTime)
		hopSpan.SetEndTimestamp(hopEndTime)
		
		// Set hop attributes
		hopSpan.Attributes().PutInt("ttl", int64(hop.ttl))
		hopSpan.Attributes().PutStr("ip", hop.ip)
		hopSpan.Attributes().PutDouble("latency.ms", hop.latency)
		
		if hop.hostname != "" {
			hopSpan.Attributes().PutStr("hostname", hop.hostname)
		}
		if hop.packetLoss > 0 {
			hopSpan.Attributes().PutDouble("packet_loss.percent", hop.packetLoss)
		}
		if hop.jitter > 0 {
			hopSpan.Attributes().PutDouble("jitter.ms", hop.jitter)
		}
		if r.config.EnableGeolocation && hop.city != "" {
			hopSpan.Attributes().PutStr("geo.city", hop.city)
			hopSpan.Attributes().PutStr("geo.country", hop.country)
		}
		if r.config.EnableASNLookup && hop.asn != "" {
			hopSpan.Attributes().PutStr("network.asn", hop.asn)
			hopSpan.Attributes().PutStr("network.provider", hop.provider)
		}
		
		// Add events for significant issues
		if hop.packetLoss > 50 {
			event := hopSpan.Events().AppendEmpty()
			event.SetName("high_packet_loss")
			event.SetTimestamp(hopEndTime)
			event.Attributes().PutDouble("packet_loss.percent", hop.packetLoss)
		}
	}

	return td
}