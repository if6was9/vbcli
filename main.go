package main

import (
	"fmt"
	"os"

	"vbcli/cmd"
)

func main() {
	if err := cmd.NewRootCmd(os.Stdin, os.Stdout, os.Stderr).Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
