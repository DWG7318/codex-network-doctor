package powershell

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/dwg7318/codex-network-doctor/internal/evidence"
)

type fakeRunner struct {
	stdout []byte
	stderr []byte
	code   int
	err    error
	name   string
	args   []string
}

func (r *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, int, error) {
	r.name = name
	r.args = append([]string(nil), args...)
	return r.stdout, r.stderr, r.code, r.err
}

func TestCollectorExecutesNonInteractiveScriptAndPreservesJSON(t *testing.T) {
	store, err := evidence.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("evidence.NewStore() error = %v", err)
	}
	runner := &fakeRunner{stdout: []byte("[]\r\n")}
	script := Script{
		ID:        "routes",
		Category:  "route",
		Body:      "$items = @(Get-NetRoute); ConvertTo-Json -InputObject $items -Compress",
		Timeout:   time.Second,
		Mode:      "passive",
		MediaType: "application/json",
	}
	collector := NewCollector("powershell.exe", script, runner)

	artifacts, err := collector.Collect(context.Background(), store)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if runner.name != "powershell.exe" {
		t.Fatalf("runner name = %q", runner.name)
	}
	for _, requiredArg := range []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command", script.Body} {
		if !slices.Contains(runner.args, requiredArg) {
			t.Fatalf("runner args = %v, missing %q", runner.args, requiredArg)
		}
	}
	if slices.Contains(runner.args, "-ExecutionPolicy") {
		t.Fatalf("runner args unexpectedly override execution policy: %v", runner.args)
	}
	if len(artifacts) != 1 || artifacts[0].Path != "evidence/windows/routes.json" {
		t.Fatalf("artifacts = %+v", artifacts)
	}
	data, err := os.ReadFile(filepath.Join(store.Root(), "evidence", "windows", "routes.json"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != string(runner.stdout) {
		t.Fatalf("stored JSON = %q, want %q", data, runner.stdout)
	}
}

func TestDefaultScriptsAreUniqueAndPassive(t *testing.T) {
	scripts := DefaultScripts()
	if len(scripts) < 10 {
		t.Fatalf("DefaultScripts() count = %d, want at least 10", len(scripts))
	}
	seen := make(map[string]struct{}, len(scripts))
	for _, script := range scripts {
		if _, exists := seen[script.ID]; exists {
			t.Fatalf("duplicate script ID %q", script.ID)
		}
		seen[script.ID] = struct{}{}
		if script.Mode != "passive" {
			t.Fatalf("script %q mode = %q, want passive", script.ID, script.Mode)
		}
		if script.Timeout <= 0 {
			t.Fatalf("script %q timeout = %s, want positive", script.ID, script.Timeout)
		}
		for _, forbidden := range []string{"Test-NetConnection", "Invoke-WebRequest", "Invoke-RestMethod", "Start-BitsTransfer", "Set-", "New-Net", "Remove-", "Enable-", "Disable-"} {
			if strings.Contains(script.Body, forbidden) {
				t.Fatalf("script %q contains active or mutating command %q", script.ID, forbidden)
			}
		}
	}
}

func TestCertificateScriptUsesReadOnlyX509StoreWithoutCertPSDrive(t *testing.T) {
	for _, script := range DefaultScripts() {
		if script.ID != "certificates" {
			continue
		}
		if strings.Contains(script.Body, `Cert:\`) {
			t.Fatal("certificate script depends on the optional Cert: PowerShell drive")
		}
		if !strings.Contains(script.Body, "OpenFlags]::ReadOnly") {
			t.Fatal("certificate script does not explicitly open X509 stores read-only")
		}
		return
	}
	t.Fatal("certificate script not found")
}
