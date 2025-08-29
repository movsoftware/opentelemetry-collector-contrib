module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/iperfreceiver

go 1.22.0

require (
	github.com/BGrewell/go-iperf v0.0.0-20240831193934-6a2b45559210
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/collector/component v0.116.0
	go.opentelemetry.io/collector/confmap/confmaptest v1.23.0
	go.opentelemetry.io/collector/consumer v1.23.0
	go.opentelemetry.io/collector/consumer/consumertest v0.116.0
	go.opentelemetry.io/collector/pdata v1.23.0
	go.opentelemetry.io/collector/receiver v0.116.0
	go.opentelemetry.io/collector/receiver/receivertest v0.116.0
	go.opentelemetry.io/collector/scraper v0.116.0
	go.uber.org/goleak v1.3.0
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.27.0
)