package tunrepair

import (
	"context"
	"fmt"
	"os"
	"time"
)

type TunStatus struct {
	InterfaceName  string
	InterfaceUp    bool
	LocalPort      int
	LocalListener  bool
	HTTPSReachable bool
	HTTPStatus     int
}

type Result struct {
	Keeper            Process
	Duplicates        []Process
	UnknownOwners     []Process
	ConfigPath        string
	BackupPath        string
	ConfigChanged     bool
	TunChanged        bool
	PortChanged       bool
	PreviousLocalPort int
	RolledBack        bool
	Status            TunStatus
}

type excludedPortHost interface {
	ExcludedPortRanges() ([]PortRange, error)
}

type Host interface {
	Processes() ([]Process, error)
	Listeners() ([]Listener, error)
	ReadConfig(candidates []string) (string, []byte, error)
	SaveBackup(path string, data []byte, executable string, now time.Time) (string, error)
	WriteConfig(path string, data []byte) error
	Terminate(process Process) error
	WaitStopped(ctx context.Context, pids []uint32) error
	Start(executable string) error
	VerifyTun(ctx context.Context, localPort int) (TunStatus, error)
}

func Execute(ctx context.Context, host Host, now time.Time) (Result, error) {
	var result Result
	processes, err := host.Processes()
	if err != nil {
		return result, fmt.Errorf("读取进程列表：%w", err)
	}
	keeper, duplicates, err := SelectKeeper(processes)
	if err != nil {
		return result, err
	}
	result.Keeper = keeper
	result.Duplicates = duplicates

	candidates := CandidateConfigPaths(keeper.Path, os.Getenv("LOCALAPPDATA"))
	configPath, original, err := host.ReadConfig(candidates)
	if err != nil {
		return result, fmt.Errorf("读取 v2rayN 配置：%w", err)
	}
	result.ConfigPath = configPath
	updated, changed, err := EnableTunConfig(original)
	if err != nil {
		return result, err
	}
	result.TunChanged = changed
	localPort, err := ConfigLocalPort(updated)
	if err != nil {
		return result, err
	}
	listeners, err := host.Listeners()
	if err != nil {
		return result, fmt.Errorf("读取监听端口：%w", err)
	}
	plan := PlanTermination(processes, listeners, keeper, []int{localPort + 5, localPort + 6})
	result.UnknownOwners = plan.UnknownOwners

	if portHost, supported := host.(excludedPortHost); supported {
		excluded, rangeErr := portHost.ExcludedPortRanges()
		if rangeErr != nil {
			return result, fmt.Errorf("读取 Windows 排除端口范围：%w", rangeErr)
		}
		selectedPort, portChanged, selectErr := ChooseBasePort(
			localPort,
			excluded,
			listeners,
			20000,
			45000,
		)
		if selectErr != nil {
			return result, selectErr
		}
		if portChanged {
			updated, _, err = SetConfigLocalPort(updated, selectedPort)
			if err != nil {
				return result, err
			}
			result.PortChanged = true
			result.PreviousLocalPort = localPort
			localPort = selectedPort
			changed = true
		}
	}

	backupPath, err := host.SaveBackup(configPath, original, keeper.Path, now)
	if err != nil {
		return result, fmt.Errorf("备份 v2rayN 配置：%w", err)
	}
	result.BackupPath = backupPath
	if changed {
		if err := host.WriteConfig(configPath, updated); err != nil {
			return result, fmt.Errorf("启用 TUN：%w", err)
		}
		result.ConfigChanged = true
	}

	pids := make([]uint32, 0, len(plan.Terminate))
	for _, process := range plan.Terminate {
		if err := host.Terminate(process); err != nil {
			return result, fmt.Errorf("停止 %s (PID %d)：%w", process.Name, process.PID, err)
		}
		pids = append(pids, process.PID)
	}
	if err := host.WaitStopped(ctx, pids); err != nil {
		return result, fmt.Errorf("等待 v2rayN 停止：%w", err)
	}
	if err := host.Start(keeper.Path); err != nil {
		if changed {
			if restoreErr := host.WriteConfig(configPath, original); restoreErr == nil {
				result.RolledBack = true
			}
		}
		_ = host.Start(keeper.Path)
		return result, fmt.Errorf("重新启动 v2rayN：%w", err)
	}
	status, err := host.VerifyTun(ctx, localPort)
	result.Status = status
	if err != nil {
		return result, fmt.Errorf("验证 TUN：%w", err)
	}
	if !status.InterfaceUp {
		return result, fmt.Errorf("验证 TUN：未发现已启用的 TUN 网卡")
	}
	if !status.HTTPSReachable {
		return result, fmt.Errorf("验证 TUN：HTTPS 仍不可达")
	}
	return result, nil
}
