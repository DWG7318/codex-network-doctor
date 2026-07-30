package cli

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dwg7318/codex-network-doctor/internal/evidence"
)

func TestRunVersionPrintsVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run(context.Background(), []string{"version"}, &stdout, &stderr, "1.2.3")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", exitCode, stderr.String())
	}
	if stdout.String() != "cnd 1.2.3\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunCaseCommandsExposeEmbeddedRepairCase(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run(
		context.Background(),
		[]string{"case", "CND-CASE-2026-001"},
		&stdout,
		&stderr,
		"test",
	)

	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", exitCode, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("excluded port range")) {
		t.Fatalf("stdout does not contain case root cause: %q", stdout.String())
	}
}

func TestRunCatalogPrintsMachineReadableProviderCatalog(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run(
		context.Background(),
		[]string{"catalog", "--provider", "powershell", "--powershell", os.Args[0]},
		&stdout,
		&stderr,
		"test",
	)

	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", exitCode, stderr.String())
	}
	var document struct {
		Provider   string            `json:"provider"`
		Collectors []json.RawMessage `json:"collectors"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &document); err != nil {
		t.Fatalf("catalog JSON error = %v; output = %q", err, stdout.String())
	}
	if document.Provider != "windows-powershell" {
		t.Fatalf("provider = %q", document.Provider)
	}
	if len(document.Collectors) < 10 {
		t.Fatalf("collector count = %d, want at least 10", len(document.Collectors))
	}
}

func TestRunValidateReturnsFailureForMissingZip(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run(context.Background(), []string{"validate", "missing.zip"}, &stdout, &stderr, "test")

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if stderr.Len() == 0 {
		t.Fatal("stderr is empty")
	}
}

func TestRunSampleCreatesSyntheticValidEvidenceZip(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	output := filepath.Join(t.TempDir(), "sample.zip")

	exitCode := Run(
		context.Background(),
		[]string{"sample", "--output", output},
		&stdout,
		&stderr,
		"test",
	)

	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", exitCode, stderr.String())
	}
	report, err := evidence.ValidateZip(output)
	if err != nil {
		t.Fatalf("ValidateZip() error = %v", err)
	}
	if !report.Valid {
		t.Fatalf("sample validation report = %+v", report)
	}
	manifest := readManifestForCLITest(t, output)
	if manifest.Capture.Hostname != "synthetic-host" {
		t.Fatalf("sample hostname = %q, want synthetic-host", manifest.Capture.Hostname)
	}
	if manifest.Capture.Status != "complete" {
		t.Fatalf("sample status = %q, want complete", manifest.Capture.Status)
	}
}

func readManifestForCLITest(t *testing.T, zipPath string) evidence.Manifest {
	t.Helper()
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("zip.OpenReader() error = %v", err)
	}
	defer func() {
		_ = reader.Close()
	}()
	for _, file := range reader.File {
		if file.Name != "manifest.json" {
			continue
		}
		stream, err := file.Open()
		if err != nil {
			t.Fatalf("manifest Open() error = %v", err)
		}
		defer func() {
			_ = stream.Close()
		}()
		var manifest evidence.Manifest
		if err := json.NewDecoder(stream).Decode(&manifest); err != nil {
			t.Fatalf("manifest Decode() error = %v", err)
		}
		return manifest
	}
	t.Fatal("manifest.json not found")
	return evidence.Manifest{}
}
