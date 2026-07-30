package main

import (
	"fmt"
	"io"

	"github.com/dwg7318/codex-network-doctor/internal/tunrepair"
)

func writeRepairResult(output io.Writer, result tunrepair.Result) {
	_, _ = fmt.Fprintln(output, "\n修复成功。")
	_, _ = fmt.Fprintf(
		output,
		"已保留并重新启动：%s（重启前 PID %d，%s）\n",
		result.Keeper.Name,
		result.Keeper.PID,
		result.Keeper.Path,
	)
	if result.TunChanged {
		_, _ = fmt.Fprintln(output, "TUN 设置：已写入启用。")
	} else {
		_, _ = fmt.Fprintln(output, "TUN 设置：运行前已经启用。")
	}
	if result.PortChanged {
		_, _ = fmt.Fprintf(
			output,
			"端口修复：原基础端口 %d 位于 Windows 排除范围，已迁移到 %d。\n",
			result.PreviousLocalPort,
			result.Status.LocalPort,
		)
	}
	for _, duplicate := range result.Duplicates {
		_, _ = fmt.Fprintf(output, "已清理重复实例：%s（PID %d）\n", duplicate.Name, duplicate.PID)
	}
	_, _ = fmt.Fprintf(output, "TUN 网卡：%s（已启动）\n", result.Status.InterfaceName)
	if result.Status.LocalListener {
		_, _ = fmt.Fprintf(output, "本机 Mixed 监听：127.0.0.1:%d\n", result.Status.LocalPort)
	}
	_, _ = fmt.Fprintf(output, "OpenAI HTTPS 验证：HTTP %d\n", result.Status.HTTPStatus)
	_, _ = fmt.Fprintf(output, "回滚备份：%s\n", result.BackupPath)
	for _, owner := range result.UnknownOwners {
		_, _ = fmt.Fprintf(
			output,
			"安全提示：端口占用者 %s（PID %d）不是 v2rayN，未终止；当前 TUN 验证已成功。\n",
			owner.Name,
			owner.PID,
		)
	}
	_, _ = fmt.Fprintln(output, "\nTUN 已接管系统流量；现在可重新打开浏览器和 Codex。")
}
