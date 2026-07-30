package tunrepair

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type fakeHost struct {
	processes  []Process
	listeners  []Listener
	configPath string
	config     []byte
	backupPath string
	tunStatus  TunStatus
	verifyErr  error
	startErrs  []error
	excluded   []PortRange

	events     []string
	terminated []uint32
	started    []string
}

func (host *fakeHost) Processes() ([]Process, error) {
	return append([]Process(nil), host.processes...), nil
}

func (host *fakeHost) Listeners() ([]Listener, error) {
	return append([]Listener(nil), host.listeners...), nil
}

func (host *fakeHost) ExcludedPortRanges() ([]PortRange, error) {
	return append([]PortRange(nil), host.excluded...), nil
}

func (host *fakeHost) ReadConfig([]string) (string, []byte, error) {
	if host.configPath == "" {
		return "", nil, errors.New("config not found")
	}
	return host.configPath, append([]byte(nil), host.config...), nil
}

func (host *fakeHost) SaveBackup(path string, data []byte, executable string, _ time.Time) (string, error) {
	host.events = append(host.events, "backup")
	if path != host.configPath || executable == "" || !reflect.DeepEqual(data, host.config) {
		return "", errors.New("unexpected backup input")
	}
	if host.backupPath == "" {
		host.backupPath = "backup.json"
	}
	return host.backupPath, nil
}

func (host *fakeHost) WriteConfig(path string, data []byte) error {
	host.events = append(host.events, "write")
	if path != host.configPath {
		return errors.New("unexpected config path")
	}
	host.config = append([]byte(nil), data...)
	return nil
}

func (host *fakeHost) Terminate(process Process) error {
	host.events = append(host.events, "terminate")
	host.terminated = append(host.terminated, process.PID)
	return nil
}

func (host *fakeHost) WaitStopped(context.Context, []uint32) error {
	host.events = append(host.events, "wait")
	return nil
}

func (host *fakeHost) Start(executable string) error {
	host.events = append(host.events, "start")
	host.started = append(host.started, executable)
	if len(host.startErrs) == 0 {
		return nil
	}
	err := host.startErrs[0]
	host.startErrs = host.startErrs[1:]
	return err
}

func (host *fakeHost) VerifyTun(_ context.Context, localPort int) (TunStatus, error) {
	host.events = append(host.events, "verify")
	status := host.tunStatus
	status.LocalPort = localPort
	return status, host.verifyErr
}

func TestExecuteEnablesTunBacksUpStopsKnownProcessesAndRestartsKeeper(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 30, 8, 30, 0, 0, time.Local)
	host := &fakeHost{
		processes: []Process{
			{PID: 10, Name: "v2rayN.exe", Path: `C:\v2rayN\v2rayN.exe`, StartedAt: now.Add(-time.Hour)},
			{PID: 20, Name: "v2rayN2.exe", Path: `C:\v2rayN\v2rayN2.exe`, StartedAt: now},
			{PID: 30, ParentPID: 10, Name: "sing-box.exe", Path: `C:\v2rayN\bin\sing_box\sing-box.exe`},
			{PID: 40, Name: "database.exe", Path: `C:\Database\database.exe`},
		},
		listeners: []Listener{
			{Address: "127.0.0.1", Port: 10807, PID: 30},
			{Address: "127.0.0.1", Port: 10808, PID: 40},
		},
		configPath: `C:\v2rayN\guiConfigs\guiNConfig.json`,
		config: []byte(`{
  "TunModeItem": {"EnableTun": false},
  "Inbound": [{"LocalPort": 10802, "Protocol": "socks"}]
}`),
		tunStatus: TunStatus{InterfaceName: "singbox_tun", InterfaceUp: true, HTTPSReachable: true},
	}

	result, err := Execute(context.Background(), host, now)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Keeper.PID != 20 || result.ConfigPath != host.configPath || !result.ConfigChanged {
		t.Fatalf("Execute() result = %#v", result)
	}
	if containsUint32(host.terminated, 40) {
		t.Fatalf("unknown process was terminated: %v", host.terminated)
	}
	for _, pid := range []uint32{10, 20, 30} {
		if !containsUint32(host.terminated, pid) {
			t.Fatalf("PID %d was not terminated: %v", pid, host.terminated)
		}
	}
	if got, want := host.started, []string{`C:\v2rayN\v2rayN2.exe`}; !reflect.DeepEqual(got, want) {
		t.Fatalf("started = %#v, want %#v", got, want)
	}
	if got, want := host.events[:2], []string{"backup", "write"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("first events = %#v, want backup before write", got)
	}
}

