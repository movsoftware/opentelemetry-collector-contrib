// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ztracereceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/ztracereceiver"

import (
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
)

// Config defines configuration for the ztrace receiver
type Config struct {
	confighttp.ServerConfig `mapstructure:",squash"`

	// Targets defines the list of targets to trace
	Targets []TargetConfig `mapstructure:"targets"`

	// CollectionInterval is the interval at which to collect ztrace data
	CollectionInterval time.Duration `mapstructure:"collection_interval"`

	// Timeout for each trace operation
	Timeout time.Duration `mapstructure:"timeout"`

	// Protocol to use for tracing (udp, icmp, tcp)
	Protocol string `mapstructure:"protocol"`

	// MaxHops is the maximum number of hops to trace
	MaxHops int `mapstructure:"max_hops"`

	// PacketSize is the size of the packet to send
	PacketSize int `mapstructure:"packet_size"`

	// Retries is the number of retries for each hop
	Retries int `mapstructure:"retries"`

	// EnableGeolocation enables geolocation lookup for IP addresses
	EnableGeolocation bool `mapstructure:"enable_geolocation"`

	// EnableASNLookup enables ASN lookup for IP addresses
	EnableASNLookup bool `mapstructure:"enable_asn_lookup"`
}

// TargetConfig defines configuration for a single target
type TargetConfig struct {
	// Endpoint is the target endpoint to trace (hostname or IP)
	Endpoint string `mapstructure:"endpoint"`

	// Port is the target port (for TCP/UDP protocols)
	Port int `mapstructure:"port"`

	// Tags are optional tags to add to the metrics
	Tags map[string]string `mapstructure:"tags"`
}

// Validate checks the receiver configuration is valid
func (cfg *Config) Validate() error {
	if len(cfg.Targets) == 0 {
		return errors.New("at least one target must be specified")
	}

	for i, target := range cfg.Targets {
		if target.Endpoint == "" {
			return fmt.Errorf("target[%d]: endpoint cannot be empty", i)
		}
		if cfg.Protocol != "icmp" && target.Port <= 0 {
			return fmt.Errorf("target[%d]: port must be specified for %s protocol", i, cfg.Protocol)
		}
	}

	if cfg.CollectionInterval <= 0 {
		return errors.New("collection_interval must be positive")
	}

	if cfg.Timeout <= 0 {
		return errors.New("timeout must be positive")
	}

	if cfg.Protocol != "udp" && cfg.Protocol != "icmp" && cfg.Protocol != "tcp" {
		return fmt.Errorf("invalid protocol %q, must be one of: udp, icmp, tcp", cfg.Protocol)
	}

	if cfg.MaxHops <= 0 || cfg.MaxHops > 64 {
		return errors.New("max_hops must be between 1 and 64")
	}

	if cfg.PacketSize <= 0 || cfg.PacketSize > 65535 {
		return errors.New("packet_size must be between 1 and 65535")
	}

	if cfg.Retries < 0 {
		return errors.New("retries must be non-negative")
	}

	return nil
}

var _ component.Config = (*Config)(nil)