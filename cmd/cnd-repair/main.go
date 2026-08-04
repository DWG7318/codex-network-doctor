package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
)

var version = "0.1.0-dev"

func main() {
	if exitCode, handled := runCaseCommand(os.Args[1:], os.Stdout, os.Stderr); handled {
		pauseIfInteractive()
		os.Exit(exitCode)
	}

	executablePath, err := os.Executable()
	if err != nil {
		executablePath = os.Args[0]
	}
	mode, err := resolveMode(os.Args[1:], executablePath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "参数错误：%v\n", err)
		pauseIfInteractive()
		os.Exit(2)
	}
	relaunched, err := ensurePlatformElevation(mode)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "请求管理员权限失败：%v\n", err)
		pauseIfInteractive()
		os.Exit(1)
	}
	if relaunched {
		return
	}

	title := "CND v2rayN TUN 网络修复器"
	if mode == modeSpeed {
		title = "CND 网络路径测速工具"
	}
	_, _ = fmt.Fprintf(os.Stdout, "%s %s\n", title, version)
	exitCode := runPlatform(context.Background(), mode, os.Stdout, os.Stderr)
	pauseIfInteractive()
	os.Exit(exitCode)
}

func pauseIfInteractive() {
	info, err := os.Stdin.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return
	}
	_, _ = fmt.Fprintln(os.Stdout, "\n按回车键关闭窗口。")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}
