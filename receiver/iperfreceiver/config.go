// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package iperfreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/iperfreceiver"

import (
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/scraper/scraperhelper"
	"go.uber.org/multierr"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/iperfreceiver/internal/metadata"
)

// Predefined error responses for configuration validation failures
var (
	errInvalidHost     = errors.New("host cannot be empty")
	errInvalidPort     = errors.New("port must be between 1 and 65535")
	errInvalidDuration = errors.New("duration must be positive")
	errInvalidStreams  = errors.New("streams must be positive")
	errNoTargets       = errors.New("at least one target must be configured")
)

// Config defines the configuration for the iperf receiver
type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	metadata.MetricsBuilderConfig  `mapstructure:",squash"`

	// Targets defines the list of iperf3 servers to test against
	Targets []TargetConfig `mapstructure:"targets"`

	// Mode defines whether to run as client or server
	Mode string `mapstructure:"mode"`

	// ServerPort defines the port to listen on when running as server
	ServerPort int `mapstructure:"server_port"`
}

// TargetConfig defines the configuration for an individual iperf target
type TargetConfig struct {
	// Host is the hostname or IP address of the iperf3 server
	Host string `mapstructure:"host"`

	// Port is the port number of the iperf3 server
	Port int `mapstructure:"port"`

	// Duration is the test duration in seconds
	Duration time.Duration `mapstructure:"duration"`

	// Streams is the number of parallel client streams to run
	Streams int `mapstructure:"streams"`

	// Protocol is the test protocol (tcp, udp, sctp)
	Protocol string `mapstructure:"protocol"`

	// Reverse runs the test in reverse mode (server sends, client receives)
	Reverse bool `mapstructure:"reverse"`

	// Bandwidth target for UDP tests (bits per second)
	Bandwidth string `mapstructure:"bandwidth"`

	// Window size (socket buffer size)
	Window string `mapstructure:"window"`

	// MSS - Maximum Segment Size
	MSS int `mapstructure:"mss"`

	// NoDelay disables Nagle's Algorithm
	NoDelay bool `mapstructure:"no_delay"`

	// OmitSec is the number of seconds to omit from the beginning of the test
	OmitSec int `mapstructure:"omit"`

	// ZeroCopy uses zero-copy sendfile() system call
	ZeroCopy bool `mapstructure:"zero_copy"`

	// Congestion algorithm (e.g., cubic, reno)
	Congestion string `mapstructure:"congestion"`
}

// Validate validates the receiver configuration
func (cfg *Config) Validate() error {
	var err error

	// Validate mode
	if cfg.Mode != "client" && cfg.Mode != "server" && cfg.Mode != "" {
		err = multierr.Append(err, fmt.Errorf("invalid mode: %s, must be 'client' or 'server'", cfg.Mode))
	}

	// Default to client mode if not specified
	if cfg.Mode == "" {
		cfg.Mode = "client"
	}

	// Validate server port if in server mode
	if cfg.Mode == "server" {
		if cfg.ServerPort < 1 || cfg.ServerPort > 65535 {
			err = multierr.Append(err, errInvalidPort)
		}
	}

	// Validate targets for client mode
	if cfg.Mode == "client" {
		if len(cfg.Targets) == 0 {
			err = multierr.Append(err, errNoTargets)
		}

		for i, target := range cfg.Targets {
			if targetErr := target.Validate(); targetErr != nil {
				err = multierr.Append(err, fmt.Errorf("target[%d]: %w", i, targetErr))
			}
		}
	}

	return err
}

// Validate validates an individual target configuration
func (cfg *TargetConfig) Validate() error {
	var err error

	if cfg.Host == "" {
		err = multierr.Append(err, errInvalidHost)
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		err = multierr.Append(err, errInvalidPort)
	}

	if cfg.Duration <= 0 {
		cfg.Duration = 10 * time.Second // Default duration
	}

	if cfg.Streams < 0 {
		err = multierr.Append(err, errInvalidStreams)
	} else if cfg.Streams == 0 {
		cfg.Streams = 1 // Default to 1 stream
	}

	// Validate protocol
	if cfg.Protocol == "" {
		cfg.Protocol = "tcp" // Default protocol
	} else if cfg.Protocol != "tcp" && cfg.Protocol != "udp" && cfg.Protocol != "sctp" {
		err = multierr.Append(err, fmt.Errorf("invalid protocol: %s, must be tcp, udp, or sctp", cfg.Protocol))
	}

	// Validate omit seconds
	if cfg.OmitSec < 0 {
		err = multierr.Append(err, fmt.Errorf("omit seconds cannot be negative"))
	}

	// Validate MSS
	if cfg.MSS < 0 {
		err = multierr.Append(err, fmt.Errorf("MSS cannot be negative"))
	}

	return err
}