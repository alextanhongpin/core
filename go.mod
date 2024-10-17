module github.com/alextanhongpin/core

go 1.23.0

require golang.org/x/exp v0.0.0-20241009180824-f66d83c29e7c

require (
	github.com/alextanhongpin/core/sync/pipeline v0.0.0-20241017073553-0b055471438e
	github.com/alextanhongpin/core/sync/promise v0.0.0-20241017073553-0b055471438e
	github.com/mitchellh/copystructure v1.2.0
	github.com/prometheus/client_golang v1.20.5
	github.com/prometheus/common v0.60.0
	github.com/segmentio/kafka-go v0.4.47
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/otel v1.31.0
	go.opentelemetry.io/otel/metric v1.31.0
	go.opentelemetry.io/otel/trace v1.31.0
	golang.org/x/exp/event v0.0.0-20241009180824-f66d83c29e7c
	golang.org/x/sync v0.8.0
)

require (
	github.com/alextanhongpin/core/sync/rate v0.0.0-20241017073553-0b055471438e // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	google.golang.org/protobuf v1.35.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/imdario/mergo => github.com/imdario/mergo v0.3.16
