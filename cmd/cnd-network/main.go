package main

import (
	"context"
	"os"

	"github.com/dwg7318/codex-network-doctor/internal/networkcli"
)

var version = "0.1.0-dev"

func main() {
	os.Exit(networkcli.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr, version))
}
