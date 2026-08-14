package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime/debug"
)

// PreloadError represents an error loading preload template files
type PreloadError struct {
	File string
	Err  error
}

func (e *PreloadError) Error() string {
	return fmt.Sprintf("error reading preload template %s: %v", e.File, e.Err)
}

// Shell env var names (uppercase to match the actual environment variables).
const (
	ENV_ALLOW_MISSING = "GOTMPL_ALLOW_MISSING"
	ENV_IGNORE_EMBED  = "GOTMPL_IGNORE_EMBED"
	ENV_PRELOAD       = "GOTMPL_PRELOAD"
	ENV_DEBUG         = "GOTMPL_DEBUG"
)

// Internal constants (Go MixedCaps).
const (
	templateName     = "STDIN"
	missingKeyError  = "missingkey=error"
	missingKeyAllow  = "missingkey=default"
	preloadSeparator = string(os.PathListSeparator) // ":" on Unix, ";" on Windows
)

func printReadme(out io.Writer, readme []byte) error {
	_, err := fmt.Fprint(out, string(readme))
	return err
}

func printVersion(out io.Writer) error {
	if info, ok := debug.ReadBuildInfo(); ok {
		output, err := json.MarshalIndent(info.Main, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshalling build info: %w", err)
		}
		_, err = fmt.Fprintln(out, string(output))
		return err
	}
	_, err := fmt.Fprintln(out, "{}")
	return err
}

func printHelp(out io.Writer, help []byte) error {
	_, err := fmt.Fprint(out, string(help))
	return err
}

// HandleFlags dispatches --help/--man/--version and returns handled=true if one of them matched. The caller should skip
// Run when handled is true. Both handled and err can be non-zero: the flag matched but the write failed.
func HandleFlags(args []string, stdout io.Writer, help, readme []byte) (handled bool, err error) {
	if len(args) != 2 {
		return false, nil
	}
	switch args[1] {
	case "--version", "-v":
		return true, printVersion(stdout)
	case "--man", "-m":
		return true, printReadme(stdout, readme)
	case "--help", "-h":
		return true, printHelp(stdout, help)
	}
	return false, nil
}

// Run renders the template read from STDIN, merging data from dataFiles and any preload files listed in GOTMPL_PRELOAD,
// then writes the result to stdout.
func Run(dataFiles []string, stdin io.Reader, stdout io.Writer) error {
	verbose := os.Getenv(ENV_DEBUG) == "1"

	preloads, err := loadPreloadFiles(verbose)
	if err != nil {
		return err
	}

	tmplContent, data, err := prepareTemplateAndData(stdin, dataFiles, preloads, verbose)
	if err != nil {
		return err
	}

	tmpl, err := buildTemplate(tmplContent, preloads, verbose)
	if err != nil {
		return err
	}

	return formatTemplateError(tmpl.Execute(stdout, data))
}

// matches the wrapper text/template inserts between frames when a user function returns an error. Global replace splits
// the chain into one line per frame so each line stays parseable by vim errorformat and friends (`template: %f:%l:%c:
// %m`)
var templateFrameSep = regexp.MustCompile(`: error calling ([^:]+): template: `)

// reshapes Go's flattened template error chain into one-frame-per-line while preserving the leading `template:` marker
// on every line. No-op for nil and for errors that don't contain the wrapper
func formatTemplateError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	reformatted := templateFrameSep.ReplaceAllString(msg, ": error calling $1\ntemplate: ")
	if reformatted == msg {
		return err
	}
	return errors.New(reformatted)
}
