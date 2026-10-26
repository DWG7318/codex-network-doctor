package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/dwg7318/codex-network-doctor/internal/networkcli"
)

var version = "0.1.0-dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := networkcli.Run(ctx, os.Args[1:], os.Stdout, os.Stderr, version)
	stop()
	os.Exit(code)
}
