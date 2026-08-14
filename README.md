# gotmpl2text

[![Go Reference](https://pkg.go.dev/badge/github.com/andrew-grechkin/gotmpl2text.svg)](https://pkg.go.dev/github.com/andrew-grechkin/gotmpl2text)

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
- GOTMPL_PRELOAD: colon-separated list of template files to preload (semicolon on Windows) (see [Template Preloading](#template-preloading))
- GOTMPL_DEBUG=1: enable debug mode (diagnostic output to STDERR)

## INSTALLATION

### Using [`mise`](https://mise.jdx.dev)

```bash
mise use go:github.com/andrew-grechkin/gotmpl2text@latest
```

### Building from source

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
- Extra functions: `uuidv7`, `uuidv7ToEpoch`, `uuidv7ToEpochNs`

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
gotmpl2text <<< 'UUID v7: {{ uuidv7 }}'
gotmpl2text <<< 'Date: {{ now | date "2006-01-02" }}'
```

### Single data file

Render with inline JSON data:

```bash
gotmpl2text <<< '{{ .name }}: {{ .replicas }}' <(echo '{"name":"my-service","replicas":3}')
```

Or with actual files:

```bash
gotmpl2text < template.tmpl data.yaml
```

Or [UUOC](<https://en.wikipedia.org/wiki/Cat_(Unix)#Useless_use_of_cat>), if one would like:

```bash
cat template.tmpl | gotmpl2text data.yaml
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
gotmpl2text < template.tmpl base.yaml override.yaml override2.json
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

#### UUID functions

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

#### JSON functions

**`toJsonL`** - JSON Lines / NDJSON: one compact JSON value per line. Common format for streaming logs and batch records.
Errors if input isn't an array or slice.

```bash
gotmpl2text <<'EOF'
{{ .items | toJsonL -}}
{{/* __DATA__
items:
  - {id: 1, name: alice}
  - {id: 2, name: bob}
*/}}
EOF
# STDOUT:
# {"id":1,"name":"alice"}
# {"id":2,"name":"bob"}
```

##### Sprig overrides

Sprig ships `toJson`/`toPrettyJson`/`toRawJson`/`fromJson` as **silent** variants that swallow errors - on marshal
failure they return `""` and on unmarshal failure they return `map{"Error": "..."}`. This is dangerous in templating and
one have to remember to always use `must` versions of these functions because this approach leading to corrupted
output that downstream code can't distinguish from real data. Since there's no meaningful continuation after a
serialization failure, this tool replaces those four names with loud variants that halt template execution on error.

Additional fix in `fromJson`: sprig's version only accepted JSON **objects** (returned `map[string]interface{}`). This
implementation accepts any valid JSON value (object, array, scalar)

```bash
# Array now parses correctly (sprig's fromJson would error-stash)
gotmpl2text <<< '{{ range fromJson "[1,2,3]" }}{{ . }} {{ end }}'
# STDOUT: 1 2 3

# Malformed JSON halts the template (was silent corruption)
gotmpl2text <<< '{{ fromJson "not valid" }}'
# STDERR: template: STDIN:1:3: ... error calling fromJson: fromJson: invalid character ...
```

#### Regex functions

**`test`** - Return `true` if the regex matches anywhere in the text:

```bash
gotmpl2text <<< '{{ if "v1.2.3" | test "^v[0-9]+" }}looks like a version{{ end }}'
# STDOUT: looks like a version
```

**`match`** - Return capture groups as a slice (full match excluded). Empty slice on no match:

```bash
gotmpl2text <<< '{{ range ("user@example.com" | match "^([^@]+)@(.+)$") }}{{ . }} {{ end }}'
# STDOUT: user example.com
```

Sprig's regex helpers have awkward argument orders that don't compose well in pipelines, so `gotmpl2text` ships its own
set:

**`substitute`** - Replace the first regex match (like Perl `s///r`):

```bash
gotmpl2text <<< '{{ "foo foo foo" | substitute "foo" "bar" }}'
# STDOUT: bar foo foo
```

Capture groups can be referenced in the replacement with `$1`, `$2`, ...:

```bash
gotmpl2text <<< '{{ "key=value" | substitute "(\\w+)=(\\w+)" "$2=$1" }}'
# STDOUT: value=key
```

**`substituteAll`** - Replace every regex match (like Perl `s///g`):

```bash
gotmpl2text <<< '{{ "a1 b22 c333" | substituteAll "[0-9]+" "N" }}'
# STDOUT: aN bN cN
```

**`splitBy`** - Split a string by a regex pattern. Two forms. Named `splitBy` (not `split`) to leave sprig's own `split`
alone; the pattern argument at the call site makes the intent obvious.

```bash
# unlimited splits: pattern then text
gotmpl2text <<< '{{ "a,b,c" | splitBy "," }}'
# STDOUT: [a b c]

# limited splits: pattern, max items, then text
gotmpl2text <<< '{{ "a,b,c,d" | splitBy "," 2 }}'
# STDOUT: [a b,c,d]
```

#### Dict functions

Look up values in nested maps by dotted path. All three read naturally as `value_at path` / `exists_at path` /
`defined_at path`.

**`valueAt`** - Return the value at a dotted path. Pipeline-friendly, default optional (nil when omitted)

```bash
gotmpl2text <<'EOF'
{{ .data | valueAt "server.tls.enabled" }}
{{/* __DATA__
data:
  server:
    tls:
      enabled: true
*/}}
EOF
# STDOUT: true

# With default when path is missing
gotmpl2text <<'EOF'
{{ .data | valueAt "server.timeout" "30s" }}
{{/* __DATA__
data:
  server:
    host: localhost
*/}}
EOF
# STDOUT: 30s
```

**`existsAt`** - Return true if the path resolves. Present but nil counts as existing

```bash
gotmpl2text <<'EOF'
{{ if .data | existsAt "server.tls" }}tls block is present{{ end }}
{{/* __DATA__
data:
  server:
    tls:
      enabled: null
*/}}
EOF
# STDOUT: tls block is present
```

**`definedAt`** - Return true only if the path resolves AND its value is non-nil. Use when a present-but-null value
should be treated the same as missing.

```bash
gotmpl2text <<'EOF'
{{ if .data | definedAt "server.tls.enabled" }}tls.enabled is set{{ end }}
{{/* __DATA__
data:
  server:
    tls:
      enabled: null
*/}}
EOF
# (no output - value is nil)
```


**`toEntries`** / **`fromEntries`** - jq-style conversion between a map and a slice of `{"key": ..., "value": ...}`
records. `toEntries` sorts keys lexicographically for deterministic output; `fromEntries` is its inverse.

```bash
# Map entries as JSONL lines
gotmpl2text <<'EOF'
{{ .m | toEntries | toJsonL }}
{{/* __DATA__
m: {b: 2, a: 1, c: 3}
*/}}
EOF
# STDOUT:
# {"key":"a","value":1}
# {"key":"b","value":2}
# {"key":"c","value":3}

# Iterate with named fields
gotmpl2text <<'EOF'
{{ range .m | toEntries }}- {{ .key }}={{ .value }}
{{ end -}}
{{/* __DATA__
m: {name: alice, role: admin}
*/}}
EOF
# Round-trip: toEntries | fromEntries reproduces the map (with keys sorted).
```

`fromEntries` treats a missing `value` field as `nil` (matches jq). `key` must be a string; non-string keys error loudly.

#### Time functions

**`strftime`** - Format a time value using C-style tokens, avoiding Go's reference-time layout (`"2006-01-02
15:04:05"`). Accepts `time.Time`, Unix epoch (int/int64), or RFC3339/date-only string.

Supported tokens: `%Y %y %m %d %H %I %M %S %p %A %a %B %b %j %Z %z %s %e %%`

```bash
gotmpl2text <<< '{{ "2024-03-15T14:07:09Z" | strftime "%Y-%m-%d %H:%M:%S" }}'
# STDOUT: 2024-03-15 14:07:09

# For current time, pipe sprig's `now`:
gotmpl2text <<< '{{ now | strftime "%A, %B %e %Y" }}'
```

Limitations: no locale-aware weekday / month names (English only), no sub-second tokens (`%N`, `%f`). Unknown tokens are
passed through verbatim.

#### Type predicates

Bare-name predicates for the common Go template kinds - avoids sprig's stringly-typed `kindIs "map" .` pattern in
conditionals.

`isString`, `isNumber`, `isBool`, `isSlice`, `isMap`, `isNil`

```bash
gotmpl2text <<'EOF'
{{ if isMap .config }}config is a map{{ end }}
{{ if isNil .missing }}missing is nil{{ end }}
{{/* __DATA__
config:
  host: localhost
*/}}
EOF
```

`isNumber` covers all built-in numeric kinds (int, uint, float variants). `isNil` handles both untyped nil and typed
nils (nil pointer, nil slice, nil map, nil chan, nil func).

#### String functions

**`slug`** - Convert an arbitrary string into a URL/filename-friendly slug: lowercase, ASCII alphanumerics kept,
everything else collapsed to a single hyphen, no leading or trailing hyphens.

```bash
gotmpl2text <<< '{{ "Hello, World! (v2.0)" | slug }}'
# STDOUT: hello-world-v2-0
```

Errors if the input produces an empty slug (e.g. `""`, `"!!!"`, `"世界"`) - an empty slug is broken as a URL fragment or
filename, so we halt at the source rather than let it propagate.

Note: ASCII-only. Non-ASCII characters (diacritics, CJK, emoji) are treated as separators. "café" becomes "caf". For
full Unicode-aware slugs, pre-process with a normalizer before calling.

#### System functions

> NOTE: any functions in this section are impure. They make template rendering not reproducible and depend on
> environment they called in.

Runtime and environment access. Like sprig's `env`:

**`getenv`** - Read an environment variable. Returns the value as a string if set (empty string included), or `nil` if
unset. Optional default fills in for `nil`.

```bash
FOO=bar gotmpl2text <<< '{{ getenv "FOO" }}'
# STDOUT: bar

gotmpl2text <<< '{{ getenv "MISSING" "fallback" }}'
# STDOUT: fallback

# Empty-string is a real value, default is ignored
FOO= gotmpl2text <<< 'x{{ getenv "FOO" "default" }}y'
# STDOUT: xy
```

**Nil-in-concatenation pitfall**: when a var is unset and no default is provided, `getenv` returns `nil`. If you
concatenate that with a string in a template, Go's default `missingkey` behaviour renders it as `<no value>`:

```bash
gotmpl2text <<< 'X={{ getenv "MISSING" }}suffix'
# STDOUT: X=<no value>suffix
```

Pass an explicit empty default if you want an empty string on missing:

```bash
gotmpl2text <<< 'X={{ getenv "MISSING" "" }}suffix'
# STDOUT: X=suffix
```

Or compose with sprig's `default`:

```bash
gotmpl2text <<< 'X={{ getenv "MISSING" | default "" }}suffix'
```

**`hostname`** - Return host name

```bash
gotmpl2text <<< 'rendering on: {{ hostname }}'
# STDOUT: rendering on: some-host.local
```

**`os`** - Return os this binary built for

```bash
gotmpl2text <<< 'current os: {{ os }}'
# STDOUT: current os: linux
```

**`arch`** - Return machine arch

```bash
gotmpl2text <<< 'current arch: {{ arch }}'
# STDOUT: current arch: amd64
```

**`uid`** - Return current user id

**`gid`** - Return current group id

**`cwd`** - Return current working directory

### Custom functions

You can define custom template functions using Sprig template syntax in a YAML file.

**File location (in order of priority):**

1. `$GOTMPL_FUNCTIONS` (if set)
2. `$XDG_CONFIG_HOME/gotmpl2text/functions.yaml`
3. `~/.config/gotmpl2text/functions.yaml`

**Format:**

```yaml
---
$schema: https://raw.githubusercontent.com/andrew-grechkin/gotmpl2text/main/schemas/functions.yaml
functions:
  - name: myFunc
    template: |-
      {{- . | toString | upper -}}
```

**Example usage:**

```bash
# Create custom functions file
cat > ~/.config/gotmpl2text/functions.yaml <<'EO_FUNCTIONS'
---
$schema: https://raw.githubusercontent.com/andrew-grechkin/gotmpl2text/main/schemas/functions.yaml
functions:
  - name: shout
    template: |-
      {{- . | toString | upper -}}
EO_FUNCTIONS

# Use the custom function
gotmpl2text <<< '{{ "hello" | shout }}'
# STDOUT: HELLO
```

See [examples/functions.yaml](examples/functions.yaml) for more examples.

#### Typed custom functions

Custom functions can specify their return type using the `type:` field. Supported types match Sprig conventions:

- `string` (default) - text values, preserves whitespace
- `int64` - integers for arithmetic (matches Sprig `add`, `div`, etc.)
- `float64` - floating-point numbers
- `bool` - boolean values for conditionals

#### IDE Support

A [JSON Schema](schemas/functions.yaml) is provided for IDE autocomplete and validation.

You can enable it by adding the `$schema` key to your `functions.yaml`:

```yaml
$schema: https://raw.githubusercontent.com/andrew-grechkin/gotmpl2text/main/schemas/functions.yaml
```

Alternatively, use the `yaml-language-server` magic comment:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/andrew-grechkin/gotmpl2text/main/schemas/functions.yaml
```

## TEMPLATE PRELOADING

You can preload template files containing common `{{define}}` blocks via the `GOTMPL_PRELOAD` environment variable.
This is useful when working with systems that store shared template definitions in separate files.

```bash
# Preload common definitions
GOTMPL_PRELOAD="common.tmpl:helpers.tmpl" gotmpl2text < main.tmpl
```

**Behavior:**

- Files are separated by the OS-native PATH list separator (`:` on Unix, `;` on Windows) and loaded in order
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

Similar behavior can be achieved by using `cat`:

```bash
cat common.tmpl helpers.tmpl base.tmpl | gotmpl2text data.yaml
```

Using `GOTMPL_PRELOAD` has an advantage - all occurred errors will be reported with correct filename and line number.
Using `cat` will report incorrect line numbers (because tool receives all templates in this case as one big file).

## DEBUG MODE

Enable debug mode to see diagnostic information about what `gotmpl2text` is doing:

```bash
GOTMPL_DEBUG=1 gotmpl2text <<< '{{ .text | indent 4 }}' <(echo '{"text":"hello"}')
```

## ERROR MESSAGES

Template errors are emitted to STDERR with one frame per line so tools like vim quickfix and any `path:line:col` parser
can walk them:

```bash
# helpers.tmpl
{{- define "greeting" -}}
Hi {{ .name }},
{{ required "email is required" .email }}
{{- end -}}

# STDIN template
GOTMPL_PRELOAD=./helpers.tmpl gotmpl2text <<'EOF'
Report:
  {{ include "greeting" . }}
{{/* __DATA__
name: Alice
*/}}
EOF
# STDERR:
# template: STDIN:2:5: executing "STDIN" at <include "greeting" .>: error calling include
# template: ./helpers.tmpl:3:32: executing "greeting" at <.email>: map has no entry for key "email"
```

Every line starts with `template: ` and has `PATH:LINE:COL:` right after, matching vim's default `errorformat`
(`template: %f:%l:%c: %m`). Pipe STDERR into `:cbuffer` and both call sites become jumpable quickfix entries.

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

## MOTIVATION

Two things drive this project:

1. **Be the most composable template renderer for text and config generation**: Read the template from STDIN, take data
   from files or embed it inline, respect Unix philosophy and conventions (single-purpose filter, exit codes, STDERR for
   errors, `path:line:col` error format), and stay out of the way of the surrounding pipeline
2. **Address the sprig design flaws**: Sprig ships a lot of useful functions but has real ergonomic problems - regex
   helpers with awkward argument orders, `dig` that doesn't handle nested paths cleanly, `toJson`/`fromJson` that
   silently swallow serialization errors and produce corrupt output the caller can't distinguish from real data, `env`
   that can't distinguish "unset" from "empty", no proper strftime, no full slugify, single-level `hasKey` with no
   path-aware existence check. This tool ships replacements and additions in the `additional/*` packages that fix these.
   Where sprig has both silent and loud variants (`toJson` / `mustToJson`), override the silent name with loud behavior
   so rendering halts on the first serialization failure instead of continuing with garbage.
   Non-serialization fixes (regex, path lookup, predicates, strftime, etc.) are additive and coexist with sprig's
   originals; users pick what reads better

## AUTHOR

- Andrew Grechkin

## LICENSE

This project is licensed under the GNU General Public License Version 2 (GPLv2).
See the `LICENSE` file for details.
