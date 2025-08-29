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

func TestReceiverLifecycle(t *testing.T) {
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

	sink := new(consumertest.MetricsSink)
	set := receivertest.NewNopSettings()
	
	r := &ztraceReceiver{
		config:   cfg,
		settings: set,
		consumer: sink,
	}

	ctx := context.Background()
	err := r.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	assert.NotNil(t, r.stopCh)
	assert.NotNil(t, r.tracer)

	err = r.Shutdown(ctx)
	require.NoError(t, err)
}

func TestConvertToMetrics(t *testing.T) {
	cfg := &Config{
		Protocol:          "udp",
		EnableGeolocation: true,
		EnableASNLookup:   true,
	}

	r := &ztraceReceiver{
		config:   cfg,
		settings: receivertest.NewNopSettings(),
	}

	result := &traceResult{
		hops: []hopInfo{
			{
				ttl:        1,
				ip:         "192.168.1.1",
				hostname:   "router.local",
				latency:    2.5,
				packetLoss: 0,
				jitter:     0.5,
				city:       "San Francisco",
				country:    "US",
				asn:        "AS15169",
				provider:   "Google",
			},
			{
				ttl:        2,
				ip:         "10.0.0.1",
				hostname:   "gateway.isp.net",
				latency:    10.2,
				packetLoss: 5.0,
				jitter:     1.2,
			},
		},
		totalLatency:  12.7,
		targetReached: true,
	}

	target := TargetConfig{
		Endpoint: "example.com",
		Port:     80,
		Tags: map[string]string{
			"env": "test",
		},
	}

	metrics := r.convertToMetrics(result, target)
	
	require.Equal(t, 1, metrics.ResourceMetrics().Len())
	rm := metrics.ResourceMetrics().At(0)
	
	// Check resource attributes
	attrs := rm.Resource().Attributes()
	val, ok := attrs.Get("ztrace.target")
	assert.True(t, ok)
	assert.Equal(t, "example.com", val.Str())
	
	val, ok = attrs.Get("ztrace.protocol")
	assert.True(t, ok)
	assert.Equal(t, "udp", val.Str())
	
	val, ok = attrs.Get("env")
	assert.True(t, ok)
	assert.Equal(t, "test", val.Str())

	// Check that metrics were created
	require.Equal(t, 1, rm.ScopeMetrics().Len())
	sm := rm.ScopeMetrics().At(0)
	assert.Greater(t, sm.Metrics().Len(), 0)

	// Verify specific metrics exist
	foundLatency := false
	foundHopCount := false
	for i := 0; i < sm.Metrics().Len(); i++ {
		metric := sm.Metrics().At(i)
		switch metric.Name() {
		case "ztrace.hop.latency":
			foundLatency = true
			assert.Equal(t, "ms", metric.Unit())
		case "ztrace.hop_count":
			foundHopCount = true
			gauge := metric.Gauge()
			assert.Equal(t, 1, gauge.DataPoints().Len())
			assert.Equal(t, int64(2), gauge.DataPoints().At(0).IntValue())
		}
	}
	assert.True(t, foundLatency, "latency metric not found")
	assert.True(t, foundHopCount, "hop count metric not found")
}

func TestConvertToTraces(t *testing.T) {
	cfg := &Config{
		Protocol:          "icmp",
		EnableGeolocation: true,
		EnableASNLookup:   true,
	}

	r := &ztraceReceiver{
		config:   cfg,
		settings: receivertest.NewNopSettings(),
	}

	result := &traceResult{
		hops: []hopInfo{
			{
				ttl:      1,
				ip:       "192.168.1.1",
				hostname: "router.local",
				latency:  2.5,
			},
			{
				ttl:        2,
				ip:         "10.0.0.1",
				hostname:   "gateway.isp.net",
				latency:    10.2,
				packetLoss: 60.0, // High packet loss to trigger event
			},
		},
		totalLatency:  12.7,
		targetReached: true,
	}

	target := TargetConfig{
		Endpoint: "example.com",
		Tags: map[string]string{
			"env": "prod",
		},
	}

	traces := r.convertToTraces(result, target)
	
	require.Equal(t, 1, traces.ResourceSpans().Len())
	rs := traces.ResourceSpans().At(0)
	
	// Check resource attributes
	attrs := rs.Resource().Attributes()
	val, ok := attrs.Get("service.name")
	assert.True(t, ok)
	assert.Equal(t, "ztrace", val.Str())
	
	val, ok = attrs.Get("env")
	assert.True(t, ok)
	assert.Equal(t, "prod", val.Str())

	// Check spans
	require.Equal(t, 1, rs.ScopeSpans().Len())
	ss := rs.ScopeSpans().At(0)
	
	// Should have root span + 2 hop spans = 3 total
	assert.Equal(t, 3, ss.Spans().Len())

	// Find and verify the root span
	var rootSpan *spanWrapper
	for i := 0; i < ss.Spans().Len(); i++ {
		span := ss.Spans().At(i)
		if span.Name() == "traceroute to example.com" {
			rootSpan = &spanWrapper{span: span}
			break
		}
	}
	require.NotNil(t, rootSpan, "root span not found")
	
	// Verify root span attributes
	hopCount, ok := rootSpan.span.Attributes().Get("hop.count")
	assert.True(t, ok)
	assert.Equal(t, int64(2), hopCount.Int())

	// Check for high packet loss event
	foundHighPacketLossEvent := false
	for i := 0; i < ss.Spans().Len(); i++ {
		span := ss.Spans().At(i)
		if span.Events().Len() > 0 {
			for j := 0; j < span.Events().Len(); j++ {
				event := span.Events().At(j)
				if event.Name() == "high_packet_loss" {
					foundHighPacketLossEvent = true
					break
				}
			}
		}
	}
	assert.True(t, foundHighPacketLossEvent, "high packet loss event not found")
}

// Helper wrapper for span testing
type spanWrapper struct {
	span interface {
		Name() string
		Attributes() interface {
			Get(string) (interface{ Int() int64 }, bool)
		}
	}
}