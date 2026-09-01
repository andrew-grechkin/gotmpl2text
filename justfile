#!/usr/bin/env -S just --one --justfile

export tool := 'gotmpl2text'

# Build the binary to cache directory
@build: fix
    go build -o "$XDG_CACHE_HOME/go/bin/"

# Install the binary globally
@install:
    go install "$tool"

# Format Go source code
@fix:
    go fmt
    go fix

# Run Go linter
@lint: cc
    go vet

# Report functions over cyclomatic complexity 15 (installs gocyclo if missing)
@cc:
    test -x "$XDG_CACHE_HOME/go/bin/gocyclo" || GOBIN="$XDG_CACHE_HOME/go/bin" go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
    "$XDG_CACHE_HOME/go/bin/gocyclo" -over 15 .

# Update Go dependencies
@update:
    go get -u
    go mod tidy

# Run Go unit tests
@test-unit:
    go test -v ./...

# Run integration tests by driving the binary against fixtures
test-int: build
    #!/usr/bin/env -S bash -Eeuo pipefail

    bin="${GOBIN:-${GOPATH:-$HOME/go}/bin}/$tool"

    # Unset environment variables that could interfere with tests
    unset GOTMPL_PRELOAD GOTMPL_FUNCTIONS || true

    for f in test/fixtures/*-expected.txt; do
        name=$(basename "$f" -expected.txt)

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

# Run all tests
test: lint test-unit test-int
