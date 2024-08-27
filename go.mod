module github.com/alextanhongpin/core

go 1.22.5

require golang.org/x/exp v0.0.0-20240823005443-9b4947da3948

require (
	github.com/alextanhongpin/core/sync/batch v0.0.0-20240826174456-c35d7bfe61bd
	github.com/alextanhongpin/core/sync/promise v0.0.0-20240820123002-affc9fec4fec
	github.com/alextanhongpin/core/sync/rate v0.0.0-20240826174456-c35d7bfe61bd
	github.com/alextanhongpin/errors v0.0.0-20230717124106-3e3c39edaa89
	github.com/alextanhongpin/testdump/httpdump v0.0.0-20240617032328-5cdd37fc0156
	github.com/go-playground/validator/v10 v10.22.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/go-cmp v0.6.0
	github.com/mitchellh/copystructure v1.2.0
	github.com/prometheus/client_golang v1.19.1
	github.com/segmentio/kafka-go v0.4.47
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/otel v1.27.0
	go.opentelemetry.io/otel/metric v1.27.0
	go.opentelemetry.io/otel/trace v1.27.0
	golang.org/x/exp/event v0.0.0-20240613232115-7f521ea00fb8
	golang.org/x/sync v0.8.0
	golang.org/x/sys v0.23.0
)

require (
	github.com/alextanhongpin/testdump/pkg/diff v0.0.0-20240617032328-5cdd37fc0156 // indirect
	github.com/alextanhongpin/testdump/pkg/reviver v0.0.0-20240617032328-5cdd37fc0156 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/gabriel-vasile/mimetype v1.4.4 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.54.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	golang.org/x/crypto v0.26.0 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	golang.org/x/tools v0.24.0 // indirect
	google.golang.org/grpc v1.64.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/imdario/mergo => github.com/imdario/mergo v0.3.16
