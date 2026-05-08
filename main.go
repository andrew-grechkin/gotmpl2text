package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"text/template"

	"dario.cat/mergo"
	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"
)

const (
	TEMPLATE_NAME     = "STDIN"
	ENV_ALLOW_MISSING = "GOTMPL_ALLOW_MISSING"
	MISSINGKEY_ERROR  = "missingkey=error"
	MISSINGKEY_ALLOW  = "missingkey=default"
)

const gnuHelpText = `Usage: gotmpl2text [OPTIONS] [FILE...]

A CLI filter for testing and rendering Go templates with YAML/JSON data.

Reads template from STDIN, outputs rendered result to STDOUT.

Data can be provided via:
  - Embedded in template as {{/* __DATA__ ... */}} comment block(s) (optional)
  - Command-line arguments (YAML/JSON files, optional)
  - Both combined
  - No data at all (for self-contained templates using Sprig functions)

Multiple data sources are merged in order:
  1. Embedded blocks are merged first (top-to-bottom, later overrides earlier)
  2. Data files are merged on top (left-to-right, files always override embedded data)
This matches Helm's behavior: gotmpl2text base.yaml override.yaml

Options:
  -h, --help        Display this help message
  -m, --man         Display full readme         (tip: gotmpl2text --man | colored-md)
  -v, --version     Display version information (tip: gotmpl2text --version | jq -r .Version)

Environment:
  GOTMPL_ALLOW_MISSING  Set to 1 to allow missing keys (renders <no value>)

Examples:
  gotmpl2text < template.tmpl base.yaml overrides.yaml

  gotmpl2text <<< '{{ .name }}' <(echo '{"name":"test"}')

  gotmpl2text <<'EO_TEMPLATE'
Hello {{ .name }}!

{{/* __DATA__
name: world
*/}}
EO_TEMPLATE
`

//go:embed README.md
var readmeContent []byte

func printReadme(out io.Writer) error {
	fmt.Fprint(out, string(readmeContent))
	return nil
}

func printVersion(out io.Writer) error {
	if info, ok := debug.ReadBuildInfo(); ok {
		output, _ := json.MarshalIndent(info.Main, "", "  ")
		fmt.Fprintln(out, string(output))
	} else {
		fmt.Fprintln(out, "{}")
	}
	return nil
}

func printHelp(out io.Writer) error {
	fmt.Fprint(out, gnuHelpText)
	return nil
}

func main() {
	if err := run(os.Args, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 2 {
		arg := args[1]
		switch arg {
		case "--version", "-v":
			return printVersion(stdout)
		case "--man", "-m":
			return printReadme(stdout)
		case "--help", "-h":
			return printHelp(stdout)
		}
	}

	tmplBytes, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("error reading template from STDIN: %w", err)
	}

	tmplContent, data, err := processTemplate(string(tmplBytes), args)
	if err != nil {
		return err
	}

	missingKeyOpt := MISSINGKEY_ALLOW
	switch os.Getenv(ENV_ALLOW_MISSING) {
	case "", "0", "false":
		missingKeyOpt = MISSINGKEY_ERROR
	}

	tmpl := template.New(TEMPLATE_NAME).Funcs(helmFuncMap()).Option(missingKeyOpt)
	if tmpl, err = tmpl.Parse(tmplContent); err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}

	tmpl.Funcs(template.FuncMap{
		"include": func(name string, data any) (string, error) {
			buf := new(bytes.Buffer)
			err := tmpl.ExecuteTemplate(buf, name, data)
			return buf.String(), err
		},
	})

	if err := tmpl.Execute(stdout, data); err != nil {
		return err
	}

	return nil
}

func processTemplate(template string, args []string) (string, map[string]any, error) {
	tmplContent, embeddedDataBlocks := splitTemplateData(template)

	var allDataMaps []map[string]any
	var err error

	if allDataMaps, err = collectDataFromEmbeddedBlocks(allDataMaps, embeddedDataBlocks); err != nil {
		return "", nil, err
	}

	if allDataMaps, err = collectDataFromFiles(allDataMaps, args); err != nil {
		return "", nil, err
	}

	var data map[string]any
	if data, err = mergeAllDataMaps(allDataMaps); err != nil {
		return "", nil, err
	}

	// If no data provided, use empty map (allows self-contained templates using only sprig functions)
	if data == nil {
		data = make(map[string]any)
	}

	return tmplContent, data, nil
}

