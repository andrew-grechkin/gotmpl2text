package cli

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
)

// helmDefaultsYAML holds the dummy Release/Chart/Capabilities/Template context injected by --helm. Values are
// intentionally obvious ("release-name", "0.0.0-dummy", ...) so they stand out in rendered output and no one mistakes
// them for real chart metadata. Overridable by any downstream data source (real preload __DATA__, embedded __DATA__,
// data file arg). Method-shaped Helm types (`.Files.Get`, `.Capabilities.APIVersions.Has`, ...) are deliberately not
// modelled - templates that reach for them need `helm template`.
//
//go:embed helm-defaults.yaml
var helmDefaultsYAML string

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

// dataFile pairs a data-file path with the wrap key that was active when the path appeared in the argument list.
// Empty wrapKey means the file is read as-is; otherwise its parsed content is nested under wrapKey before deep-merge.
type dataFile struct {
	path    string
	wrapKey string
}

// parseRunArgs pulls known CLI flags out of the positional argument list and returns them alongside the remaining
// entries, which are treated as data files.
//
// Flags recognised:
//   - `--helm` boolean, position-independent, injects dummy Release/Chart/Capabilities/Template defaults.
//   - `--wrap KEY` / `-w KEY` / `--wrap=KEY` / `-w=KEY` positional: sets the "current wrap key" that gets applied to
//     every data file that follows, until another `--wrap` switches it or the arg list ends.
//
// Empty KEY (`--wrap=` or `-w=`) and a trailing `--wrap` with no key argument are rejected. Files before the first
// `--wrap` are unwrapped.
func parseRunArgs(args []string) (helm bool, files []dataFile, err error) {
	currentWrap := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		var nextWrap string
		wrapSetHere := false
		switch {
		case arg == "--helm":
			helm = true
		case arg == "--wrap" || arg == "-w":
			if i+1 >= len(args) {
				return false, nil, fmt.Errorf("%s requires a key argument", arg)
			}
			nextWrap = args[i+1]
			wrapSetHere = true
			i++
		case strings.HasPrefix(arg, "--wrap="):
			nextWrap = strings.TrimPrefix(arg, "--wrap=")
			wrapSetHere = true
		case strings.HasPrefix(arg, "-w="):
			nextWrap = strings.TrimPrefix(arg, "-w=")
			wrapSetHere = true
		default:
			files = append(files, dataFile{path: arg, wrapKey: currentWrap})
		}
		if wrapSetHere {
			if nextWrap == "" {
				return false, nil, fmt.Errorf("%s requires a non-empty key argument", arg)
			}
			currentWrap = nextWrap
		}
	}
	return helm, files, nil
}

// Run renders the template read from STDIN, merging data from any preload files listed in GOTMPL_PRELOAD and the
// positional data-file arguments, then writes the result to stdout. args is the positional argument list (no program
// name prefix); flags recognised by parseRunArgs are consumed here.
func Run(args []string, stdin io.Reader, stdout io.Writer) error {
	verbose := os.Getenv(ENV_DEBUG) == "1"

	helm, files, err := parseRunArgs(args)
	if err != nil {
		return err
	}

	preloads, err := loadPreloadFiles(verbose)
	if err != nil {
		return err
	}

	tmplContent, data, err := prepareTemplateAndData(stdin, files, preloads, helm, verbose)
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
