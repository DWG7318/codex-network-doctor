package osquery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/dwg7318/codex-network-doctor/internal/evidence"
)

type fakeRunner struct {
	stdout   []byte
	stderr   []byte
	exitCode int
	err      error
	name     string
	args     []string
}

func (r *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, int, error) {
	r.name = name
	r.args = append([]string(nil), args...)
	return r.stdout, r.stderr, r.exitCode, r.err
}

func TestCollectorRunsReadOnlyQueryAndPreservesRawJSON(t *testing.T) {
	store, err := evidence.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("evidence.NewStore() error = %v", err)
	}
	runner := &fakeRunner{stdout: []byte("[{\"hostname\":\"test-host\"}]\n")}
	collector := NewCollector(
		`C:\tools\osqueryi.exe`,
		Query{ID: "system_info", Category: "system", SQL: "SELECT hostname FROM system_info;", Timeout: 3 * time.Second},
		runner,
	)

	artifacts, err := collector.Collect(context.Background(), store)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if runner.name != `C:\tools\osqueryi.exe` {
		t.Fatalf("runner name = %q", runner.name)
	}
	for _, requiredArg := range []string{
		"--json",
		"--disable_events=true",
		"--disable_extensions=true",
		"--disable_logging=true",
		"--logger_stderr=false",
		"--utc=true",
		"SELECT hostname FROM system_info;",
	} {
		if !slices.Contains(runner.args, requiredArg) {
			t.Fatalf("runner args = %v, missing %q", runner.args, requiredArg)
		}
	}
	if len(artifacts) != 1 || artifacts[0].Path != "evidence/osquery/system_info.json" {
		t.Fatalf("artifacts = %+v", artifacts)
	}
	data, err := os.ReadFile(filepath.Join(store.Root(), "evidence", "osquery", "system_info.json"))
	if err != nil {
		t.Fatalf("ReadFile(raw JSON) error = %v", err)
	}
	if string(data) != string(runner.stdout) {
		t.Fatalf("raw JSON = %q, want %q", data, runner.stdout)
	}
}

func TestCollectorPreservesOutputAndReturnsErrorForInvalidJSON(t *testing.T) {
	store, err := evidence.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("evidence.NewStore() error = %v", err)
	}
	runner := &fakeRunner{
		stdout:   []byte("not-json"),
		stderr:   []byte("query failed"),
		exitCode: 1,
		err:      errors.New("exit status 1"),
	}
	collector := NewCollector(
		"osqueryi.exe",
		Query{ID: "broken", Category: "system", SQL: "SELECT * FROM missing;", Timeout: time.Second},
		runner,
	)

	artifacts, err := collector.Collect(context.Background(), store)
	if err == nil {
		t.Fatal("Collect() error = nil, want osquery failure")
	}
	if !strings.Contains(err.Error(), "exit code 1") {
		t.Fatalf("Collect() error = %q, want exit code", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("artifacts = %+v, want stdout and stderr", artifacts)
	}
	for _, relativePath := range []string{
		"evidence/osquery/broken.json",
		"logs/osquery/broken.stderr.txt",
	} {
		if _, statErr := os.Stat(filepath.Join(store.Root(), filepath.FromSlash(relativePath))); statErr != nil {
			t.Fatalf("expected preserved %s: %v", relativePath, statErr)
		}
	}
}

func TestDefaultQueriesAreUniquePassiveSelectStatements(t *testing.T) {
	queries := DefaultQueries()
	if len(queries) < 10 {
		t.Fatalf("DefaultQueries() count = %d, want at least 10", len(queries))
	}
	seen := make(map[string]struct{}, len(queries))
	for _, query := range queries {
		if _, exists := seen[query.ID]; exists {
			t.Fatalf("duplicate query ID %q", query.ID)
		}
		seen[query.ID] = struct{}{}
		normalized := strings.ToUpper(strings.TrimSpace(query.SQL))
		if !strings.HasPrefix(normalized, "SELECT ") {
			t.Fatalf("query %q is not SELECT-only: %s", query.ID, query.SQL)
		}
		if query.Mode != "passive" {
			t.Fatalf("query %q mode = %q, want passive", query.ID, query.Mode)
		}
		if query.Timeout <= 0 {
			t.Fatalf("query %q timeout = %s, want positive", query.ID, query.Timeout)
		}
	}
}

func TestDefaultDNSQueryUsesWindowsInterfaceDetailsSchema(t *testing.T) {
	foundWindowsDNSFields := false
	for _, query := range DefaultQueries() {
		if query.Category != "dns" {
			continue
		}
		if strings.Contains(strings.ToLower(query.SQL), "dns_resolvers") {
			t.Fatalf("DNS query %q references dns_resolvers, which is absent from the Windows osquery build", query.ID)
		}
		if strings.Contains(strings.ToLower(query.SQL), "dns_server_search_order") &&
			strings.Contains(strings.ToLower(query.SQL), "interface_details") {
			foundWindowsDNSFields = true
		}
	}
	if !foundWindowsDNSFields {
		t.Fatal("default catalog has no DNS query backed by Windows interface_details fields")
	}
}