func collectDataFromEmbeddedBlocks(allDataMaps []map[string]any, embeddedDataBlocks []string) ([]map[string]any, error) {
	for i, block := range embeddedDataBlocks {
		var blockData map[string]any
		if err := yaml.Unmarshal([]byte(block), &blockData); err != nil {
			return nil, fmt.Errorf("error parsing embedded YAML data block %d: %w", i+1, err)
		}
		allDataMaps = append(allDataMaps, blockData)
	}
	return allDataMaps, nil
}

func collectDataFromFiles(allDataMaps []map[string]any, args []string) ([]map[string]any, error) {
	dataFiles := args[1:]

	for _, dataFile := range dataFiles {
		dataBytes, err := os.ReadFile(dataFile)
		if err != nil {
			return nil, fmt.Errorf("error reading data file %s: %w", dataFile, err)
		}

		var fileData map[string]any
		if err := yaml.Unmarshal(dataBytes, &fileData); err != nil {
			return nil, fmt.Errorf("error parsing YAML data from %s: %w", dataFile, err)
		}
		allDataMaps = append(allDataMaps, fileData)
	}

	return allDataMaps, nil
}

func mergeAllDataMaps(allDataMaps []map[string]any) (map[string]any, error) {
	var data map[string]any
	for _, dataMap := range allDataMaps {
		if data == nil {
			data = dataMap
		} else {
			if err := mergo.Merge(&data, dataMap, mergo.WithOverride); err != nil {
				return nil, fmt.Errorf("error merging data: %w", err)
			}
		}
	}
	return data, nil
}

// splitTemplateData splits template content from embedded data sections
// Returns (tmplText, dataBlocks) where dataBlocks contains all embedded YAML blocks
// Looks for ALL {{/* __DATA__ */}} blocks and extracts them in order
func splitTemplateData(content string) (string, []string) {
	// Go's regexp package shamefully doesn't support the (?x) free-spacing flag,
	// so have to use string concatenation instead.
	re := regexp.MustCompile(
		`\n*` + // optional new lines
			`\{\{/\*` + // match opening comment: {{/*
			`\s*` + // optional whitespace
			`__DATA__` + // match the expected data marker
			`\s*` + // optional whitespace
			`([\s\S]*?)` + // group 1: Capture the actual data (non-greedy)
			`\*/\}\}` + // match closing comment: */}}
			`\n*`, // optional new lines
	)

	var dataBlocks []string
	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			dataBlocks = append(dataBlocks, strings.TrimSpace(match[1]))
		}
	}

	tmplText := re.ReplaceAllString(content, "\n")
	return tmplText, dataBlocks
}

// helmFuncMap returns a FuncMap with Sprig functions plus Helm-specific functions
func helmFuncMap() template.FuncMap {
	funcMap := sprig.TxtFuncMap()

	// Add Helm-specific functions
	helmFuncs := template.FuncMap{
		"include": func(name string, data any) (string, error) {
			return "", fmt.Errorf("include function not properly initialized")
		},

		"required": func(msg string, val any) (any, error) {
			if val == nil {
				return nil, fmt.Errorf("required value not provided: %s", msg)
			}
			if str, ok := val.(string); ok && str == "" {
				return nil, fmt.Errorf("required value not provided: %s", msg)
			}
			return val, nil
		},

		"toYaml": func(v any) string {
			data, err := yaml.Marshal(v)
			if err != nil {
				return ""
			}
			return strings.TrimSuffix(string(data), "\n")
		},

		"fromYaml": func(str string) any {
			var v any
			if err := yaml.Unmarshal([]byte(str), &v); err != nil {
				return map[string]any{}
			}
			return v
		},

		"nindent": func(spaces int, v string) string {
			padding := strings.Repeat(" ", spaces)
			lines := strings.Split(v, "\n")
			var result []string
			for _, line := range lines {
				if line != "" {
					result = append(result, padding+line)
				} else {
					result = append(result, line)
				}
			}
			return "\n" + strings.Join(result, "\n")
		},

		"indent": func(spaces int, v string) string {
			padding := strings.Repeat(" ", spaces)
			lines := strings.Split(v, "\n")
			var result []string
			for _, line := range lines {
				if line != "" {
					result = append(result, padding+line)
				} else {
					result = append(result, line)
				}
			}
			return strings.Join(result, "\n")
		},
	}

	maps.Copy(funcMap, helmFuncs)

	return funcMap
}
