#!/usr/bin/env -S just --one --justfile

# Build the binary to cache directory
@build:
    go build -o "$XDG_CACHE_HOME/go/bin/"

# Install the binary globally
@install:
    go install gotmpl2text

# Format Go source code
@fix:
    go fmt

# Run Go linter
@lint:
    go vet

# Update Go dependencies
@update:
    go get -u
    go mod tidy

# Run Go unit tests
@test-unit:
    go test -v ./...

# Run all tests
test: build test-unit
    #!/usr/bin/env bash
    set -Eeuo pipefail

    # Unset environment variables that could interfere with tests
    unset GOTMPL_PRELOAD GOTMPL_FUNCTIONS || true

    for f in test/fixtures/*-expected.txt; do
        name=$(basename "$f" -expected.txt)

        if [[ -f "test/fixtures/${name}-full.tmpl" ]]; then
            template="test/fixtures/${name}-full.tmpl"
            echo -n "Testing $name (embedded)... " >&2
            if result=$(gotmpl2text < "$template") && [[ "$result" == "$(cat "$f")" ]]; then
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
            if result=$(gotmpl2text $data < "$template") && [[ "$result" == "$(cat "$f")" ]]; then
                echo "✓ PASS" >&2
            else
                echo "✗ FAIL" >&2
                exit 1
            fi
        fi
    done
