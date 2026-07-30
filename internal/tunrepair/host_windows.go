//go:build windows

package tunrepair

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	tcpTableOwnerPIDAll = 5
	mibTCPStateListen   = 2
	afINET              = 2
	tcpOwnerPIDRowSize  = 24
)

var getExtendedTCPTable = windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetExtendedTcpTable")
var excludedPortLine = regexp.MustCompile(`(?m)^\s*(\d+)\s+(\d+)(?:\s+\*)?\s*$`)

type WindowsHost struct {
	backupDirectory string
}

type Backup struct {
	SchemaVersion int       `json:"schema_version"`
	CreatedAt     time.Time `json:"created_at"`
	ConfigPath    string    `json:"config_path"`
	Executable    string    `json:"executable"`
	Original      []byte    `json:"original"`
	SHA256        string    `json:"sha256"`
}

func NewWindowsHost(backupDirectory string) (*WindowsHost, error) {
	if backupDirectory == "" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			var err error
			base, err = os.UserConfigDir()
			if err != nil {
				return nil, fmt.Errorf("定位用户配置目录：%w", err)
			}
		}
		backupDirectory = filepath.Join(base, "CND", "tun-repair", "backups")
	}
	if err := os.MkdirAll(backupDirectory, 0o700); err != nil {
		return nil, fmt.Errorf("创建备份目录：%w", err)
	}
	return &WindowsHost{backupDirectory: backupDirectory}, nil
}

func (host *WindowsHost) Processes() ([]Process, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot: %w", err)
	}
	defer func() { _ = windows.CloseHandle(snapshot) }()

	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return nil, fmt.Errorf("Process32First: %w", err)
	}
	var processes []Process
	for {
		process := Process{
			PID:       entry.ProcessID,
			ParentPID: entry.ParentProcessID,
			Name:      windows.UTF16ToString(entry.ExeFile[:]),
		}
		if detail, detailErr := queryProcessDetail(entry.ProcessID); detailErr == nil {
			process.Path = detail.Path
			process.StartedAt = detail.StartedAt
		}
		processes = append(processes, process)
		err := windows.Process32Next(snapshot, &entry)
		if err == nil {
			continue
		}
		if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
			break
		}
		return nil, fmt.Errorf("Process32Next: %w", err)
	}
	return processes, nil
}

func (host *WindowsHost) Listeners() ([]Listener, error) {
	var size uint32
	result, _, _ := getExtendedTCPTable.Call(
		0,
		uintptr(unsafe.Pointer(&size)),
		0,
		afINET,
		tcpTableOwnerPIDAll,
		0,
	)
	if result != uintptr(windows.ERROR_INSUFFICIENT_BUFFER) && result != 0 {
		return nil, fmt.Errorf("GetExtendedTcpTable size query: %w", syscall.Errno(result))
	}
	if size < 4 {
		return nil, fmt.Errorf("GetExtendedTcpTable returned invalid size %d", size)
	}
	buffer := make([]byte, size)
	result, _, _ = getExtendedTCPTable.Call(
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
		afINET,
		tcpTableOwnerPIDAll,
		0,
	)
	if result != 0 {
		return nil, fmt.Errorf("GetExtendedTcpTable: %w", syscall.Errno(result))
	}
	rowCount := int(binary.LittleEndian.Uint32(buffer[:4]))
	if 4+(rowCount*tcpOwnerPIDRowSize) > len(buffer) {
		return nil, errors.New("GetExtendedTcpTable returned truncated data")
	}
	listeners := make([]Listener, 0, rowCount)
	for index := 0; index < rowCount; index++ {
		offset := 4 + index*tcpOwnerPIDRowSize
		if binary.LittleEndian.Uint32(buffer[offset:offset+4]) != mibTCPStateListen {
			continue
		}
		listeners = append(listeners, Listener{
			Address: net.IP(append([]byte(nil), buffer[offset+4:offset+8]...)).String(),
			Port:    int(binary.BigEndian.Uint16(buffer[offset+8 : offset+10])),
			PID:     binary.LittleEndian.Uint32(buffer[offset+20 : offset+24]),
		})
	}
	return listeners, nil
}

func (host *WindowsHost) ExcludedPortRanges() ([]PortRange, error) {
	var combined []PortRange
	succeeded := 0
	var failures []string
	for _, family := range []string{"ipv4", "ipv6"} {
		for _, protocol := range []string{"tcp", "udp"} {
			command := exec.Command(
				"netsh",
				"interface",
				family,
				"show",
				"excludedportrange",
				"protocol="+protocol,
			)
			output, err := command.CombinedOutput()
			if err != nil {
				failures = append(
					failures,
					fmt.Sprintf("%s/%s: %v", family, protocol, err),
				)
				continue
			}
			succeeded++
			combined = append(combined, parseExcludedPortRanges(output)...)
		}
	}
	if succeeded == 0 {
		return nil, fmt.Errorf("netsh 查询全部失败（%s）", strings.Join(failures, "；"))
	}

	seen := make(map[PortRange]bool, len(combined))
	unique := make([]PortRange, 0, len(combined))
	for _, item := range combined {
		if seen[item] {
			continue
		}
		seen[item] = true
		unique = append(unique, item)
	}
	sort.Slice(unique, func(i, j int) bool {
		if unique[i].Start == unique[j].Start {
			return unique[i].End < unique[j].End
		}
		return unique[i].Start < unique[j].Start
	})
	return unique, nil
}

