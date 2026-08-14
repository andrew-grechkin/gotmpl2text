package custom

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

const EnvFunctions = "GOTMPL_FUNCTIONS"

// typeAdapter converts string output from template to target type
type typeAdapter func(string) (any, error)

// typeAdapters maps type names to conversion functions
var typeAdapters = map[string]typeAdapter{
	"string": func(s string) (any, error) {
		return s, nil
	},

	"int64": func(s string) (any, error) {
		trimmed := strings.TrimSpace(s)
		i, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to int64: %w", s, err)
		}
		return i, nil
	},

	"float64": func(s string) (any, error) {
		trimmed := strings.TrimSpace(s)
		f, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to float64: %w", s, err)
		}
		return f, nil
	},

	"bool": func(s string) (any, error) {
		trimmed := strings.TrimSpace(s)
		b, err := strconv.ParseBool(trimmed)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to bool: %w", s, err)
		}
		return b, nil
	},
}

// funcDef represents a custom function definition from YAML
type funcDef struct {
	Name     string `yaml:"name"`
	Template string `yaml:"template"`
	Type     string `yaml:"type"` // Optional: string (default), int64, float64, bool
}

type funcsConfig struct {
	Functions []funcDef `yaml:"functions"`
}

// getPath returns the path to custom functions file
// Priority: GOTMPL_FUNCTIONS env -> XDG_CONFIG_HOME -> ~/.config
func getPath() string {
	if funcFile := os.Getenv(EnvFunctions); funcFile != "" {
		return funcFile
	}

	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return configHome + "/gotmpl2text/functions.yaml"
	}

	if home := os.Getenv("HOME"); home != "" {
		return home + "/.config/gotmpl2text/functions.yaml"
	}

	return ""
}

// Load loads custom function definitions from config file. Returns an empty
// FuncMap (never nil) if no config path is resolvable or the file is missing;
// this lets callers unconditionally merge without a nil check.
//
// baseFuncMap is captured by every returned function's closure and is read
// LAZILY at invocation time - so callers may mutate it (e.g. bind `include`
// after template parsing) and custom functions will see the updated map.
func Load(baseFuncMap template.FuncMap, verbose bool) (template.FuncMap, error) {
	funcFile := getPath()
	if funcFile == "" {
		return template.FuncMap{}, nil
	}

	data, err := os.ReadFile(funcFile)
	if err != nil {
		if os.IsNotExist(err) {
			return template.FuncMap{}, nil
		}
		return nil, fmt.Errorf("error reading custom functions file: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[debug] Loading custom functions from: %s\n", funcFile)
	}

	var config funcsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing custom functions YAML: %w", err)
	}

	customFuncs := make(template.FuncMap)
	for _, fn := range config.Functions {
		// Determine type adapter (default: "string" for backward compatibility)
		funcType := fn.Type
		if funcType == "" {
			funcType = "string"
		}

		adapter, ok := typeAdapters[funcType]
		if !ok {
			return nil, fmt.Errorf("unknown type %q for custom function %q (supported: string, int64, float64, bool)",
				funcType, fn.Name)
		}

		if verbose {
			suffix := ""
			if fn.Type == "" {
				suffix = ", default"
			}
			fmt.Fprintf(os.Stderr, "[debug]   - %s (type: %s%s)\n", fn.Name, funcType, suffix)
		}

		customFuncs[fn.Name] = func(v any) (any, error) {
			tmpl := template.New("custom_" + fn.Name).Funcs(baseFuncMap)
			tmpl, err := tmpl.Parse(fn.Template)
			if err != nil {
				return nil, fmt.Errorf("error parsing custom function template %s: %w", fn.Name, err)
			}

			buf := new(bytes.Buffer)
			if err := tmpl.Execute(buf, v); err != nil {
				return nil, fmt.Errorf("error executing custom function %s: %w", fn.Name, err)
			}

			result, err := adapter(buf.String())
			if err != nil {
				return nil, fmt.Errorf("custom function %q: %w", fn.Name, err)
			}
			return result, nil
		}
	}

	return customFuncs, nil
}