func TestExecuteRestoresConfigWhenRelaunchFails(t *testing.T) {
	t.Parallel()

	original := []byte(`{"TunModeItem":{"EnableTun":false},"Inbound":[{"LocalPort":10802}]}`)
	host := &fakeHost{
		processes: []Process{
			{PID: 20, Name: "v2rayN2.exe", Path: `C:\v2rayN\v2rayN2.exe`},
		},
		configPath: `C:\v2rayN\guiConfigs\guiNConfig.json`,
		config:     append([]byte(nil), original...),
		startErrs:  []error{errors.New("launch failed"), nil},
	}

	result, err := Execute(context.Background(), host, time.Now())
	if err == nil {
		t.Fatal("Execute() error = nil, want relaunch failure")
	}
	if !result.RolledBack {
		t.Fatalf("Execute() result = %#v, want RolledBack", result)
	}
	if !reflect.DeepEqual(host.config, original) {
		t.Fatalf("config after rollback = %q, want %q", host.config, original)
	}
	if len(host.started) != 2 {
		t.Fatalf("start attempts = %d, want 2", len(host.started))
	}
}

func TestExecuteRelocatesBaseWhenControllerFallsInExcludedRange(t *testing.T) {
	t.Parallel()

	const originalBase = 12000
	host := &fakeHost{
		processes: []Process{
			{PID: 20, Name: "v2rayN.exe", Path: `C:\v2rayN\v2rayN.exe`},
		},
		configPath: `C:\v2rayN\guiConfigs\guiNConfig.json`,
		config: []byte(`{
  "TunModeItem": {"EnableTun": true},
  "Inbound": [{"LocalPort": 12000}]
}`),
		excluded: []PortRange{{Start: originalBase + 6, End: originalBase + 40}},
		tunStatus: TunStatus{
			InterfaceName:  "singbox_tun",
			InterfaceUp:    true,
			HTTPSReachable: true,
			HTTPStatus:     401,
		},
	}

	result, err := Execute(context.Background(), host, time.Now())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.PortChanged || result.PreviousLocalPort != originalBase {
		t.Fatalf("Execute() result = %#v, want recorded port relocation", result)
	}
	selected, err := ConfigLocalPort(host.config)
	if err != nil {
		t.Fatalf("ConfigLocalPort(updated) error = %v", err)
	}
	if selected == originalBase || result.Status.LocalPort != selected {
		t.Fatalf(
			"selected port = %d, status port = %d, original = %d",
			selected,
			result.Status.LocalPort,
			originalBase,
		)
	}
}

func TestExecuteDoesNotMutateOrTerminateWhenConfigCannotBeRead(t *testing.T) {
	t.Parallel()

	host := &fakeHost{
		processes: []Process{
			{PID: 20, Name: "v2rayN2.exe", Path: `C:\v2rayN\v2rayN2.exe`},
		},
	}

	if _, err := Execute(context.Background(), host, time.Now()); err == nil {
		t.Fatal("Execute() error = nil, want config discovery error")
	}
	if len(host.events) != 0 || len(host.terminated) != 0 {
		t.Fatalf("host was mutated before config validation: events=%v terminated=%v", host.events, host.terminated)
	}
}

func containsUint32(values []uint32, value uint32) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