func parseExcludedPortRanges(output []byte) []PortRange {
	matches := excludedPortLine.FindAllSubmatch(output, -1)
	ranges := make([]PortRange, 0, len(matches))
	for _, match := range matches {
		start, startErr := strconv.Atoi(string(match[1]))
		end, endErr := strconv.Atoi(string(match[2]))
		if startErr != nil || endErr != nil || start < 1 || end < start || end > 65535 {
			continue
		}
		ranges = append(ranges, PortRange{Start: start, End: end})
	}
	return ranges
}

func (host *WindowsHost) ReadConfig(candidates []string) (string, []byte, error) {
	ordered := append([]string(nil), candidates...)
	if len(ordered) >= 2 {
		portableRoot := filepath.Dir(filepath.Dir(ordered[0]))
		_, markerErr := os.Stat(filepath.Join(portableRoot, "NotStoreConfigHere.txt"))
		if markerErr == nil || (strings.Contains(portableRoot, "Users") && strings.Contains(portableRoot, "WinGet")) {
			ordered[0], ordered[1] = ordered[1], ordered[0]
		}
	}
	var checked []string
	for _, candidate := range ordered {
		if candidate == "" {
			continue
		}
		checked = append(checked, candidate)
		content, err := os.ReadFile(candidate)
		if err == nil {
			return candidate, content, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", nil, fmt.Errorf("读取 %s：%w", candidate, err)
		}
	}
	return "", nil, fmt.Errorf("未找到 guiNConfig.json（已检查 %s）", strings.Join(checked, "、"))
}

func (host *WindowsHost) SaveBackup(
	path string,
	data []byte,
	executable string,
	now time.Time,
) (string, error) {
	digest := sha256.Sum256(data)
	backup := Backup{
		SchemaVersion: 1,
		CreatedAt:     now,
		ConfigPath:    path,
		Executable:    executable,
		Original:      append([]byte(nil), data...),
		SHA256:        hex.EncodeToString(digest[:]),
	}
	encoded, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化备份：%w", err)
	}
	fileName := "cnd-tun-backup-" + now.Format("20060102-150405.000") + ".json"
	backupPath := filepath.Join(host.backupDirectory, fileName)
	if err := atomicWriteFile(backupPath, encoded, 0o600); err != nil {
		return "", err
	}
	if err := atomicWriteFile(filepath.Join(host.backupDirectory, "latest.json"), encoded, 0o600); err != nil {
		return "", err
	}
	return backupPath, nil
}

func (host *WindowsHost) LoadLatestBackup() (Backup, error) {
	var backup Backup
	content, err := os.ReadFile(filepath.Join(host.backupDirectory, "latest.json"))
	if err != nil {
		return backup, fmt.Errorf("读取最近备份：%w", err)
	}
	if err := json.Unmarshal(content, &backup); err != nil {
		return backup, fmt.Errorf("解析最近备份：%w", err)
	}
	if backup.SchemaVersion != 1 || backup.ConfigPath == "" || backup.Executable == "" {
		return backup, errors.New("最近备份格式无效")
	}
	digest := sha256.Sum256(backup.Original)
	if !strings.EqualFold(backup.SHA256, hex.EncodeToString(digest[:])) {
		return backup, errors.New("最近备份校验失败")
	}
	return backup, nil
}

func (host *WindowsHost) WriteConfig(path string, data []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("读取配置属性：%w", err)
	}
	return atomicWriteFile(path, data, info.Mode().Perm())
}

func (host *WindowsHost) Terminate(planned Process) error {
	if planned.PID <= 4 || planned.PID == uint32(os.Getpid()) || planned.Path == "" {
		return errors.New("拒绝终止未通过身份校验的进程")
	}
	current, err := queryProcessDetail(planned.PID)
	if err != nil {
		if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
			return nil
		}
		return err
	}
	if !strings.EqualFold(filepath.Clean(current.Path), filepath.Clean(planned.Path)) {
		return errors.New("PID 已被其他程序复用")
	}
	if !planned.StartedAt.IsZero() && !current.StartedAt.Equal(planned.StartedAt) {
		return errors.New("PID 启动时间发生变化")
	}
	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.SYNCHRONIZE, false, planned.PID)
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(handle) }()
	if err := windows.TerminateProcess(handle, 0); err != nil {
		return err
	}
	return nil
}

