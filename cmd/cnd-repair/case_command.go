package main

import (
	"io"
	"strings"

	"github.com/dwg7318/codex-network-doctor/internal/casebook"
)

func runCaseCommand(args []string, output, errorOutput io.Writer) (int, bool) {
	if len(args) == 0 {
		return 0, false
	}
	command := strings.ToLower(strings.TrimSpace(args[0]))
	if command != "case" && command != "cases" {
		return 0, false
	}
	return casebook.Run(args, output, errorOutput), true
}
