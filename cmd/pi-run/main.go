package main

import (
	"os"

	"github.com/forrestthomas1/pi-harness/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
