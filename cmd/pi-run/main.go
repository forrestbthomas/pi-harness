package main

import (
	"os"

	"github.com/forrestthomas/harness/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
