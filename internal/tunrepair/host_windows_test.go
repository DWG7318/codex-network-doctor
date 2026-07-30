//go:build windows

package tunrepair

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestWindowsHostProcessesIncludesCurrentExecutablePathAndStartTime(t *testing.T) {
	t.Parallel()

	host, err := NewWindowsHost(t.TempDir())
	if err != nil {
		t.Fatalf("NewWindowsHost() error = %v", err)
	}
	processes, err := host.Processes()
	if err != nil {
		t.Fatalf("Processes() error = %v", err)
	}
	for _, process := range processes {
		if process.PID != uint32(os.Getpid()) {
			continue
		}
		if process.Path == "" {
			t.Fatal("current process path is empty")
		}
		if process.StartedAt.IsZero() {
			t.Fatal("current process start time is zero")
		}
		return
	}
	t.Fatalf("current PID %d not found in process list", os.Getpid())
}

func TestWindowsHostBackupAndAtomicRestoreRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "guiNConfig.json")
	original := []byte(`{"TunModeItem":{"EnableTun":false}}`)
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	host, err := NewWindowsHost(filepath.Join(root, "backups"))
	if err != nil {
		t.Fatalf("NewWindowsHost() error = %v", err)
	}
	backupPath, err := host.SaveBackup(
		configPath,
		original,
		`C:\v2rayN\v2rayN2.exe`,
		time.Date(2026, 7, 30, 8, 40, 0, 0, time.Local),
	)
	if err != nil {
		t.Fatalf("SaveBackup() error = %v", err)
	}
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("backup file stat error = %v", err)
	}
	if err := host.WriteConfig(configPath, []byte(`{"TunModeItem":{"EnableTun":true}}`)); err != nil {
		t.Fatalf("WriteConfig() error = %v", err)
	}
	backup, err := host.LoadLatestBackup()
	if err != nil {
		t.Fatalf("LoadLatestBackup() error = %v", err)
	}
	if backup.ConfigPath != configPath || string(backup.Original) != string(original) {
		t.Fatalf("backup = %#v", backup)
	}
	if err := host.WriteConfig(backup.ConfigPath, backup.Original); err != nil {
		t.Fatalf("restore WriteConfig() error = %v", err)
	}
	restored, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != string(original) {
		t.Fatalf("restored = %q, want %q", restored, original)
	}
}

func TestParseExcludedPortRangesIgnoresHeadersAndAcceptsAdminMarker(t *testing.T) {
	t.Parallel()

	output := []byte(`
Protocol tcp Port Exclusion Ranges

Start Port    End Port
----------    --------
     12000       12099
     20000       20031     *
`)

	got := parseExcludedPortRanges(output)
	want := []PortRange{
		{Start: 12000, End: 12099},
		{Start: 20000, End: 20031},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseExcludedPortRanges() = %#v, want %#v", got, want)
	}
}
