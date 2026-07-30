package app

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dwg7318/codex-network-doctor/internal/engine"
	"github.com/dwg7318/codex-network-doctor/internal/evidence"
)

func TestPackageCreatesValidPartialEvidenceZip(t *testing.T) {
	success := engine.CollectorFunc{
		Spec: engine.Definition{ID: "system", Category: "system", Provider: "test", Timeout: time.Second},
		Run: func(_ context.Context, store *evidence.Store) ([]evidence.Artifact, error) {
			artifact, err := store.Write("evidence/test/system.json", []byte("{}\n"), "application/json")
			return []evidence.Artifact{artifact}, err
		},
	}
	failure := engine.CollectorFunc{
		Spec: engine.Definition{ID: "routes", Category: "route", Provider: "test", Timeout: time.Second},
		Run: func(context.Context, *evidence.Store) ([]evidence.Artifact, error) {
			return nil, errors.New("access denied")
		},
	}
	output := filepath.Join(t.TempDir(), "evidence.zip")

	summary, err := Package(context.Background(), PackageOptions{
		OutputPath: output,
		Tool:       evidence.ToolInfo{Name: "cnd", Version: "test"},
		Provider:   "test",
		Collectors: []engine.Collector{success, failure},
		Catalog:    map[string]any{"provider": "test"},
	})
	if err != nil {
		t.Fatalf("Package() error = %v", err)
	}

	if summary.Status != "partial" {
		t.Fatalf("summary status = %q, want partial", summary.Status)
	}
	if summary.SuccessfulCollectors != 1 || summary.FailedCollectors != 1 {
		t.Fatalf("summary collector counts = %+v", summary)
	}
	report, err := evidence.ValidateZip(output)
	if err != nil {
		t.Fatalf("ValidateZip() error = %v", err)
	}
	if !report.Valid {
		t.Fatalf("ValidateZip() report = %+v", report)
	}

	manifest := readManifestFromZip(t, output)
	if manifest.Capture.Status != "partial" {
		t.Fatalf("manifest capture status = %q, want partial", manifest.Capture.Status)
	}
	if len(manifest.Collectors) != 2 {
		t.Fatalf("manifest collector count = %d, want 2", len(manifest.Collectors))
	}
}

func TestPackageRefusesToOverwriteOutputWithoutForce(t *testing.T) {
	output := filepath.Join(t.TempDir(), "existing.zip")
	if err := os.WriteFile(output, []byte("keep me"), 0o600); err != nil {
		t.Fatalf("WriteFile(existing) error = %v", err)
	}

	_, err := Package(context.Background(), PackageOptions{
		OutputPath: output,
		Tool:       evidence.ToolInfo{Name: "cnd", Version: "test"},
	})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("Package() error = %v, want already exists", err)
	}
	data, readErr := os.ReadFile(output)
	if readErr != nil {
		t.Fatalf("ReadFile(existing) error = %v", readErr)
	}
	if string(data) != "keep me" {
		t.Fatalf("existing output was modified: %q", data)
	}
}

func readManifestFromZip(t *testing.T, zipPath string) evidence.Manifest {
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
