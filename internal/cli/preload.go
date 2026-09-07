package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// holds a preload template's on-disk path (converted to project-relative so template parse errors report
// "dir/file.tmpl:10" rather than a full absolute path) alongside its already-read content
type preloadFile struct {
	path    string
	content string
}

// reads and validates each file listed in GOTMPL_PRELOAD. Files are read once here and their content is reused by both
// data-block extraction and template parsing downstream. Returns nil on empty env, (nil, *PreloadError) on the first
// unreadable file, and normalises absolute paths to project-relative for clean error messages
func loadPreloadFiles(verbose bool) ([]preloadFile, error) {
	env := os.Getenv(ENV_PRELOAD)
	if env == "" {
		return nil, nil
	}

	var paths []string
	for part := range strings.SplitSeq(env, preloadSeparator) {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			paths = append(paths, trimmed)
		}
	}
	if len(paths) == 0 {
		return nil, nil
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[debug] Preloading %d template file(s)\n", len(paths))
		for _, p := range paths {
			fmt.Fprintf(os.Stderr, "[debug]   - %s\n", p)
		}
	}

	var cwd string
	files := make([]preloadFile, 0, len(paths))
	for _, p := range paths {
		content, err := os.ReadFile(p)
		if err != nil {
			return nil, &PreloadError{File: p, Err: err}
		}

		name := p
		if filepath.IsAbs(p) {
			if cwd == "" {
				cwd, err = os.Getwd()
				if err != nil {
					return nil, fmt.Errorf("error getting current directory: %w", err)
				}
			}
			rel, err := filepath.Rel(cwd, p)
			if err != nil {
				return nil, fmt.Errorf("error converting %s to relative path: %w", p, err)
			}
			name = rel
		}

		files = append(files, preloadFile{path: name, content: string(content)})
	}

	return files, nil
}

// reads STDIN and merges data from embedded __DATA__ blocks (both in preload files and STDIN), then data files given on
// argv. Per-file wrapping (set positionally via --wrap on the command line) is baked into each dataFile at parse time.
// When helm is true, dummy Release/Chart/Capabilities/Template context is prepended to the preload __DATA__ slot so it
// acts as defaults overridable by any downstream data source.
func prepareTemplateAndData(stdin io.Reader, files []dataFile, preloads []preloadFile, helm, verbose bool) (string, map[string]any, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, "[debug] gotmpl2text starting")
		if helm {
			fmt.Fprintln(os.Stderr, "[debug] Injecting --helm dummy Release/Chart/Capabilities/Template context")
		}
	}

	tmplBytes, err := io.ReadAll(stdin)
	if err != nil {
		return "", nil, fmt.Errorf("error reading template from STDIN: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[debug] Read %d bytes from STDIN\n", len(tmplBytes))
	}

	var preloadDataBlocks []string
	if helm {
		preloadDataBlocks = append(preloadDataBlocks, helmDefaultsYAML)
	}
	for _, p := range preloads {
		_, blocks := splitTemplateData(p.content)
		preloadDataBlocks = append(preloadDataBlocks, blocks...)
	}

	tmplContent, data, err := processTemplate(string(tmplBytes), preloadDataBlocks, files)
	if err != nil {
		return "", nil, err
	}

	if verbose && len(files) > 0 {
		fmt.Fprintf(os.Stderr, "[debug] Loaded %d data file(s)\n", len(files))
	}

	return tmplContent, data, nil
}
