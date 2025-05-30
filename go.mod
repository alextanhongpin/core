module github.com/alextanhongpin/core

go 1.23.0

require golang.org/x/exp v0.0.0-20250506013437-ce4c2cf36ca6

require (
	github.com/alextanhongpin/core/sync/pipeline v0.0.0-20250530081951-9764c3eb58c7
	github.com/alextanhongpin/core/sync/promise v0.0.0-20250529172053-f5cc2f332e08
	github.com/prometheus/client_golang v1.22.0
	github.com/prometheus/common v0.64.0
	github.com/segmentio/kafka-go v0.4.48
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/otel v1.36.0
	go.opentelemetry.io/otel/metric v1.36.0
	go.opentelemetry.io/otel/trace v1.36.0
	golang.org/x/exp/event v0.0.0-20250506013437-ce4c2cf36ca6
)

require (
	github.com/alextanhongpin/core/sync/rate v0.0.0-20250529172053-f5cc2f332e08 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	golang.org/x/sync v0.14.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/imdario/mergo => github.com/imdario/mergo v0.3.16
