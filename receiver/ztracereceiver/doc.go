// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package ztracereceiver provides a receiver that performs network traceroute
// operations and converts the results into OpenTelemetry metrics and traces.
// It supports multiple protocols (UDP, ICMP, TCP) and can enrich hop data with
// geolocation and ASN information.
package ztracereceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/ztracereceiver"