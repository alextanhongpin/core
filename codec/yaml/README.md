# yaml

Load YAML with relative file includes and shallow, ordered merges.

```go
import yaml "github.com/core/codec/yaml"

var config map[string]any
err := yaml.LoadExtendedYAML("config.yaml", &config)
```

```yaml
"!merge": [defaults.yaml, overrides.yaml]
app: my-app
db: !include database/config.yaml
services: !include [services-a.yaml, services-b.yaml]
```

Paths resolve relative to the file containing each directive. Single includes
accept any YAML value. Multiple includes combine mappings (later keys win) or
concatenate sequences. `"!merge"` imports mappings at its position; later entries
win and nested mappings are replaced. Duplicate ordinary keys are errors.
Native YAML anchors and aliases work.

Missing files, invalid directives, cycles, nesting beyond 100 files, and multiple
documents return errors. Filesystem errors remain inspectable with `errors.Is`.
Load trusted configuration only: includes may access paths outside the root directory.

Sample YAML files live in `testdata/`; `main.yaml` demonstrates includes and merges
and is verified by the test suite.

Run `make test` for race-enabled regression tests. `cmd/comments` preserves the
original standalone comment-anchor experiment; it is not part of the loader API.
