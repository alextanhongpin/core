# yaml

Load YAML with relative file includes and shallow, ordered merges.

```go
import yaml "github.com/alextanhongpin/core/codec/yaml"

var config map[string]any
err := yaml.LoadExtendedYAML("config/main.yaml", &config)

// Or choose the root explicitly, e.g. to include from a parent directory:
err = yaml.Load(os.DirFS("config"), "prod/main.yaml", &config)
```

```yaml
"!merge": [defaults.yaml, overrides.yaml]
app: my-app
db: !include database/config.yaml
services: !include [services-a.yaml, services-b.yaml]
```

- Paths are slash-separated and resolve relative to the file containing the
  directive.
- A single include accepts any YAML value. Multiple includes combine mappings
  (later keys win) or concatenate sequences.
- `"!merge"` imports mappings at its position; later entries win and nested
  mappings are replaced. Duplicate ordinary keys are errors.
- Native YAML anchors and aliases work within a file.

`LoadExtendedYAML` confines includes to the loaded file's directory and below,
through `os.Root`, so neither `..` nor symlinks can leave it. `Load` confines
them to the given `fs.FS`; note that `os.DirFS` follows symlinks out of its
directory, while `(*os.Root).FS` does not. Absolute paths and `..` past the
root fail with `ErrPathEscape`.

Missing files, invalid directives, include cycles, nesting deeper than 100,
more than 1000 file loads in total, and multiple documents per file return
errors. Filesystem errors can be checked with `errors.Is` (e.g.
`fs.ErrNotExist`).
