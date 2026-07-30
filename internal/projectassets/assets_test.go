package projectassets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRequiredEngineeringAssetsExist(t *testing.T) {
	root := repositoryRoot(t)
	required := []string{
		"README.md",
		"VERSION",
		"CHANGELOG.md",
		"LICENSE",
		"CONTRIBUTING.md",
		"SECURITY.md",
		"docs/MASTER_SPEC.md",
		"docs/SDD.md",
		"docs/THREAT_MODEL.md",
		"docs/OPEN_SOURCE_EVALUATION.md",
		"docs/COLLECTOR_CATALOG.md",
		"docs/TEST_HANDBOOK.md",
		"docs/VERSIONING.md",
		"internal/casebook/cases/CND-CASE-2026-001.md",
		"docs/RFC/0001-evidence-package.md",
		"docs/ADR/0001-use-osquery-as-optional-core-provider.md",
		"schemas/manifest.schema.json",
		"schemas/timeline.schema.json",
		"schemas/collector-catalog.schema.json",
		"scripts/bootstrap-osquery.ps1",
		"scripts/build.ps1",
		".github/CODEOWNERS",
	}
	for _, relativePath := range required {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relativePath))); err != nil {
			t.Errorf("required asset %s: %v", relativePath, err)
		}
	}
}

func TestJSONSchemasAreValidJSON(t *testing.T) {
	root := repositoryRoot(t)
	for _, relativePath := range []string{
		"schemas/manifest.schema.json",
		"schemas/timeline.schema.json",
		"schemas/collector-catalog.schema.json",
	} {
		t.Run(relativePath, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relativePath)))
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			var document map[string]any
			if err := json.Unmarshal(data, &document); err != nil {
				t.Fatalf("schema JSON error = %v", err)
			}
			if document["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
				t.Fatalf("$schema = %v", document["$schema"])
			}
		})
	}
}

func TestBootstrapPinsVerifiedOsqueryWithoutInstallingService(t *testing.T) {
	scriptPath := filepath.Join(repositoryRoot(t), "scripts", "bootstrap-osquery.ps1")
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	script := string(data)
	for _, required := range []string{
		"5.23.1",
		"7bd411050ef6b5aae1b23956aec0dc5ce6e800c5656f0cd463ac70a6e1bdf30b",
		"a5be97a7cf3fcf58276f619f4519c2d3fbdf5ade58c1cb51e37cf4215c5beee5",
		"$expandedRoot",
		"Get-FileHash",
		"Expand-Archive",
	} {
		if !strings.Contains(script, required) {
			t.Errorf("bootstrap script missing %q", required)
		}
	}
	for _, forbidden := range []string{"New-Service", "Start-Service", "msiexec", "--install"} {
		if strings.Contains(script, forbidden) {
			t.Errorf("bootstrap script contains service installation primitive %q", forbidden)
		}
	}
}

func TestBuildScriptConfiguresWindowsRaceToolchain(t *testing.T) {
	scriptPath := filepath.Join(repositoryRoot(t), "scripts", "build.ps1")
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	script := string(data)
	for _, required := range []string{"CGO_ENABLED", "mingw64", "CC", "test -race"} {
		if !strings.Contains(script, required) {
			t.Errorf("build script missing race toolchain configuration %q", required)
		}
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}
