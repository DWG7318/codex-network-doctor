package providers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareAutoPrefersExplicitOsquery(t *testing.T) {
	root := t.TempDir()
	osqueryPath := filepath.Join(root, "osqueryi.exe")
	powershellPath := filepath.Join(root, "powershell.exe")
	writeExecutablePlaceholder(t, osqueryPath)
	writeExecutablePlaceholder(t, powershellPath)

	selection, err := Prepare(Options{
		Provider:       "auto",
		OsqueryPath:    osqueryPath,
		PowerShellPath: powershellPath,
	})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if selection.Provider != "osquery" {
		t.Fatalf("provider = %q, want osquery", selection.Provider)
	}
	if len(selection.Collectors) == 0 {
		t.Fatal("collectors are empty")
	}
}

func TestPrepareAutoFallsBackToExplicitPowerShell(t *testing.T) {
	powershellPath := filepath.Join(t.TempDir(), "powershell.exe")
	writeExecutablePlaceholder(t, powershellPath)

	selection, err := Prepare(Options{
		Provider:       "auto",
		OsqueryPath:    filepath.Join(t.TempDir(), "missing-osqueryi.exe"),
		PowerShellPath: powershellPath,
	})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if selection.Provider != "windows-powershell" {
		t.Fatalf("provider = %q, want windows-powershell", selection.Provider)
	}
}

func TestPrepareExplicitOsqueryRejectsMissingBinary(t *testing.T) {
	_, err := Prepare(Options{
		Provider:    "osquery",
		OsqueryPath: filepath.Join(t.TempDir(), "missing-osqueryi.exe"),
	})
	if err == nil {
		t.Fatal("Prepare() error = nil, want missing binary error")
	}
}

func writeExecutablePlaceholder(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("placeholder"), 0o700); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}
