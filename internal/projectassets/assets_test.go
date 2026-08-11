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
		"docs/RFC/0003-ubuntu-network-core.md",
		"docs/ADR/0001-use-osquery-as-optional-core-provider.md",
		"schemas/manifest.schema.json",
		"schemas/timeline.schema.json",
		"schemas/collector-catalog.schema.json",
		"schemas/network-node.schema.json",
		"schemas/network-target-set.schema.json",
		"schemas/network-route-profile.schema.json",
		"schemas/network-change-plan.schema.json",
		"schemas/network-execution-receipt.schema.json",
		"schemas/network-health-report.schema.json",
		"samples/network/node-china-egress.synthetic.json",
		"samples/network/node-global-egress.synthetic.json",
		"samples/network/targets-domestic.synthetic.json",
		"samples/network/targets-global.synthetic.json",
		"samples/network/protected-management.synthetic.json",
		"samples/network/profile-overseas-host.synthetic.json",
		"samples/network/profile-domestic-host.synthetic.json",
		"scripts/bootstrap-osquery.ps1",
		"scripts/build.ps1",
		"scripts/build-linux.ps1",
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
		"schemas/network-node.schema.json",
		"schemas/network-target-set.schema.json",
		"schemas/network-route-profile.schema.json",
		"schemas/network-change-plan.schema.json",
		"schemas/network-execution-receipt.schema.json",
		"schemas/network-health-report.schema.json",
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

func TestNetworkJSONSchemasRejectAdditionalProperties(t *testing.T) {
	root := repositoryRoot(t)
	for _, relativePath := range []string{
		"schemas/network-node.schema.json",
		"schemas/network-target-set.schema.json",
		"schemas/network-route-profile.schema.json",
		"schemas/network-change-plan.schema.json",
		"schemas/network-execution-receipt.schema.json",
		"schemas/network-health-report.schema.json",
	} {
		t.Run(relativePath, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relativePath)))
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			var schema map[string]any
			if err := json.Unmarshal(data, &schema); err != nil {
				t.Fatalf("schema JSON error = %v", err)
			}
			if schema["additionalProperties"] != false {
				t.Fatalf("additionalProperties = %v", schema["additionalProperties"])
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
	script := readAsset(t, "scripts/build.ps1")
	for _, required := range []string{"CGO_ENABLED", "mingw64", "CC", "test -race"} {
		if !strings.Contains(script, required) {
			t.Errorf("build script missing race toolchain configuration %q", required)
		}
	}
	for _, required := range []string{"cmd\\cnd-network", "cnd-network.exe"} {
		if !strings.Contains(script, required) {
			t.Errorf("build script missing network CLI output %q", required)
		}
	}
}

func TestLinuxBuildScriptCrossCompilesOnlyNetworkCLI(t *testing.T) {
	script := readAsset(t, "scripts/build-linux.ps1")
	for _, required := range []string{"GOOS", "linux", "GOARCH", "amd64", "CGO_ENABLED", "cmd\\cnd-network"} {
		if !strings.Contains(script, required) {
			t.Errorf("linux build script missing %q", required)
		}
	}
	for _, forbidden := range []string{"docker", "systemctl", "sudo", "apt install", "sing-box install"} {
		if strings.Contains(strings.ToLower(script), forbidden) {
			t.Errorf("linux build script contains side-effect primitive %q", forbidden)
		}
	}
}

func readAsset(t *testing.T, relativePath string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repositoryRoot(t), filepath.FromSlash(relativePath)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}
