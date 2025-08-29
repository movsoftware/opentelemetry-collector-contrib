// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package iperfreceiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		expectedErr string
	}{
		{
			name: "valid client config",
			cfg: &Config{
				Mode: "client",
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
			expectedErr: "",
		},
		{
			name: "valid server config",
			cfg: &Config{
				Mode:       "server",
				ServerPort: 5201,
			},
			expectedErr: "",
		},
		{
			name: "invalid mode",
			cfg: &Config{
				Mode: "invalid",
			},
			expectedErr: "invalid mode: invalid",
		},
		{
			name: "client mode without targets",
			cfg: &Config{
				Mode:    "client",
				Targets: []TargetConfig{},
			},
			expectedErr: "at least one target must be configured",
		},
		{
			name: "server mode with invalid port",
			cfg: &Config{
				Mode:       "server",
				ServerPort: 0,
			},
			expectedErr: "port must be between 1 and 65535",
		},
		{
			name: "target with empty host",
			cfg: &Config{
				Mode: "client",
				Targets: []TargetConfig{
					{
						Host: "",
						Port: 5201,
					},
				},
			},
			expectedErr: "host cannot be empty",
		},
		{
			name: "target with invalid port",
			cfg: &Config{
				Mode: "client",
				Targets: []TargetConfig{
					{
						Host: "localhost",
						Port: 70000,
					},
				},
			},
			expectedErr: "port must be between 1 and 65535",
		},
		{
			name: "target with invalid protocol",
			cfg: &Config{
				Mode: "client",
				Targets: []TargetConfig{
					{
						Host:     "localhost",
						Port:     5201,
						Protocol: "invalid",
					},
				},
			},
			expectedErr: "invalid protocol: invalid",
		},
		{
			name: "target with negative omit",
			cfg: &Config{
				Mode: "client",
				Targets: []TargetConfig{
					{
						Host:    "localhost",
						Port:    5201,
						OmitSec: -1,
					},
				},
			},
			expectedErr: "omit seconds cannot be negative",
		},
		{
			name: "target with negative MSS",
			cfg: &Config{
				Mode: "client",
				Targets: []TargetConfig{
					{
						Host: "localhost",
						Port: 5201,
						MSS:  -1,
					},
				},
			},
			expectedErr: "MSS cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			}
		})
	}
}

func TestTargetConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *TargetConfig
		expectedErr string
	}{
		{
			name: "valid config with defaults",
			cfg: &TargetConfig{
				Host: "localhost",
				Port: 5201,
			},
			expectedErr: "",
		},
		{
			name: "valid UDP config",
			cfg: &TargetConfig{
				Host:      "localhost",
				Port:      5201,
				Protocol:  "udp",
				Bandwidth: "10M",
			},
			expectedErr: "",
		},
		{
			name: "valid TCP config with options",
			cfg: &TargetConfig{
				Host:       "localhost",
				Port:       5201,
				Protocol:   "tcp",
				Streams:    4,
				Duration:   30 * time.Second,
				Window:     "416K",
				MSS:        1460,
				NoDelay:    true,
				ZeroCopy:   true,
				Congestion: "cubic",
			},
			expectedErr: "",
		},
		{
			name: "empty host",
			cfg: &TargetConfig{
				Host: "",
				Port: 5201,
			},
			expectedErr: "host cannot be empty",
		},
		{
			name: "invalid port low",
			cfg: &TargetConfig{
				Host: "localhost",
				Port: 0,
			},
			expectedErr: "port must be between 1 and 65535",
		},
		{
			name: "invalid port high",
			cfg: &TargetConfig{
				Host: "localhost",
				Port: 70000,
			},
			expectedErr: "port must be between 1 and 65535",
		},
		{
			name: "negative streams",
			cfg: &TargetConfig{
				Host:    "localhost",
				Port:    5201,
				Streams: -1,
			},
			expectedErr: "streams must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.expectedErr == "" {
				require.NoError(t, err)
				// Check defaults are set
				if tt.cfg.Duration == 0 {
					assert.Equal(t, 10*time.Second, tt.cfg.Duration)
				}
				if tt.cfg.Streams == 0 {
					assert.Equal(t, 1, tt.cfg.Streams)
				}
				if tt.cfg.Protocol == "" {
					assert.Equal(t, "tcp", tt.cfg.Protocol)
				}
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			}
		})
	}
}