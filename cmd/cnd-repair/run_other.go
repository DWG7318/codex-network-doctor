//go:build !windows

package main

import (
	"context"
	"fmt"
	"io"
)

func runPlatform(_ context.Context, _ operationMode, _ io.Writer, errorOutput io.Writer) int {
	_, _ = fmt.Fprintln(errorOutput, "CND v2rayN 网络修复器仅支持 Windows。")
	return 1
}
