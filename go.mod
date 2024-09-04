module github.com/alextanhongpin/core

go 1.23.0

require golang.org/x/exp v0.0.0-20240823005443-9b4947da3948

require (
	github.com/alextanhongpin/core/sync/pipeline v0.0.0-20240903045143-084d45e55594
	github.com/alextanhongpin/core/sync/promise v0.0.0-20240903045143-084d45e55594
	github.com/mitchellh/copystructure v1.2.0
	github.com/prometheus/client_golang v1.20.2
	github.com/segmentio/kafka-go v0.4.47
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/otel v1.29.0
	go.opentelemetry.io/otel/metric v1.29.0
	go.opentelemetry.io/otel/trace v1.29.0
	golang.org/x/exp/event v0.0.0-20240823005443-9b4947da3948
	golang.org/x/sync v0.8.0
)

require (
	github.com/alextanhongpin/core/sync/rate v0.0.0-20240903045143-084d45e55594 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.58.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/imdario/mergo => github.com/imdario/mergo v0.3.16
