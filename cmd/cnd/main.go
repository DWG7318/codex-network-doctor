package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dwg7318/codex-network-doctor/internal/cli"
	"github.com/dwg7318/codex-network-doctor/internal/oneclick"
)

var version = "0.1.0-dev"

func main() {
	executablePath, err := os.Executable()
	if err != nil {
		executablePath = os.Args[0]
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}
	outputDirectory := oneclick.OutputDirectory(oneclick.DesktopDirectory, executablePath)
	args, directClick, outputPath := resolveInvocation(
		os.Args[1:],
		outputDirectory,
		hostname,
		time.Now(),
	)
	if directClick {
		_, _ = fmt.Fprintln(os.Stdout, "Codex Network Doctor 正在采集网络证据，请勿关闭此窗口……")
		_, _ = fmt.Fprintf(os.Stdout, "输出文件：%s\n\n", outputPath)
	}

	exitCode := cli.Run(context.Background(), args, os.Stdout, os.Stderr, version)
	if directClick {
		if exitCode == 0 {
			_, _ = fmt.Fprintf(os.Stdout, "\n采集完成：%s\n", outputPath)
		} else {
			_, _ = fmt.Fprintln(os.Stderr, "\n采集失败；上方输出包含具体错误。")
		}
		pauseIfInteractive()
	}
	os.Exit(exitCode)
}

func resolveInvocation(args []string, outputDirectory, hostname string, now time.Time) ([]string, bool, string) {
	if len(args) != 0 {
		return args, false, ""
	}
	resolved, output := oneclick.Arguments(outputDirectory, hostname, now)
	return resolved, true, output
}

func pauseIfInteractive() {
	info, err := os.Stdin.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return
	}
	_, _ = fmt.Fprintln(os.Stdout, "\n按回车键关闭窗口。")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}
