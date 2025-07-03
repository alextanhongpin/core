module github.com/alextanhongpin/core

go 1.23.0

require golang.org/x/exp v0.0.0-20250620022241-b7579e27df2b

require (
	github.com/alextanhongpin/core/sync/pipeline v0.0.0-20250703044817-a8c0737b507c
	github.com/alextanhongpin/core/sync/promise v0.0.0-20250703044817-a8c0737b507c
	github.com/segmentio/kafka-go v0.4.48
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/alextanhongpin/core/sync/rate v0.0.0-20250703044817-a8c0737b507c // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sync v0.15.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/imdario/mergo => github.com/imdario/mergo v0.3.16
