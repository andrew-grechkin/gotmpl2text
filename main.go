package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"text/template"

	"dario.cat/mergo"
	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"
)

// PreloadError represents an error loading preload template files
type PreloadError struct {
	File string
	Err  error
}

func (e *PreloadError) Error() string {
	return fmt.Sprintf("error reading preload template %s: %v", e.File, e.Err)
}

const (
	TEMPLATE_NAME     = "STDIN"
	ENV_ALLOW_MISSING = "GOTMPL_ALLOW_MISSING"
	ENV_IGNORE_EMBED  = "GOTMPL_IGNORE_EMBED"
	ENV_FUNCTIONS     = "GOTMPL_FUNCTIONS"
	ENV_PRELOAD       = "GOTMPL_PRELOAD"
	ENV_DEBUG         = "GOTMPL_DEBUG"
	MISSINGKEY_ERROR  = "missingkey=error"
	MISSINGKEY_ALLOW  = "missingkey=default"
)

//go:embed help.txt
var gnuHelpText []byte

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
	fmt.Fprint(out, string(gnuHelpText))
	return nil
}

func main() {
	if err := run(os.Args, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		if _, ok := err.(*PreloadError); ok {
			os.Exit(2)
		}
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

	verbose := os.Getenv(ENV_DEBUG) == "1"

	preloadFiles, err := parsePreloadTemplatesEnv(verbose)
	if err != nil {
		return err
	}

	tmplContent, data, err := prepareTemplateAndData(stdin, args, verbose)
	if err != nil {
		return err
	}

	tmpl, err := buildTemplate(tmplContent, preloadFiles, verbose)
	if err != nil {
		return err
	}

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
	switch os.Getenv(ENV_IGNORE_EMBED) {
	case "", "0", "false":
		for i, block := range embeddedDataBlocks {
			var blockData map[string]any
			if err := yaml.Unmarshal([]byte(block), &blockData); err != nil {
				return nil, fmt.Errorf("error parsing embedded YAML data block %d: %w", i+1, err)
			}
			allDataMaps = append(allDataMaps, blockData)
		}
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

// indentLines adds padding to non-empty lines in a string
func indentLines(spaces int, v string) string {
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
}

// helmFuncMap returns a FuncMap with Sprig functions plus Helm-specific functions
func helmFuncMap() template.FuncMap {
	funcMap := sprig.TxtFuncMap()

	// Add Helm-specific functions
	helmFuncs := template.FuncMap{
		// include is a stub here and replaced in run() after template parsing
		// because it needs access to the parsed template object (circular dependency)
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
			return "\n" + indentLines(spaces, v)
		},

		"indent": func(spaces int, v string) string {
			return indentLines(spaces, v)
		},
	}

	maps.Copy(funcMap, helmFuncs)
	maps.Copy(funcMap, additionalFuncMap())

	return funcMap
}

// parsePreloadTemplatesEnv splits and returns files specified in GOTMPL_PRELOAD
func parsePreloadTemplatesEnv(verbose bool) ([]string, error) {
	var preloadFiles []string

	preloadEnv := os.Getenv(ENV_PRELOAD)
	if preloadEnv == "" {
		return preloadFiles, nil
	}

	parts := strings.Split(preloadEnv, ",")
	for i := range parts {
		if trimmed := strings.TrimSpace(parts[i]); trimmed != "" {
			preloadFiles = append(preloadFiles, trimmed)
		}
	}

	if verbose && len(preloadFiles) > 0 {
		fmt.Fprintf(os.Stderr, "[debug] Preloading %d template file(s)\n", len(preloadFiles))
		for _, file := range preloadFiles {
			fmt.Fprintf(os.Stderr, "[debug]   - %s\n", file)
		}
	}

	// Verify files exist and convert absolute paths to relative
	var cwd string
	var err error
	for i, file := range preloadFiles {
		if _, err = os.Stat(file); err != nil {
			return preloadFiles, &PreloadError{File: file, Err: err}
		}

		if filepath.IsAbs(file) {
			if cwd == "" {
				cwd, err = os.Getwd()
				if err != nil {
					return preloadFiles, fmt.Errorf("error getting current directory: %w", err)
				}
			}
			relPath, err := filepath.Rel(cwd, file)
			if err != nil {
				return preloadFiles, fmt.Errorf("error converting %s to relative path: %w", file, err)
			}
			preloadFiles[i] = relPath
		}
	}

	return preloadFiles, nil
}

// prepareTemplateAndData reads stdin, processes template and merges data
func prepareTemplateAndData(stdin io.Reader, args []string, verbose bool) (string, map[string]any, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, "[debug] gotmpl2text starting")
	}

	tmplBytes, err := io.ReadAll(stdin)
	if err != nil {
		return "", nil, fmt.Errorf("error reading template from STDIN: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[debug] Read %d bytes from STDIN\n", len(tmplBytes))
	}

	tmplContent, data, err := processTemplate(string(tmplBytes), args)
	if err != nil {
		return "", nil, err
	}

	if verbose {
		dataCount := len(args) - 1
		if dataCount > 0 {
			fmt.Fprintf(os.Stderr, "[debug] Loaded %d data file(s)\n", dataCount)
		}
	}

	return tmplContent, data, nil
}

// getMissingKeyOption returns the template missingkey option based on environment variable
func getMissingKeyOption() string {
	switch os.Getenv(ENV_ALLOW_MISSING) {
	case "", "0", "false":
		return MISSINGKEY_ERROR
	default:
		return MISSINGKEY_ALLOW
	}
}

// buildTemplate builds and parses the template with all function maps
func buildTemplate(tmplContent string, preloadFiles []string, verbose bool) (*template.Template, error) {
	missingKeyOpt := getMissingKeyOption()
	funcMap := helmFuncMap()

	// Load custom functions from XDG config
	if customFuncs, err := loadCustomFunctions(funcMap, verbose); err != nil {
		return nil, fmt.Errorf("error loading custom functions: %w", err)
	} else if customFuncs != nil {
		maps.Copy(funcMap, customFuncs)
	}

	// Create a variable to hold the template reference for the include function
	// and set it after parsing
	var tmplPtr *template.Template
	var err error

	// This MUST be done before creating the template so that all templates
	// (including preload files) have access to the proper include function

	// Replace the stub include function in funcMap with one that uses tmplPtr
	funcMap["include"] = func(name string, data any) (string, error) {
		if tmplPtr == nil {
			return "", fmt.Errorf("template not initialized")
		}
		buf := new(bytes.Buffer)
		err := tmplPtr.ExecuteTemplate(buf, name, data)
		return buf.String(), err
	}

	tmpl := template.New(TEMPLATE_NAME).Funcs(funcMap).Option(missingKeyOpt)

	// I decided that it's better if preload templates can depend on custom functions, not other way around
	// Parse preload template files manually to preserve relative paths as template names
	// This ensures error messages show the relative path (e.g., "dir/file.tmpl:10" instead of just "file.tmpl:10")
	for _, file := range preloadFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("error reading preload template file %s: %w", file, err)
		}
		if _, err = tmpl.New(file).Parse(string(content)); err != nil {
			return nil, fmt.Errorf("error parsing preload template file %s: %w", file, err)
		}
	}

	if tmpl, err = tmpl.Parse(tmplContent); err != nil {
		return nil, fmt.Errorf("error parsing template: %w", err)
	}

	// Rendering engine is initialized, now it is set to make sure include function can work (breaking circular dep)
	tmplPtr = tmpl

	return tmpl, nil
}
