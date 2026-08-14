package cli

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"text/template"

	"github.com/andrew-grechkin/gotmpl2text/pkg/functions/additional"
	"github.com/andrew-grechkin/gotmpl2text/pkg/functions/custom"
	"github.com/andrew-grechkin/gotmpl2text/pkg/functions/helm"
)

// returns the template missingkey option based on environment variable
func getMissingKeyOption() string {
	switch os.Getenv(ENV_ALLOW_MISSING) {
	case "", "0", "false":
		return missingKeyError
	default:
		return missingKeyAllow
	}
}

// builds and parses the template with all function maps. preloads must contain content already read from disk (see
// loadPreloadFiles); the path field is used verbatim as the template name so parse errors report project-relative
// locations
func buildTemplate(tmplContent string, preloads []preloadFile, verbose bool) (*template.Template, error) {
	missingKeyOpt := getMissingKeyOption()
	funcMap := helm.FuncMap()
	maps.Copy(funcMap, additional.FuncMap())

	// Load custom functions from XDG config
	customFuncs, err := custom.Load(funcMap, verbose)
	if err != nil {
		return nil, fmt.Errorf("error loading custom functions: %w", err)
	}
	maps.Copy(funcMap, customFuncs)

	// The include function needs the parsed *template.Template to look up named templates, but template.Funcs() must be
	// called BEFORE Parse(). We resolve the cycle by putting a placeholder in funcMap now, then mutating the same
	// funcMap entry after Parse() to close over the parsed template. Template functions are looked up at execution
	// time, so this late binding works
	var tmplPtr *template.Template
	funcMap["include"] = func(name string, data any) (string, error) {
		if tmplPtr == nil {
			return "", fmt.Errorf("template not initialized")
		}
		buf := new(bytes.Buffer)
		err := tmplPtr.ExecuteTemplate(buf, name, data)
		return buf.String(), err
	}

	tmpl := template.New(templateName).Funcs(funcMap).Option(missingKeyOpt)

	// Preloads are parsed first so the main template (and any customFuncs it uses) can reference their definitions
	for _, p := range preloads {
		if _, err := tmpl.New(p.path).Parse(p.content); err != nil {
			return nil, fmt.Errorf("error parsing preload template file %s: %w", p.path, err)
		}
	}

	if tmpl, err = tmpl.Parse(tmplContent); err != nil {
		return nil, fmt.Errorf("error parsing template: %w", err)
	}

	tmplPtr = tmpl // late-bind include, breaking the circular dependency above

	return tmpl, nil
}
