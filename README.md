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

## SYNOPSIS:

```bash
# template from STDIN, data as files
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

## OPTIONS:

* -h, --help        Show help
* -m, --man         Show full manual (this README)
* -v, --version     Show version

## INSTALLATION

To install `gotmpl2text`:

```bash
go install github.com/andrew-grechkin/gotmpl2text@latest
```

## FEATURES

- Renders Go templates to STDOUT
- Loads data from one or more YAML/JSON files
- Deep merges multiple data files (like Helm) - later files override earlier ones
- Embedded data support - include YAML data in template with `{{/* __DATA__ ... */}}` comment
- Includes [Sprig](http://masterminds.github.io/sprig/) template functions (100+ functions)
- Includes Helm-specific functions: `include`, `required`, `toYaml`, `fromYaml`, `nindent`, `indent`
- Fails safely on missing variables (`missingkey=error` is enabled by default)
- Proper error handling and exit codes

### Use Cases

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
gotmpl2text <<< 'UUID: {{ uuidv4 }}'
gotmpl2text <<< 'Date: {{ now | date "2006-01-02" }}'
```

### Single data file

Render with inline JSON data:
```bash
gotmpl2text <<< '{{ .name }}: {{ .replicas }}' <(echo '{"name":"my-service","replicas":3}')
```

Or with actual files:
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
gotmpl2text base.yaml override.yaml < template.tmpl
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
Merge order: embedded blocks (top-to-bottom) -> data files (left-to-right). Files always override embedded data:

```bash
# Embedded data acts as defaults, data in files override that
gotmpl2text <(echo '{"env":"prod","replicas":10}') << 'EO_TEMPLATE'
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

## CI/CD INTEGRATION

For example one can use `gotmpl2text` in CI pipelines to validate templates before deployment:

**GitHub Actions:**
```yaml
- name: Install gotmpl2text
  run: go install github.com/andrew-grechkin/gotmpl2text@latest

- name: Test Kubernetes manifests
  run: |
    for tmpl in k8s/*.tmpl; do
      gotmpl2text values.yaml < "$tmpl" | kubectl apply --dry-run=client -f -
    done
```

**GitLab CI:**
```yaml
test:templates:
  script: |-
    go install github.com/andrew-grechkin/gotmpl2text@latest
    gotmpl2text base.yaml overrides.yaml < deployment.tmpl > deployment.yaml
    kubectl apply --dry-run=client -f deployment.yaml
```

**Pre-commit hook:**
```bash
#!/usr/bin/env bash
# .git/hooks/pre-commit
for tmpl in $(git diff --cached --name-only | grep '\.tmpl$$'); do
  if ! gotmpl2text values.yaml < "$tmpl" > /dev/null 2>&1; then
    echo "Template validation failed: $tmpl"
    exit 1
  fi
done
```

## AUTHOR

- Andrew Grechkin

## LICENSE

This project is licensed under the GNU General Public License Version 2 (GPLv2).
See the `LICENSE` file for details.
