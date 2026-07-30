package proxyrepair

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestExecuteAppliesFirstWorkingProxyAndVerifiesOpenAI(t *testing.T) {
	t.Parallel()

	host := &fakeHost{
		currentProxy: "127.0.0.1:10809",
		candidates: []Candidate{
			{Host: "127.0.0.1", Port: 10808, V2RayRelated: true},
			{Host: "127.0.0.1", Port: 10809, V2RayRelated: true},
		},
		probeErrors: map[int]error{10808: errors.New("not an HTTP proxy")},
		verifyCodes: map[string]int{
			"https://chatgpt.com/":             200,
			"https://api.openai.com/v1/models": 401,
		},
		snapshot: SettingsSnapshot{SchemaVersion: "1.0.0"},
	}
	now := time.Date(2026, 7, 30, 8, 0, 0, 0, time.FixedZone("CST", 8*60*60))

	result, err := Execute(context.Background(), host, DefaultOptions(), now)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Proxy.Port != 10809 {
		t.Fatalf("selected proxy port = %d, want 10809", result.Proxy.Port)
	}
	if result.BackupPath != `C:\backup.json` {
		t.Fatalf("backup path = %q, want C:\\backup.json", result.BackupPath)
	}
	if len(result.Checks) != 2 {
		t.Fatalf("verification checks = %d, want 2", len(result.Checks))
	}
	wantCalls := []string{
		"current-proxy",
		"discover",
		"probe:10809",
		"capture",
		"save-backup",
		"apply:10809",
		"refresh",
		"flush-dns",
		"verify:https://chatgpt.com/",
		"verify:https://api.openai.com/v1/models",
	}
	if !reflect.DeepEqual(host.calls, wantCalls) {
		t.Fatalf("calls = %#v, want %#v", host.calls, wantCalls)
	}
}

func TestExecuteRollsBackWhenOpenAIVerificationFails(t *testing.T) {
	t.Parallel()

	host := &fakeHost{
		candidates: []Candidate{
			{Host: "127.0.0.1", Port: 10808, V2RayRelated: true},
		},
		verifyErrors: map[string]error{
			"https://chatgpt.com/":             errors.New("TLS failed"),
			"https://api.openai.com/v1/models": errors.New("connection reset"),
		},
		snapshot: SettingsSnapshot{SchemaVersion: "1.0.0"},
	}

	result, err := Execute(context.Background(), host, DefaultOptions(), time.Now())

	if err == nil {
		t.Fatal("Execute returned nil error, want verification failure")
	}
	if !result.RolledBack {
		t.Fatal("Execute did not report automatic rollback")
	}
	wantTail := []string{"restore", "refresh"}
	if len(host.calls) < len(wantTail) ||
		!reflect.DeepEqual(host.calls[len(host.calls)-len(wantTail):], wantTail) {
		t.Fatalf("call tail = %#v, want %#v", host.calls, wantTail)
	}
}

func TestExecuteRestoresSnapshotWhenApplyFails(t *testing.T) {
	t.Parallel()

	host := &fakeHost{
		candidates: []Candidate{
			{Host: "127.0.0.1", Port: 10808, V2RayRelated: true},
		},
		applyError: errors.New("registry write interrupted"),
		snapshot:   SettingsSnapshot{SchemaVersion: "1.0.0"},
	}

	result, err := Execute(context.Background(), host, DefaultOptions(), time.Now())

	if err == nil {
		t.Fatal("Execute returned nil error, want apply failure")
	}
	if !result.RolledBack {
		t.Fatal("Execute did not report automatic rollback after partial apply")
	}
	wantTail := []string{"apply:10808", "restore", "refresh"}
	if len(host.calls) < len(wantTail) ||
		!reflect.DeepEqual(host.calls[len(host.calls)-len(wantTail):], wantTail) {
		t.Fatalf("call tail = %#v, want %#v", host.calls, wantTail)
	}
}