func (host *WindowsHost) WaitStopped(ctx context.Context, pids []uint32) error {
	if len(pids) == 0 {
		return nil
	}
	waitContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	targets := make(map[uint32]bool, len(pids))
	for _, pid := range pids {
		targets[pid] = true
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		processes, err := host.Processes()
		if err != nil {
			return err
		}
		for pid := range targets {
			delete(targets, pid)
		}
		for _, process := range processes {
			for _, pid := range pids {
				if process.PID == pid {
					targets[pid] = true
				}
			}
		}
		if len(targets) == 0 {
			return nil
		}
		select {
		case <-waitContext.Done():
			return fmt.Errorf("仍有 %d 个进程未退出", len(targets))
		case <-ticker.C:
		}
	}
}

func (host *WindowsHost) Start(executable string) error {
	if executable == "" {
		return errors.New("v2rayN 路径为空")
	}
	command := exec.Command(executable)
	command.Dir = filepath.Dir(executable)
	if err := command.Start(); err != nil {
		return err
	}
	return command.Process.Release()
}

func (host *WindowsHost) VerifyTun(ctx context.Context, localPort int) (TunStatus, error) {
	verifyContext, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()
	status := TunStatus{LocalPort: localPort}
	var lastErr error
	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	for {
		status.InterfaceName, status.InterfaceUp = findTunInterface()
		status.LocalListener = host.localPortListening(localPort)
		if status.InterfaceUp {
			status.HTTPStatus, lastErr = probeHTTPSViaTun(verifyContext)
			if lastErr == nil {
				status.HTTPSReachable = true
				return status, nil
			}
		}
		select {
		case <-verifyContext.Done():
			if lastErr != nil {
				return status, lastErr
			}
			return status, verifyContext.Err()
		case <-ticker.C:
		}
	}
}

func (host *WindowsHost) localPortListening(port int) bool {
	listeners, err := host.Listeners()
	if err != nil {
		return false
	}
	for _, listener := range listeners {
		address := net.ParseIP(listener.Address)
		if listener.Port == port &&
			address != nil &&
			(address.IsLoopback() || address.IsUnspecified()) {
			return true
		}
	}
	return false
}

func (host *WindowsHost) RestoreLatest(ctx context.Context) (Backup, error) {
	backup, err := host.LoadLatestBackup()
	if err != nil {
		return backup, err
	}
	processes, err := host.Processes()
	if err != nil {
		return backup, err
	}
	keeper := Process{Name: filepath.Base(backup.Executable), Path: backup.Executable}
	plan := PlanTermination(processes, nil, keeper, nil)
	var pids []uint32
	for _, process := range plan.Terminate {
		if err := host.Terminate(process); err != nil {
			return backup, fmt.Errorf("停止 %s (PID %d)：%w", process.Name, process.PID, err)
		}
		pids = append(pids, process.PID)
	}
	if err := host.WaitStopped(ctx, pids); err != nil {
		return backup, err
	}
	if err := host.WriteConfig(backup.ConfigPath, backup.Original); err != nil {
		return backup, fmt.Errorf("恢复配置：%w", err)
	}
	if err := host.Start(backup.Executable); err != nil {
		return backup, fmt.Errorf("重新启动 v2rayN：%w", err)
	}
	return backup, nil
}

type processDetail struct {
	Path      string
	StartedAt time.Time
}

func queryProcessDetail(pid uint32) (processDetail, error) {
	var detail processDetail
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return detail, err
	}
	defer func() { _ = windows.CloseHandle(handle) }()
	buffer := make([]uint16, 32768)
	size := uint32(len(buffer))
	if err := windows.QueryFullProcessImageName(handle, 0, &buffer[0], &size); err != nil {
		return detail, err
	}
	detail.Path = windows.UTF16ToString(buffer[:size])
	var created, exited, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &created, &exited, &kernel, &user); err == nil {
		detail.StartedAt = time.Unix(0, created.Nanoseconds())
	}
	return detail, nil
}

func atomicWriteFile(path string, data []byte, mode os.FileMode) (returnErr error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".cnd-tun-*.tmp")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	defer func() {
		_ = file.Close()
		if returnErr != nil {
			_ = os.Remove(tempPath)
		}
	}()
	if err := file.Chmod(mode); err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := windows.Rename(tempPath, path); err != nil {
		return err
	}
	return nil
}

func findTunInterface() (string, bool) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", false
	}
	for _, networkInterface := range interfaces {
		name := strings.ToLower(networkInterface.Name)
		if name != "singbox_tun" && !strings.Contains(name, "singbox") {
			continue
		}
		return networkInterface.Name, networkInterface.Flags&net.FlagUp != 0
	}
	return "", false
}

func probeHTTPSViaTun(ctx context.Context) (int, error) {
	requestContext, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	transport := &http.Transport{
		Proxy:               nil,
		DisableKeepAlives:   true,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: 6 * time.Second,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}
	request, err := http.NewRequestWithContext(
		requestContext,
		http.MethodGet,
		"https://api.openai.com/v1/models",
		nil,
	)
	if err != nil {
		return 0, err
	}
	request.Header.Set("User-Agent", "CND-TUN-Repair/1")
	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer func() { _ = response.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	return response.StatusCode, nil
}
