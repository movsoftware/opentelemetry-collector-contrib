module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/ztracereceiver

go 1.22.0

require (
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/collector/component v0.118.0
	go.opentelemetry.io/collector/config/confighttp v0.118.0
	go.opentelemetry.io/collector/consumer v1.24.0
	go.opentelemetry.io/collector/consumer/consumertest v0.118.0
	go.opentelemetry.io/collector/pdata v1.24.0
	go.opentelemetry.io/collector/receiver v0.118.0
	go.opentelemetry.io/collector/receiver/receivertest v0.118.0
	go.uber.org/zap v1.27.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/open-telemetry/opentelemetry-collector-contrib/receiver/ztracereceiver => ./

retract (
	v0.76.2
	v0.76.1
)