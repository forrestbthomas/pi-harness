package main

import (
	"os"

	"github.com/forrestbthomas/pi-harness/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
