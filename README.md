# gotmpl2text

[![Go Reference](https://pkg.go.dev/badge/github.com/andrew-grechkin/gotmpl2text.svg)](https://pkg.go.dev/github.com/andrew-grechkin/gotmpl2text)
[![Go Report Card](https://goreportcard.com/badge/github.com/andrew-grechkin/gotmpl2text)](https://goreportcard.com/report/github.com/andrew-grechkin/gotmpl2text)

A CLI filter for testing and rendering Go templates with YAML/JSON data.

Tailored for developers working with **Helm charts**, **Kubernetes manifests**, **CI/CD templates**, **config
generators**, or any Go template-based workflow.

While working with Go template-based systems (Helm charts, CI/CD templates, config generators), developers often need to
test template snippets quickly without spinning up the full toolchain. The best if they can be tested directly in an
editor.

`gotmpl2text` is just a simple CLI filter and embedding it in any workflow is a piece of cake.

## SYNOPSIS

```bash
# template from STDIN, data as file(s)
gotmpl2text <<< '{{ .name }}: {{ .replicas }}' <(echo '{"name":"api","replicas":3}')
```

```bash
# template with embedded data from STDIN
gotmpl2text << 'EO_TEMPLATE'
Service: {{ .name }}

{{/* __DATA__
name: api
*/}}
EO_TEMPLATE
```

## OPTIONS

- -h, --help Display help message
- -m, --man Display full readme (tip: gotmpl2text --man | colored-md)
- -v, --version Display version information (tip: gotmpl2text --version | jq -r .Version)

## ENVIRONMENT

- GOTMPL_ALLOW_MISSING=1: to allow missing keys (renders `<no value>`)
- GOTMPL_IGNORE_EMBED=1: to ignore embedded `__DATA__` blocks
- GOTMPL_FUNCTIONS: path to custom functions YAML file (see [Custom Functions](#custom-functions))
- GOTMPL_PRELOAD: comma-separated list of template files to preload (see [Template Preloading](#template-preloading))
- GOTMPL_DEBUG=1: enable debug mode (diagnostic output to stderr)

## INSTALLATION

```bash
go install github.com/andrew-grechkin/gotmpl2text@latest
```

By default, `go install` creates binaries in `$GOBIN` or `$GOPATH/bin`.
To make sure you can use the installed binary you need to add this directory to your path.

```bash
# ensure the go install binaries are in your PATH, consider adding to your shell startup config
export PATH="${GOBIN:-${GOPATH:-$HOME/go}/bin}:$PATH"
```

## FEATURES

- Renders Go templates to STDOUT
- Loads data from one or more YAML/JSON files
- Fails safely on missing variables (`missingkey=error` is enabled by default)
- Deep merges multiple data files (like Helm) - later files override earlier ones
- Embedded data support - include YAML data in template with `{{/* __DATA__ ... */}}` comment
- Includes [Sprig](http://masterminds.github.io/sprig/) template functions
- Includes Helm-specific functions: `include`, `required`, `toYaml`, `fromYaml`, `nindent`, `indent`
- Extra functions: `toJson`, `uuidv7`, `uuidv7ToEpoch`, `uuidv7ToEpochNs`

### Use cases

- **Helm chart development** - test value overrides and template logic
- **Kubernetes manifest generation** - render templates with different configs
- **CI/CD template testing** - GitHub Actions, GitLab CI, Harness, etc.
- **Config file generation** - render app configs from templates
- **Documentation** - provide runnable template examples
- **Template debugging** - isolate and test problematic snippets

## USAGE

### Self-contained templates (no data needed)

```bash
# Templates can use Sprig functions without any data:
gotmpl2text <<< 'Random: {{ randAlpha 10 }}'
gotmpl2text <<< 'UUID v4: {{ uuidv4 }}'
gotmpl2text <<< 'Date: {{ now | date "2006-01-02" }}'
```

### Single data file

Render with inline JSON data:

```bash
gotmpl2text <<< '{{ .name }}: {{ .replicas }}' <(echo '{"name":"my-service","replicas":3}')
```

Or with actual files:

```bash
gotmpl2text < template.tmpl data.yaml override.json
```

Or [UUOC](<https://en.wikipedia.org/wiki/Cat_(Unix)#Useless_use_of_cat>), if one would like:

```bash
cat template.tmpl | gotmpl2text data.yaml override.json
```

### Multiple data files with deep merge

```bash
gotmpl2text \
    <<< '{{ .name }}: {{ .replicas }} replicas, debug={{ .config.debug }}' \
    <(echo '{"name":"my-service","replicas":3,"config":{"timeout":30,"debug":false}}') \
    <(echo '{"replicas":5,"config":{"debug":true,"cache":"enabled"}}')
```

Or with actual files:

```bash
gotmpl2text < template.tmpl base.yaml override.yaml
```

Later files override earlier ones (just like with `helm install -f base.yaml -f override.yaml`).

### Embedded data with `__DATA__` comment

You can embed YAML data directly in the template file using Go template comment blocks with `{{/* __DATA__ ... */}}`:

```bash
gotmpl2text <<'EO_TEMPLATE'
Hello {{ .name }}!

{{/* __DATA__
name: world
*/}}
EO_TEMPLATE
```

**Multiple embedded data blocks are supported** and work exactly like multiple data files - they're deep-merged in order
(later blocks override earlier):

```bash
gotmpl2text <<'EO_TEMPLATE'
{{ .name }}: {{ .replicas }} replicas, debug={{ .config.debug }}

{{/* __DATA__
name: my-service
replicas: 3
config:
  timeout: 30
  debug: false
*/}}

{{/* __DATA__
replicas: 5
config:
  debug: true
  cache: enabled
*/}}
EO_TEMPLATE
```

**Combining embedded data with file arguments**.
Merge order: embedded blocks (top-to-bottom) -> data files (left-to-right):

```bash
# Embedded data acts as defaults, data in files override that
gotmpl2text << 'EO_TEMPLATE' <(echo '{"env":"prod","replicas":10}')
{{ .name }}: replicas={{ .replicas }}, env={{ .env }}

{{/* __DATA__
name: my-service
replicas: 3
env: dev
*/}}
EO_TEMPLATE
```

**I consider this as a killer feature because:**

- Templates remain **100% compatible** with Helm, Sprig, and any Go template renderer
- Other tools simply ignore the comments - no syntax errors
- Perfect for self-contained template examples, testing and used in CI
- Shows base + override patterns in a single file
- No need for separate data files during development
- Works as a contract to expose what data need's to be provided

### Helm-specific functions

The tool includes Helm template functions for compatibility with Helm charts and similar systems:

**`include`** - Execute a template and return its output as a string (can be piped):

```bash
gotmpl2text <<< '{{- define "helper" -}}Hello {{ .name }}{{- end -}}{{ include "helper" . | upper }}' <(echo '{"name":"world"}')
```

**`required`** - Error if a value is missing or empty:

```bash
gotmpl2text <<< '{{ required "name must be set" .name }}' <(echo '{"name":"test"}')
gotmpl2text <<< '{{ required "name must be set" .name }}' <(echo '{"foo":"bar"}')
```

**`toYaml`** - Convert a value to YAML string:

```bash
gotmpl2text <<< 'config:{{ .config | toYaml | nindent 2 }}' <(echo '{"config":{"timeout":30,"debug":true}}')
```

**`nindent`** - Add newline and indent:

```bash
gotmpl2text <<< 'data:{{ .items | toYaml | nindent 2 }}' <(echo '{"items":["a","b","c"]}')
```

**`indent`** - Indent without leading newline:

```bash
gotmpl2text <<< '{{ .text | indent 4 }}' <(echo '{"text":"hello"}')
```

Also available: `fromYaml` (parse YAML string)

### Additional functions

Beyond Sprig and Helm functions, `gotmpl2text` provides:

**`toJson`** - Convert a value to JSON string:

```bash
gotmpl2text <<< '{{ . | toJson }}' <(echo $'hash:\n  key: value')
```

**`uuidv7`** - Generate time-ordered UUID v7:

```bash
gotmpl2text <<< 'UUID: {{ uuidv7 }}'
```

**`uuidv7ToEpochNs`** - Extract Unix epoch nanoseconds from UUID v7:

```bash
gotmpl2text <<< '{{ $uuid := uuidv7 }}{{ $uuid | uuidv7ToEpochNs }}'
```

**Note:** Returns nanoseconds as int64, which limits the range to ~292 years from Unix epoch (until year ~2262).
For dates beyond this range, use `uuidv7ToEpoch`.

**`uuidv7ToEpoch`** - Extract Unix epoch seconds from UUID v7:

```bash
gotmpl2text <<< '{{ uuidv7 | uuidv7ToEpoch }}'
```

### Custom functions

You can define custom template functions using Sprig template syntax in a YAML file.

**File location (in order of priority):**

1. `$GOTMPL_FUNCTIONS` (if set)
2. `$XDG_CONFIG_HOME/gotmpl2text/functions.yaml`
3. `~/.config/gotmpl2text/functions.yaml`

**Format:**

```yaml
---
# yaml-language-server: $schema=https://raw.githubusercontent.com/andrew-grechkin/gotmpl2text/main/schemas/functions.yaml
functions:
  - name: myFunc
    template: |-
      {{- . | toString | upper -}}
  - name: slugify
    template: |-
      {{- regexReplaceAll "[^a-z0-9]+" (. | toString | lower) "-" | trimSuffix "-" -}}
```

**Example usage:**

```bash
# Create custom functions file
cat > ~/.config/gotmpl2text/functions.yaml <<'EO_FUNCTIONS'
---
functions:
  - name: shout
    template: |-
      {{- . | toString | upper -}}
EO_FUNCTIONS

# Use the custom function
gotmpl2text <<< '{{ "hello" | shout }}'
# Output: HELLO
```

See [examples/functions.yaml](examples/functions.yaml) for more examples.

**IDE Support:** A [JSON Schema](schemas/functions.yaml) is provided for IDE autocomplete and validation.
Add this line to the top of your `functions.yaml`:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/andrew-grechkin/gotmpl2text/main/schemas/functions.yaml
```

#### Typed custom functions

Custom functions can specify their return type using the `type:` field. Supported types match Sprig conventions:

- `string` (default) - text values, preserves whitespace
- `int64` - integers for arithmetic (matches Sprig `add`, `div`, etc.)
- `float64` - floating-point numbers
- `bool` - boolean values for conditionals

## TEMPLATE PRELOADING

You can preload template files containing common `{{define}}` blocks via the `GOTMPL_PRELOAD` environment variable.
This is useful when working with systems that store shared template definitions in separate files.

```bash
# Preload common definitions
GOTMPL_PRELOAD="common.tmpl,helpers.tmpl" gotmpl2text < main.tmpl
```

**Behavior:**

- Files are comma-separated and loaded in order
- Preloaded content is concatenated before the STDIN template
- Missing preload files cause an error with exit code 2

**Example:**

```bash
# common.tmpl contains shared definitions
cat > common.tmpl <<'EO_TEMPLATE'
{{- define "app.name" -}}
my-app
{{- end -}}
EO_TEMPLATE

# main template uses the preloaded definitions
GOTMPL_PRELOAD="common.tmpl" gotmpl2text <<'EO_TEMPLATE'
Application: {{ include "app.name" . }}
EO_TEMPLATE
```

Frankly speaking this is a syntax sugar.
I personally use it for the case when I know some common definitions template must be included:
it's easier to just export environment variable than passing full set of files each time.
The same behavior can be achieved by using `cat`:

```bash
cat common.tmpl helpers.tmpl base.tmpl | gotmpl2text data.yaml
```

## DEBUG MODE

Enable debug mode to see diagnostic information about what `gotmpl2text` is doing:

```bash
GOTMPL_DEBUG=1 gotmpl2text <<< '{{ .text | indent 4 }}' <(echo '{"text":"hello"}')
```

## EXIT CODES

- **0**: Success
- **1**: Template errors, missing keys, parsing errors
- **2**: Missing preload files (GOTMPL_PRELOAD)

## CI/CD INTEGRATION

For example one can use `gotmpl2text` in CI pipelines to validate templates before deployment:

**GitHub Actions:**

```yaml
- name: Install gotmpl2text
  run: go install github.com/andrew-grechkin/gotmpl2text@latest

- name: Test Kubernetes manifests
  run: |-
    for tmpl in k8s/*.tmpl; do
        gotmpl2text values.yaml < "$tmpl" | kubectl apply --dry-run=client -f -
    done
```

**GitLab CI:**

```yaml
test:templates:
  script: |-
    go install github.com/andrew-grechkin/gotmpl2text@latest
    gotmpl2text < deployment.tmpl base.yaml overrides.yaml > deployment.yaml
    kubectl apply --dry-run=client -f deployment.yaml
```

**Pre-commit hook:**

```bash
#!/usr/bin/env bash
# .git/hooks/pre-commit
for tmpl in $(git diff --cached --name-only | grep '\.tmpl$'); do
    if ! gotmpl2text < "$tmpl" values.yaml > &>/dev/null; then
        echo "Template validation failed: $tmpl"
        gotmpl2text < "$tmpl" values.yaml
        exit 1
    fi
done
```

## AUTHOR

- Andrew Grechkin

## LICENSE

This project is licensed under the GNU General Public License Version 2 (GPLv2).
See the `LICENSE` file for details.
