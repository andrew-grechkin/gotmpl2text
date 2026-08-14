// Package host exposes machine / OS information (hostname, os, arch)
package host

import (
	"os"
	"runtime"
	"text/template"
)

// All are snapshots of the invoking environment at render time
func FuncMap() template.FuncMap {
	return template.FuncMap{
		"hostname": os.Hostname,
		"os":       func() string { return runtime.GOOS },
		"arch":     func() string { return runtime.GOARCH },
	}
}
