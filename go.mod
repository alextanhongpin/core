module github.com/alextanhongpin/core

go 1.21

require golang.org/x/exp v0.0.0-20240103183307-be819d1f06fc

require (
	cuelang.org/go v0.8.1
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/DATA-DOG/go-txdb v0.1.8
	github.com/alextanhongpin/errors v0.0.0-20230717124106-3e3c39edaa89
	github.com/go-playground/validator/v10 v10.16.0
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.5.0
	github.com/lib/pq v1.10.9
	github.com/mitchellh/copystructure v1.2.0
	github.com/ory/dockertest/v3 v3.10.0
	github.com/pganalyze/pg_query_go/v4 v4.2.3
	github.com/prometheus/client_golang v1.18.0
	github.com/redis/go-redis/v9 v9.4.0
	github.com/segmentio/kafka-go v0.4.47
	github.com/stretchr/testify v1.8.4
	github.com/uptrace/bun v1.1.16
	github.com/uptrace/bun/dialect/pgdialect v1.1.16
	github.com/uptrace/bun/driver/pgdriver v1.1.16
	github.com/uptrace/bun/extra/bundebug v1.1.16
	github.com/wk8/go-ordered-map/v2 v2.1.8
	go.opentelemetry.io/otel v1.21.0
	go.opentelemetry.io/otel/metric v1.21.0
	go.opentelemetry.io/otel/trace v1.21.0
	golang.org/x/exp/event v0.0.0-20240103183307-be819d1f06fc
	golang.org/x/oauth2 v0.18.0
	golang.org/x/sync v0.6.0
	golang.org/x/tools v0.19.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240108191215-35c7eff3a6b1
	google.golang.org/grpc v1.60.1
	google.golang.org/grpc/examples v0.0.0-20210430044426-28078834f35b
	google.golang.org/protobuf v1.32.0
	gopkg.in/yaml.v3 v3.0.1
	vitess.io/vitess v0.18.2
)

require (
	cloud.google.com/go/compute v1.23.3 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cuelabs.dev/go/oci/ociregistry v0.0.0-20240314152124-224736b49f2e // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/DataDog/appsec-internal-go v1.4.0 // indirect
	github.com/DataDog/datadog-agent/pkg/obfuscate v0.50.2 // indirect
	github.com/DataDog/datadog-agent/pkg/remoteconfig/state v0.50.2 // indirect
	github.com/DataDog/datadog-go/v5 v5.4.0 // indirect
	github.com/DataDog/go-libddwaf/v2 v2.2.3 // indirect
	github.com/DataDog/go-sqllexer v0.0.10 // indirect
	github.com/DataDog/go-tuf v1.0.2-0.5.2 // indirect
	github.com/DataDog/sketches-go v1.4.4 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cockroachdb/apd/v3 v3.2.1 // indirect
	github.com/containerd/continuity v0.4.3 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/docker/cli v24.0.7+incompatible // indirect
	github.com/docker/docker v24.0.7+incompatible // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ebitengine/purego v0.5.1 // indirect
	github.com/emicklei/proto v1.10.0 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v1.2.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/safehtml v0.1.0 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-5 // indirect
	github.com/imdario/mergo v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/opencontainers/runc v1.1.11 // indirect
	github.com/opentracing-contrib/go-grpc v0.0.0-20210225150812-73cb765af46e // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/outcaste-io/ristretto v0.2.3 // indirect
	github.com/pelletier/go-toml/v2 v2.1.1 // indirect
	github.com/philhofer/fwd v1.1.2 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.45.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/protocolbuffers/txtpbfmt v0.0.0-20230328191034-3462fbc510c0 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/secure-systems-lab/go-securesystemslib v0.8.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/cobra v1.8.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.18.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tinylib/msgp v1.1.9 // indirect
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc // indirect
	github.com/uber/jaeger-client-go v2.30.0+incompatible // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go4.org/intern v0.0.0-20230525184215-6c62f75575cb // indirect
	go4.org/unsafe/assume-no-moving-gc v0.0.0-20231121144256-b99613f794b6 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/mod v0.16.0 // indirect
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	gopkg.in/DataDog/dd-trace-go.v1 v1.59.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	inet.af/netaddr v0.0.0-20230525184311-b8eac61e914a // indirect
	mellium.im/sasl v0.3.1 // indirect
)

replace github.com/imdario/mergo => github.com/imdario/mergo v0.3.16
