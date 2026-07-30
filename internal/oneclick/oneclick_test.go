package oneclick

import (
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestArgumentsCreateTimestampedEvidenceOnDesktop(t *testing.T) {
	desktop := filepath.Join(t.TempDir(), "Desktop")
	now := time.Date(2026, 7, 30, 15, 30, 45, 0, time.Local)

	args, output := Arguments(desktop, "PC / 测试:01", now)

	wantOutput := filepath.Join(desktop, "cnd诊断结果-PC-01-20260730-153045.zip")
	if output != wantOutput {
		t.Fatalf("output = %q, want %q", output, wantOutput)
	}
	for _, expected := range []string{"collect", "--provider", "powershell", "--output", wantOutput} {
		if !slices.Contains(args, expected) {
			t.Fatalf("args = %v, missing %q", args, expected)
		}
	}
}

func TestArgumentsUseSafeFallbackForEmptyHostname(t *testing.T) {
	_, output := Arguments(
		t.TempDir(),
		"测试 / :",
		time.Date(2026, 1, 2, 3, 4, 5, 0, time.Local),
	)

	if filepath.Base(output) != "cnd诊断结果-unknown-host-20260102-030405.zip" {
		t.Fatalf("output base = %q", filepath.Base(output))
	}
}

func TestOutputDirectoryPrefersWindowsKnownDesktop(t *testing.T) {
	desktop := filepath.Join(t.TempDir(), "OneDrive", "Desktop")
	got := OutputDirectory(
		func() (string, error) { return desktop, nil },
		filepath.Join(t.TempDir(), "Tools", "cnd.exe"),
	)

	if got != desktop {
		t.Fatalf("OutputDirectory() = %q, want known desktop %q", got, desktop)
	}
}

func TestOutputDirectoryFallsBackToExecutableDirectory(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "Tools", "cnd.exe")
	got := OutputDirectory(
		func() (string, error) { return "", errors.New("known folder unavailable") },
		executable,
	)

	if got != filepath.Dir(executable) {
		t.Fatalf("OutputDirectory() = %q, want %q", got, filepath.Dir(executable))
	}
}
