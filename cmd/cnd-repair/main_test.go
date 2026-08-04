package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dwg7318/codex-network-doctor/internal/tunrepair"
)

func TestResolveModeDefaultsToRepair(t *testing.T) {
	t.Parallel()

	mode, err := resolveMode(nil, "cnd网络修复.exe")

	if err != nil {
		t.Fatalf("resolveMode returned error: %v", err)
	}
	if mode != modeRepair {
		t.Fatalf("mode = %q, want repair", mode)
	}
}

func TestResolveModeUsesRollbackForUndoExecutableName(t *testing.T) {
	t.Parallel()

	mode, err := resolveMode(nil, "cnd网络修复-撤销.exe")

	if err != nil {
		t.Fatalf("resolveMode returned error: %v", err)
	}
	if mode != modeRollback {
		t.Fatalf("mode = %q, want rollback", mode)
	}
}

func TestResolveModeSupportsExplicitInspect(t *testing.T) {
	t.Parallel()

	mode, err := resolveMode([]string{"inspect"}, "cnd网络修复.exe")

	if err != nil {
		t.Fatalf("resolveMode returned error: %v", err)
	}
	if mode != modeInspect {
		t.Fatalf("mode = %q, want inspect", mode)
	}
}

func TestResolveModeSupportsExplicitSpeed(t *testing.T) {
	t.Parallel()

	mode, err := resolveMode([]string{"speed"}, "cnd网络修复.exe")

	if err != nil {
		t.Fatalf("resolveMode returned error: %v", err)
	}
	if mode != modeSpeed {
		t.Fatalf("mode = %q, want speed", mode)
	}
}

func TestResolveModeDefaultsSpeedExecutableToReadOnlySpeedMode(t *testing.T) {
	t.Parallel()

	mode, err := resolveMode(nil, "cnd网速检测.exe")

	if err != nil {
		t.Fatalf("resolveMode returned error: %v", err)
	}
	if mode != modeSpeed {
		t.Fatalf("mode = %q, want speed", mode)
	}
}

func TestRunCaseCommandDoesNotRequireRepairMode(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	var errorOutput bytes.Buffer
	exitCode, handled := runCaseCommand(
		[]string{"case", "CND-CASE-2026-001"},
		&output,
		&errorOutput,
	)

	if !handled || exitCode != 0 {
		t.Fatalf(
			"runCaseCommand() = (%d, %v), stderr = %q",
			exitCode,
			handled,
			errorOutput.String(),
		)
	}
	if !strings.Contains(output.String(), "excluded port range") {
		t.Fatalf("output does not contain case root cause:\n%s", output.String())
	}
}

func TestWriteRepairResultDescribesTunDuplicateCleanupAndUnknownOwnerSafety(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	writeRepairResult(&output, tunrepair.Result{
		Keeper: tunrepair.Process{
			PID:  20,
			Name: "v2rayN2.exe",
			Path: `C:\v2rayN\v2rayN2.exe`,
		},
		Duplicates: []tunrepair.Process{
			{PID: 10, Name: "v2rayN.exe"},
		},
		UnknownOwners: []tunrepair.Process{
			{PID: 40, Name: "database.exe"},
		},
		ConfigChanged: true,
		BackupPath:    `C:\backup.json`,
		Status: tunrepair.TunStatus{
			InterfaceName:  "singbox_tun",
			InterfaceUp:    true,
			HTTPSReachable: true,
			HTTPStatus:     401,
		},
	})

	for _, text := range []string{
		"v2rayN2.exe",
		"v2rayN.exe",
		"singbox_tun",
		`C:\backup.json`,
		"HTTP 401",
		"database.exe",
		"未终止",
		"TUN",
	} {
		if !strings.Contains(output.String(), text) {
			t.Errorf("output does not contain %q:\n%s", text, output.String())
		}
	}
}

func TestWriteRepairResultDistinguishesPortOnlyConfigChange(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	writeRepairResult(&output, tunrepair.Result{
		ConfigChanged:     true,
		TunChanged:        false,
		PortChanged:       true,
		PreviousLocalPort: 12000,
		Status: tunrepair.TunStatus{
			LocalPort:      24000,
			InterfaceName:  "singbox_tun",
			InterfaceUp:    true,
			HTTPSReachable: true,
			HTTPStatus:     401,
		},
	})

	for _, required := range []string{"运行前已经启用", "端口修复", "12000", "24000"} {
		if !strings.Contains(output.String(), required) {
			t.Errorf("output does not contain %q:\n%s", required, output.String())
		}
	}
}
