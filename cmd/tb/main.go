package main

import (
	"os"

	"github.com/szdytom/tb/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:]))
}
