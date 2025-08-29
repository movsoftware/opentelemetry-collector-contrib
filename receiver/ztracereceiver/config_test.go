// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ztracereceiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/confighttp"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr string
	}{
		{
			name: "valid config with UDP",
			config: &Config{
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
			},
		},
		{
			name: "valid config with ICMP",
			config: &Config{
				ServerConfig: confighttp.ServerConfig{
					Endpoint: "localhost:8080",
				},
				Targets: []TargetConfig{
					{
						Endpoint: "8.8.8.8",
					},
				},
				CollectionInterval: 30 * time.Second,
				Timeout:            10 * time.Second,
				Protocol:           "icmp",
				MaxHops:            30,
				PacketSize:         56,
				Retries:            3,
			},
		},
		{
			name: "no targets",
			config: &Config{
				CollectionInterval: 30 * time.Second,
				Timeout:            10 * time.Second,
				Protocol:           "udp",
				MaxHops:            30,
				PacketSize:         56,
				Retries:            3,
			},
			wantErr: "at least one target must be specified",
		},
		{
			name: "empty endpoint",
			config: &Config{
				Targets: []TargetConfig{
					{
						Endpoint: "",
						Port:     80,
					},
				},
				CollectionInterval: 30 * time.Second,
				Timeout:            10 * time.Second,
				Protocol:           "udp",
				MaxHops:            30,
				PacketSize:         56,
				Retries:            3,
			},
			wantErr: "target[0]: endpoint cannot be empty",
		},
		{
			name: "missing port for UDP",
			config: &Config{
				Targets: []TargetConfig{
					{
						Endpoint: "example.com",
					},
				},
				CollectionInterval: 30 * time.Second,
				Timeout:            10 * time.Second,
				Protocol:           "udp",
				MaxHops:            30,
				PacketSize:         56,
				Retries:            3,
			},
			wantErr: "target[0]: port must be specified for udp protocol",
		},
		{
			name: "invalid protocol",
			config: &Config{
				Targets: []TargetConfig{
					{
						Endpoint: "example.com",
						Port:     80,
					},
				},
				CollectionInterval: 30 * time.Second,
				Timeout:            10 * time.Second,
				Protocol:           "invalid",
				MaxHops:            30,
				PacketSize:         56,
				Retries:            3,
			},
			wantErr: `invalid protocol "invalid", must be one of: udp, icmp, tcp`,
		},
		{
			name: "invalid collection interval",
			config: &Config{
				Targets: []TargetConfig{
					{
						Endpoint: "example.com",
						Port:     80,
					},
				},
				CollectionInterval: 0,
				Timeout:            10 * time.Second,
				Protocol:           "udp",
				MaxHops:            30,
				PacketSize:         56,
				Retries:            3,
			},
			wantErr: "collection_interval must be positive",
		},
		{
			name: "invalid timeout",
			config: &Config{
				Targets: []TargetConfig{
					{
						Endpoint: "example.com",
						Port:     80,
					},
				},
				CollectionInterval: 30 * time.Second,
				Timeout:            0,
				Protocol:           "udp",
				MaxHops:            30,
				PacketSize:         56,
				Retries:            3,
			},
			wantErr: "timeout must be positive",
		},
		{
			name: "invalid max hops",
			config: &Config{
				Targets: []TargetConfig{
					{
						Endpoint: "example.com",
						Port:     80,
					},
				},
				CollectionInterval: 30 * time.Second,
				Timeout:            10 * time.Second,
				Protocol:           "udp",
				MaxHops:            100,
				PacketSize:         56,
				Retries:            3,
			},
			wantErr: "max_hops must be between 1 and 64",
		},
		{
			name: "invalid packet size",
			config: &Config{
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
				PacketSize:         100000,
				Retries:            3,
			},
			wantErr: "packet_size must be between 1 and 65535",
		},
		{
			name: "negative retries",
			config: &Config{
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
				Retries:            -1,
			},
			wantErr: "retries must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}