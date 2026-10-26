//go:build linux

package networkcli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/dwg7318/codex-network-doctor/internal/networkhealth"
)

func TestDiagnoseLinuxWritesRealFailureReportWithPrivateMode(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("untrusted TLS must not reach HTTP") }))
	defer srv.Close()
	dir := t.TempDir()
	input := filepath.Join(dir, "targets.json")
	output := filepath.Join(dir, "health.json")
	txt := filepath.Join(dir, "health.txt")
	config := networkhealth.Config{SchemaVersion: "1.0.0", Samples: 3, TimeoutMS: 1000, Targets: []networkhealth.Target{{ID: "local-only", URL: srv.URL, ExpectedStatus: 204}}, Routes: []networkhealth.Route{{ID: "direct"}}}
	b, _ := json.Marshal(config)
	if err := os.WriteFile(input, b, 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	args := []string{"diagnose", "--input", input, "--output", output, "--text-output", txt}
	if code := Run(context.Background(), args, &stdout, &stderr, "test"); code != 3 {
		t.Fatalf("code=%d stderr=%s", code, &stderr)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var report networkhealth.Report
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.EvidenceMode != "live" || len(report.Samples) != 3 || report.Assessment.Status == "healthy" {
		t.Fatal("invalid live report")
	}
	for _, p := range []string{output, txt} {
		st, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if st.Mode().Perm() != 0600 {
			t.Errorf("mode=%o", st.Mode().Perm())
		}
	}
	if bytes.Contains(data, []byte(srv.URL)) {
		t.Fatal("target URL exposed in report")
	}
	if code := Run(context.Background(), args, &stdout, &stderr, "test"); code != 1 {
		t.Fatalf("overwrite code=%d", code)
	}
	after, _ := os.ReadFile(output)
	if !bytes.Equal(data, after) {
		t.Fatal("overwrote evidence")
	}
}
