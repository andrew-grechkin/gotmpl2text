#!/usr/bin/env -S just --one --justfile

set export

export tool := 'gotmpl2text'

export GOBIN := `echo "${GOBIN:-${GOPATH:-$HOME/go}/bin}"`

alias fmt := fix

# Default recipe
[private]
@default: test

# Build the binary
@build: fix
    go build

# Run complexity lint
cc:
    #!/usr/bin/env -S bash -Eeuo pipefail
    [[ -x "$GOBIN/gocyclo" ]] || go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
    "$GOBIN/gocyclo" -over 15 .

# Remove build artifacts (everything in .gitignore)
@clean:
    git clean -Xdf

# Format and modernize Go source code
@fix:
    go fmt
    go fix

# Install the binary to $GOBIN
@install:
    go install

# Run Go linter
@lint: cc
    go vet

# Run all tests
test: lint test-unit test-int

# Run integration tests
test-int: install
    #!/usr/bin/env -S bash -Eeuo pipefail

    bin="$GOBIN/$tool"

    # Unset environment variables that could interfere with tests
    unset GOTMPL_PRELOAD GOTMPL_FUNCTIONS || true

    for f in test/fixtures/*-expected.txt; do
        name=$(basename "$f" -expected.txt)

        # ${name}-args.txt: one CLI argument per line (blank lines and #-comments ignored). Enables fixtures that
        # exercise flags like --helm and --wrap that the auto-detected -data.yaml / -base+-override pattern cannot
        # express. Requires ${name}-template.tmpl (or -full.tmpl) alongside
        args_file="test/fixtures/${name}-args.txt"
        if [[ -f "$args_file" ]]; then
            template="test/fixtures/${name}-template.tmpl"
            if [[ ! -f "$template" ]]; then template="test/fixtures/${name}-full.tmpl"; fi
            if [[ ! -f "$template" ]]; then
                echo "✗ FAIL: $name has args.txt but no template" >&2
                exit 1
            fi
            args=()
            while IFS= read -r line || [[ -n "$line" ]]; do
                [[ -z "$line" || "$line" == \#* ]] && continue
                args+=("$line")
            done < "$args_file"
            echo -n "Testing $name (with args)... " >&2
            if result=$("$bin" "${args[@]}" < "$template") && [[ "$result" == "$(cat "$f")" ]]; then
                echo "✓ PASS" >&2
            else
                echo "✗ FAIL" >&2
                exit 1
            fi
            continue
        fi

        if [[ -f "test/fixtures/${name}-full.tmpl" ]]; then
            template="test/fixtures/${name}-full.tmpl"
            echo -n "Testing $name (embedded)... " >&2
            if result=$("$bin" < "$template") && [[ "$result" == "$(cat "$f")" ]]; then
                echo "✓ PASS" >&2
            else
                echo "✗ FAIL" >&2
                exit 1
            fi
            continue
        fi

        template="test/fixtures/${name}-template.tmpl"
        data="test/fixtures/${name}-data.yaml"
        if [[ ! -f "$data" ]]; then data="test/fixtures/${name}-data.json"; fi
        if [[ ! -f "$data" ]]; then
            base_y="test/fixtures/${name}-base.yaml"
            over_y="test/fixtures/${name}-override.yaml"
            base_j="test/fixtures/${name}-base.json"
            over_j="test/fixtures/${name}-override.json"
            if [[ -f "$base_y" ]] && [[ -f "$over_y" ]]; then
                data="$base_y $over_y"
            elif [[ -f "$base_j" ]] && [[ -f "$over_j" ]]; then
                data="$base_j $over_j"
            fi
        fi
        if [[ -n "$data" ]] && [[ -f "$template" ]]; then
            echo -n "Testing $name... " >&2
            if result=$("$bin" $data < "$template") && [[ "$result" == "$(cat "$f")" ]]; then
                echo "✓ PASS" >&2
            else
                echo "✗ FAIL" >&2
                exit 1
            fi
        fi
    done

# Run unit tests
@test-unit: install
    go test -v ./...

# Update Go dependencies
@update:
    go get -u
    go mod tidy

# Upgrade Golang
upgrade: && update
    #!/usr/bin/env -S bash -Eeuo pipefail
    go get go@latest

# Scan dependencies for known CVEs
vulncheck:
    #!/usr/bin/env -S bash -Eeuo pipefail
    [[ -x "$GOBIN/govulncheck" ]] || go install golang.org/x/vuln/cmd/govulncheck@latest
    "$GOBIN/govulncheck" ./...

# Watch tests
[positional-arguments, no-exit-message]
watch recipe='test' *args:
    #!/usr/bin/env -S bash -Eeuo pipefail
    shift 1
    [[ -x "$(command -v inotifywait)" ]] || { echo "inotifywait not found; install inotify-tools" >&2; exit 1; }

    while true; do
        just "$recipe" "$@" || true
        echo "{{YELLOW}}> watching for changes (ctrl-c to stop){{NORMAL}}" >&2
        inotifywait -r -q -e modify,create,delete,move --exclude '(^|/)\.git(/|$)' .
    done