func TestExecuteRestoresSnapshotWhenRefreshFails(t *testing.T) {
	t.Parallel()

	host := &fakeHost{
		candidates: []Candidate{
			{Host: "127.0.0.1", Port: 10808, V2RayRelated: true},
		},
		refreshErrors: []error{errors.New("WinINet refresh failed"), nil},
		snapshot:      SettingsSnapshot{SchemaVersion: "1.0.0"},
	}

	result, err := Execute(context.Background(), host, DefaultOptions(), time.Now())

	if err == nil {
		t.Fatal("Execute returned nil error, want refresh failure")
	}
	if !result.RolledBack {
		t.Fatal("Execute did not report automatic rollback after refresh failure")
	}
	wantTail := []string{"apply:10808", "refresh", "restore", "refresh"}
	if len(host.calls) < len(wantTail) ||
		!reflect.DeepEqual(host.calls[len(host.calls)-len(wantTail):], wantTail) {
		t.Fatalf("call tail = %#v, want %#v", host.calls, wantTail)
	}
}

func TestRollbackLatestRestoresSavedSettings(t *testing.T) {
	t.Parallel()

	host := &fakeHost{snapshot: SettingsSnapshot{SchemaVersion: "1.0.0"}}

	result, err := RollbackLatest(host)

	if err != nil {
		t.Fatalf("RollbackLatest returned error: %v", err)
	}
	if result.RestoredFrom != `C:\backup.json` {
		t.Fatalf("restored from = %q, want C:\\backup.json", result.RestoredFrom)
	}
	wantCalls := []string{"load-backup", "restore", "refresh", "flush-dns"}
	if !reflect.DeepEqual(host.calls, wantCalls) {
		t.Fatalf("calls = %#v, want %#v", host.calls, wantCalls)
	}
}

type fakeHost struct {
	currentProxy  string
	candidates    []Candidate
	probeErrors   map[int]error
	verifyCodes   map[string]int
	verifyErrors  map[string]error
	applyError    error
	refreshErrors []error
	refreshCount  int
	snapshot      SettingsSnapshot
	calls         []string
}

func (host *fakeHost) CurrentProxy() (string, error) {
	host.calls = append(host.calls, "current-proxy")
	return host.currentProxy, nil
}

func (host *fakeHost) DiscoverCandidates(context.Context) ([]Candidate, error) {
	host.calls = append(host.calls, "discover")
	return host.candidates, nil
}

func (host *fakeHost) Probe(_ context.Context, candidate Candidate) error {
	host.calls = append(host.calls, "probe:"+portText(candidate.Port))
	return host.probeErrors[candidate.Port]
}

func (host *fakeHost) Capture(time.Time) (SettingsSnapshot, error) {
	host.calls = append(host.calls, "capture")
	return host.snapshot, nil
}

func (host *fakeHost) SaveBackup(SettingsSnapshot) (string, error) {
	host.calls = append(host.calls, "save-backup")
	return `C:\backup.json`, nil
}

func (host *fakeHost) Apply(candidate Candidate) error {
	host.calls = append(host.calls, "apply:"+portText(candidate.Port))
	return host.applyError
}

func (host *fakeHost) Refresh() error {
	host.calls = append(host.calls, "refresh")
	if host.refreshCount < len(host.refreshErrors) {
		err := host.refreshErrors[host.refreshCount]
		host.refreshCount++
		return err
	}
	return nil
}

func (host *fakeHost) FlushDNS() error {
	host.calls = append(host.calls, "flush-dns")
	return nil
}

func (host *fakeHost) Verify(_ context.Context, _ Candidate, target string) (int, error) {
	host.calls = append(host.calls, "verify:"+target)
	return host.verifyCodes[target], host.verifyErrors[target]
}

func (host *fakeHost) Restore(SettingsSnapshot) error {
	host.calls = append(host.calls, "restore")
	return nil
}

func (host *fakeHost) LoadLatestBackup() (SettingsSnapshot, string, error) {
	host.calls = append(host.calls, "load-backup")
	return host.snapshot, `C:\backup.json`, nil
}

func portText(port int) string {
	const digits = "0123456789"
	if port == 0 {
		return "0"
	}
	var buf [5]byte
	index := len(buf)
	for port > 0 {
		index--
		buf[index] = digits[port%10]
		port /= 10
	}
	return string(buf[index:])
}
