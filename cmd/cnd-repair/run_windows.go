//go:build windows

package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/dwg7318/codex-network-doctor/internal/proxyrepair"
	"github.com/dwg7318/codex-network-doctor/internal/speedtest"
	"github.com/dwg7318/codex-network-doctor/internal/tunrepair"
)

func runPlatform(ctx context.Context, mode operationMode, output, errorOutput io.Writer) int {
	if mode == modeSpeed {
		return runSpeedDiagnostics(ctx, output, errorOutput)
	}
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

func runSpeedDiagnostics(ctx context.Context, output, errorOutput io.Writer) int {
	host, err := proxyrepair.NewWindowsHost("")
	if err != nil {
		_, _ = fmt.Fprintf(errorOutput, "初始化只读代理发现失败：%v\n", err)
		return 1
	}
	defer func() { _ = host.Close() }()
	currentProxy, err := host.CurrentProxy()
	if err != nil {
		_, _ = fmt.Fprintf(errorOutput, "读取系统代理失败：%v\n", err)
		return 1
	}
	candidates, err := host.DiscoverCandidates(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(errorOutput, "发现本地代理端口失败：%v\n", err)
		return 1
	}
	ranked := proxyrepair.RankCandidates(candidates, currentProxy)
	if len(ranked) == 0 {
		_, _ = fmt.Fprintln(errorOutput, "未发现 v2rayN/sing-box 本地代理监听端口。")
		return 1
	}
	selected, err := selectSpeedCandidate(ctx, ranked, host.Probe)
	if err != nil {
		_, _ = fmt.Fprintf(errorOutput, "未发现可用的本地 HTTP/mixed 代理端口：%v\n", err)
		return 1
	}
	endpoint := net.JoinHostPort(selected.Host, fmt.Sprintf("%d", selected.Port))
	_, _ = fmt.Fprintln(output, "主动测速已显式启用；不会修改代理、TUN、DNS、路由或进程。")
	_, _ = fmt.Fprintf(output, "本地 mixed 候选：%s（%s，PID %d）\n", endpoint, selected.ProcessName, selected.PID)
	_, _ = fmt.Fprintln(output, "system_path 在 TUN 开启时代表 TUN 路径，关闭时代表直接系统路径。")

	runContext, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	report := runSpeedPlan(
		runContext,
		speedPlan{
			Endpoint:  endpoint,
			TargetURL: "https://speed.cloudflare.com/__down?bytes=1000000",
			Attempts:  3,
			Timeout:   25 * time.Second,
			Delay:     250 * time.Millisecond,
		},
		speedtest.Measure,
		output,
	)
	successes := 0
	for _, summary := range report.Summaries {
		successes += summary.SuccessfulSamples
	}
	if successes == 0 {
		_, _ = fmt.Fprintln(errorOutput, "全部测速路径失败；请检查节点连通性后重试。")
		return 1
	}
	return 0
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
