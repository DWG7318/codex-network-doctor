package providers

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dwg7318/codex-network-doctor/internal/engine"
	osqueryprovider "github.com/dwg7318/codex-network-doctor/internal/providers/osquery"
	powershellprovider "github.com/dwg7318/codex-network-doctor/internal/providers/powershell"
	"github.com/dwg7318/codex-network-doctor/internal/runner"
)

type Options struct {
	Provider       string
	OsqueryPath    string
	PowerShellPath string
}

type Selection struct {
	Provider   string
	BinaryPath string
	Collectors []engine.Collector
	Catalog    any
}

func Prepare(options Options) (Selection, error) {
	provider := strings.ToLower(strings.TrimSpace(options.Provider))
	if provider == "" {
		provider = "auto"
	}

	switch provider {
	case "auto":
		if binary, err := resolveExecutable(options.OsqueryPath, "osqueryi.exe"); err == nil {
			return osquerySelection(binary), nil
		}
		if binary, err := resolveExecutable(options.PowerShellPath, "powershell.exe", "pwsh.exe"); err == nil {
			return powershellSelection(binary), nil
		}
		return Selection{}, errors.New("no supported provider found; install osquery or provide a PowerShell executable")
	case "osquery":
		binary, err := resolveExecutable(options.OsqueryPath, "osqueryi.exe")
		if err != nil {
			return Selection{}, fmt.Errorf("osquery provider unavailable: %w", err)
		}
		return osquerySelection(binary), nil
	case "powershell", "windows-powershell":
		binary, err := resolveExecutable(options.PowerShellPath, "powershell.exe", "pwsh.exe")
		if err != nil {
			return Selection{}, fmt.Errorf("PowerShell provider unavailable: %w", err)
		}
		return powershellSelection(binary), nil
	default:
		return Selection{}, fmt.Errorf("unsupported provider %q; expected auto, osquery, or powershell", options.Provider)
	}
}

func resolveExecutable(explicit string, candidates ...string) (string, error) {
	if explicit != "" {
		absolute, err := filepath.Abs(explicit)
		if err != nil {
			return "", err
		}
		info, err := os.Stat(absolute)
		if err != nil {
			return "", err
		}
		if info.IsDir() {
			return "", fmt.Errorf("%q is a directory", absolute)
		}
		return absolute, nil
	}
	for _, candidate := range candidates {
		if found, err := exec.LookPath(candidate); err == nil {
			return found, nil
		}
	}
	return "", fmt.Errorf("none of %v were found on PATH", candidates)
}

func osquerySelection(binary string) Selection {
	queries := osqueryprovider.DefaultQueries()
	collectors := make([]engine.Collector, 0, len(queries))
	for _, query := range queries {
		collectors = append(collectors, osqueryprovider.NewCollector(binary, query, runner.OSRunner{}))
	}
	return Selection{
		Provider:   "osquery",
		BinaryPath: binary,
		Collectors: collectors,
		Catalog:    queries,
	}
}

func powershellSelection(binary string) Selection {
	scripts := powershellprovider.DefaultScripts()
	collectors := make([]engine.Collector, 0, len(scripts))
	for _, script := range scripts {
		collectors = append(collectors, powershellprovider.NewCollector(binary, script, runner.OSRunner{}))
	}
	return Selection{
		Provider:   "windows-powershell",
		BinaryPath: binary,
		Collectors: collectors,
		Catalog:    scripts,
	}
}
