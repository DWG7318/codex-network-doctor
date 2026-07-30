//go:build windows

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dwg7318/codex-network-doctor/internal/tunrepair"
)

func runPlatform(ctx context.Context, mode operationMode, output, errorOutput io.Writer) int {
	host, err := tunrepair.NewWindowsHost("")
	if err != nil {
		_, _ = fmt.Fprintf(errorOutput, "初始化失败：%v\n", err)
		return 1
	}

	switch mode {
	case modeRepair:
		_, _ = fmt.Fprintln(output, "正在检查重复 v2rayN、控制端口冲突和 TUN 配置……")
		runContext, cancel := context.WithTimeout(ctx, 55*time.Second)
		defer cancel()
		result, repairErr := tunrepair.Execute(runContext, host, time.Now())
		if repairErr != nil {
			_, _ = fmt.Fprintf(errorOutput, "\n修复未完成：%v\n", repairErr)
			if result.BackupPath != "" {
				_, _ = fmt.Fprintf(errorOutput, "配置备份：%s\n", result.BackupPath)
			}
			if result.RolledBack {
				_, _ = fmt.Fprintln(errorOutput, "重新启动失败，已恢复运行前的 TUN 设置。")
			}
			for _, owner := range result.UnknownOwners {
				_, _ = fmt.Fprintf(
					errorOutput,
					"未终止未知端口占用者：%s（PID %d）\n",
					owner.Name,
					owner.PID,
				)
			}
			_, _ = fmt.Fprintln(errorOutput, "请保留窗口内容或重新运行 cnd.exe 生成“cnd诊断结果”发回。")
			return 1
		}
		writeRepairResult(output, result)
		return 0

	case modeRollback:
		_, _ = fmt.Fprintln(output, "正在恢复最近一次修复前的 v2rayN TUN 配置……")
		runContext, cancel := context.WithTimeout(ctx, 25*time.Second)
		defer cancel()
		backup, rollbackErr := host.RestoreLatest(runContext)
		if rollbackErr != nil {
			_, _ = fmt.Fprintf(errorOutput, "恢复失败：%v\n", rollbackErr)
			return 1
		}
		_, _ = fmt.Fprintf(output, "恢复完成：%s\n", backup.ConfigPath)
		_, _ = fmt.Fprintf(output, "已重新启动：%s\n", backup.Executable)
		return 0

	case modeInspect:
		return inspectTun(host, output, errorOutput)

	default:
		_, _ = fmt.Fprintf(errorOutput, "不支持的操作：%s\n", mode)
		return 2
	}
}

func inspectTun(host *tunrepair.WindowsHost, output, errorOutput io.Writer) int {
	processes, err := host.Processes()
	if err != nil {
		_, _ = fmt.Fprintf(errorOutput, "读取进程失败：%v\n", err)
		return 1
	}
	keeper, duplicates, err := tunrepair.SelectKeeper(processes)
	if err != nil {
		_, _ = fmt.Fprintf(errorOutput, "%v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(output, "将保留：%s（PID %d，%s）\n", keeper.Name, keeper.PID, keeper.Path)
	for _, duplicate := range duplicates {
		_, _ = fmt.Fprintf(output, "重复实例：%s（PID %d，%s）\n", duplicate.Name, duplicate.PID, duplicate.Path)
	}
	path, content, err := host.ReadConfig(
		tunrepair.CandidateConfigPaths(keeper.Path, os.Getenv("LOCALAPPDATA")),
	)
	if err != nil {
		_, _ = fmt.Fprintf(errorOutput, "读取配置失败：%v\n", err)
		return 1
	}
	_, changed, err := tunrepair.EnableTunConfig(content)
	if err != nil {
		_, _ = fmt.Fprintf(errorOutput, "检查 TUN 设置失败：%v\n", err)
		return 1
	}
	state := "已开启"
	if changed {
		state = "未开启"
	}
	_, _ = fmt.Fprintf(output, "配置：%s\nTUN：%s\n", path, state)
	localPort, err := tunrepair.ConfigLocalPort(content)
	if err != nil {
		_, _ = fmt.Fprintf(errorOutput, "读取本机端口失败：%v\n", err)
		return 1
	}
	listeners, err := host.Listeners()
	if err != nil {
		_, _ = fmt.Fprintf(errorOutput, "读取监听端口失败：%v\n", err)
		return 1
	}
	plan := tunrepair.PlanTermination(
		processes,
		listeners,
		keeper,
		[]int{localPort + 5, localPort + 6},
	)
	for _, owner := range plan.UnknownOwners {
		_, _ = fmt.Fprintf(
			output,
			"未知控制端口占用者（不会终止）：%s（PID %d）\n",
			owner.Name,
			owner.PID,
		)
	}
	return 0
}
