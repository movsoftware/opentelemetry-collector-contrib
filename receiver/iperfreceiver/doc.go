// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdgen --source=metadata.yaml --package=metadata --output=internal/metadata
//go:generate mdatagen metadata.yaml

// Package iperfreceiver implements a receiver that collects network performance
// metrics using iperf3 tests.
package iperfreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/iperfreceiver"