package evidence

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dwg7318/codex-network-doctor/internal/model"
)

func TestMarshalManifestSortsCollectorsAndFilesDeterministically(t *testing.T) {
	capturedAt := time.Date(2026, 7, 30, 1, 2, 3, 0, time.UTC)
	manifest := BuildManifest(
		ToolInfo{Name: "cnd", Version: "test"},
		CaptureInfo{ID: "capture-1", Hostname: "host", StartedAt: capturedAt, FinishedAt: capturedAt, Status: "complete"},
		[]model.CollectorResult{
			{ID: "z-last", Artifacts: []model.Artifact{}},
			{ID: "a-first", Artifacts: []model.Artifact{}},
		},
		[]model.Artifact{
			{Path: "evidence/z.json", SHA256: "z"},
			{Path: "evidence/a.json", SHA256: "a"},
		},
	)

	first, err := MarshalManifest(manifest)
	if err != nil {
		t.Fatalf("MarshalManifest() error = %v", err)
	}
	second, err := MarshalManifest(manifest)
	if err != nil {
		t.Fatalf("MarshalManifest() second error = %v", err)
	}

	if !bytes.Equal(first, second) {
		t.Fatal("MarshalManifest() returned different bytes for the same value")
	}
	if manifest.Collectors[0].ID != "a-first" || manifest.Files[0].Path != "evidence/a.json" {
		t.Fatalf("manifest is not sorted: collectors=%v files=%v", manifest.Collectors, manifest.Files)
	}
	if len(first) == 0 || first[len(first)-1] != '\n' {
		t.Fatal("manifest JSON must end with a newline")
	}
}

func TestBuildZipIsDeterministicAcrossFileCreationOrderAndTimestamps(t *testing.T) {
	firstRoot := t.TempDir()
	secondRoot := t.TempDir()
	writePackageFile(t, firstRoot, "evidence/z.txt", "z")
	writePackageFile(t, firstRoot, "collectors/a.json", "{}")
	writePackageFile(t, secondRoot, "collectors/a.json", "{}")
	writePackageFile(t, secondRoot, "evidence/z.txt", "z")

	old := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2025, 5, 5, 5, 5, 5, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(firstRoot, "evidence", "z.txt"), old, old); err != nil {
		t.Fatalf("Chtimes(first) error = %v", err)
	}
	if err := os.Chtimes(filepath.Join(secondRoot, "evidence", "z.txt"), newer, newer); err != nil {
		t.Fatalf("Chtimes(second) error = %v", err)
	}

	firstZip := filepath.Join(t.TempDir(), "first.zip")
	secondZip := filepath.Join(t.TempDir(), "second.zip")
	if _, err := BuildZip(firstRoot, firstZip); err != nil {
		t.Fatalf("BuildZip(first) error = %v", err)
	}
	if _, err := BuildZip(secondRoot, secondZip); err != nil {
		t.Fatalf("BuildZip(second) error = %v", err)
	}

	firstBytes, err := os.ReadFile(firstZip)
	if err != nil {
		t.Fatalf("ReadFile(first ZIP) error = %v", err)
	}
	secondBytes, err := os.ReadFile(secondZip)
	if err != nil {
		t.Fatalf("ReadFile(second ZIP) error = %v", err)
	}
	if !bytes.Equal(firstBytes, secondBytes) {
		t.Fatal("ZIP bytes differ for equivalent staging trees")
	}
}

func TestBuildZipRejectsDestinationInsideStagingRoot(t *testing.T) {
	root := t.TempDir()
	writePackageFile(t, root, "evidence/item.txt", "evidence")

	if _, err := BuildZip(root, filepath.Join(root, "nested", "evidence.zip")); err == nil {
		t.Fatal("BuildZip() error = nil, want destination-inside-staging rejection")
	}
}

func TestValidateZipDetectsEvidenceHashMismatch(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	artifact, err := store.Write("evidence/system.json", []byte("{}\n"), "application/json")
	if err != nil {
		t.Fatalf("Write(evidence) error = %v", err)
	}
	required := []struct {
		path string
		data string
	}{
		{"timeline.json", "{\"schema_version\":\"1.0.0\",\"events\":[]}\n"},
		{"collectors/catalog.json", "{\"collectors\":[]}\n"},
		{"validation/preflight.json", "{\"valid\":true}\n"},
	}
	files := []model.Artifact{artifact}
	for _, item := range required {
		written, writeErr := store.Write(item.path, []byte(item.data), "application/json")
		if writeErr != nil {
			t.Fatalf("Write(%s) error = %v", item.path, writeErr)
		}
		files = append(files, written)
	}

	now := time.Date(2026, 7, 30, 1, 2, 3, 0, time.UTC)
	manifest := BuildManifest(
		ToolInfo{Name: "cnd", Version: "test"},
		CaptureInfo{ID: "capture-1", Hostname: "host", StartedAt: now, FinishedAt: now, Status: "complete"},
		nil,
		files,
	)
	manifestBytes, err := MarshalManifest(manifest)
	if err != nil {
		t.Fatalf("MarshalManifest() error = %v", err)
	}
	if _, err := store.Write("manifest.json", manifestBytes, "application/json"); err != nil {
		t.Fatalf("Write(manifest) error = %v", err)
	}

	validZip := filepath.Join(t.TempDir(), "valid.zip")
	if _, err := BuildZip(root, validZip); err != nil {
		t.Fatalf("BuildZip(valid) error = %v", err)
	}
	validReport, err := ValidateZip(validZip)
	if err != nil {
		t.Fatalf("ValidateZip(valid) error = %v", err)
	}
	if !validReport.Valid {
		t.Fatalf("valid ZIP report = %+v", validReport)
	}

	if err := os.WriteFile(filepath.Join(root, "evidence", "system.json"), []byte("{\"tampered\":true}\n"), 0o600); err != nil {
		t.Fatalf("tamper evidence error = %v", err)
	}
	tamperedZip := filepath.Join(t.TempDir(), "tampered.zip")
	if _, err := BuildZip(root, tamperedZip); err != nil {
		t.Fatalf("BuildZip(tampered) error = %v", err)
	}
	tamperedReport, err := ValidateZip(tamperedZip)
	if err != nil {
		t.Fatalf("ValidateZip(tampered) error = %v", err)
	}
	if tamperedReport.Valid {
		t.Fatalf("tampered ZIP reported valid: %+v", tamperedReport)
	}
	if !hasIssueCode(tamperedReport.Issues, "HASH_MISMATCH") {
		t.Fatalf("tampered issues = %+v, want HASH_MISMATCH", tamperedReport.Issues)
	}
}

func TestValidateZipRejectsWindowsAlternateDataStreamPath(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "ads.zip")
	output, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("os.Create() error = %v", err)
	}
	writer := zip.NewWriter(output)
	entry, err := writer.Create("evidence/file.txt:stream")
	if err != nil {
		t.Fatalf("zip Create() error = %v", err)
	}
	if _, err := entry.Write([]byte("hidden")); err != nil {
		t.Fatalf("zip entry Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("zip writer Close() error = %v", err)
	}
	if err := output.Close(); err != nil {
		t.Fatalf("ZIP Close() error = %v", err)
	}

	report, err := ValidateZip(zipPath)
	if err != nil {
		t.Fatalf("ValidateZip() error = %v", err)
	}
	if report.Valid || !hasIssueCode(report.Issues, "UNSAFE_PATH") {
		t.Fatalf("report = %+v, want UNSAFE_PATH", report)
	}
}

func writePackageFile(t *testing.T, root, relativePath, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", relativePath, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", relativePath, err)
	}
}

func hasIssueCode(issues []ValidationIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
