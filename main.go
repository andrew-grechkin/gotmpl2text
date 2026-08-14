package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os"

	"github.com/andrew-grechkin/gotmpl2text/internal/cli"
)

//go:embed help.txt
var helpText []byte

//go:embed README.md
var readmeContent []byte

func main() {
	if handled, err := cli.HandleFlags(os.Args, os.Stdout, helpText, readmeContent); handled {
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := cli.Run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		var pe *cli.PreloadError
		if errors.As(err, &pe) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
