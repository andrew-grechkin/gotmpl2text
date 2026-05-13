package main

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"gopkg.in/yaml.v3"
)

// customFuncDef represents a custom function definition from YAML
type customFuncDef struct {
	Name     string `yaml:"name"`
	Template string `yaml:"template"`
	Type     string `yaml:"type"` // Optional: string (default), int64, float64, bool
}

type customFuncsConfig struct {
	Functions []customFuncDef `yaml:"functions"`
}

// getCustomFunctionsPath returns the path to custom functions file
// Priority: GOTMPL_FUNCTIONS env -> XDG_CONFIG_HOME -> ~/.config
func getCustomFunctionsPath() string {
	if funcFile := os.Getenv(ENV_FUNCTIONS); funcFile != "" {
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

// loadCustomFunctions loads custom function definitions from config file
func loadCustomFunctions(baseFuncMap template.FuncMap, verbose bool) (template.FuncMap, error) {
	funcFile := getCustomFunctionsPath()
	if funcFile == "" {
		return nil, nil // No config path available
	}

	data, err := os.ReadFile(funcFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No custom functions file, not an error
		}
		return nil, fmt.Errorf("error reading custom functions file: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[debug] Loading custom functions from: %s\n", funcFile)
	}

	var config customFuncsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing custom functions YAML: %w", err)
	}

	customFuncs := make(template.FuncMap)
	for _, fn := range config.Functions {
		funcDef := fn

		// Determine type adapter (default: "string" for backward compatibility)
		funcType := funcDef.Type
		if funcType == "" {
			funcType = "string"
		}

		adapter, ok := typeAdapters[funcType]
		if !ok {
			return nil, fmt.Errorf("unknown type %q for custom function %q (supported: string, int64, float64, bool)",
				funcType, funcDef.Name)
		}

		if verbose {
			if funcType == "string" && funcDef.Type == "" {
				fmt.Fprintf(os.Stderr, "[debug]   - %s (type: %s, default)\n", funcDef.Name, funcType)
			} else {
				fmt.Fprintf(os.Stderr, "[debug]   - %s (type: %s)\n", funcDef.Name, funcType)
			}
		}

		// Create wrapped function with type adapter
		currentFuncDef := funcDef
		currentAdapter := adapter
		customFuncs[funcDef.Name] = func(v any) (any, error) {
			tmpl := template.New("custom_" + currentFuncDef.Name).Funcs(baseFuncMap)
			tmpl, err := tmpl.Parse(currentFuncDef.Template)
			if err != nil {
				return nil, fmt.Errorf("error parsing custom function template %s: %w", currentFuncDef.Name, err)
			}

			buf := new(bytes.Buffer)
			if err := tmpl.Execute(buf, v); err != nil {
				return nil, fmt.Errorf("error executing custom function %s: %w", currentFuncDef.Name, err)
			}

			result, err := currentAdapter(buf.String())
			if err != nil {
				return nil, fmt.Errorf("custom function %q: %w", currentFuncDef.Name, err)
			}
			return result, nil
		}
	}

	return customFuncs, nil
}
