package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/dwg7318/codex-network-doctor/internal/app"
	"github.com/dwg7318/codex-network-doctor/internal/casebook"
	"github.com/dwg7318/codex-network-doctor/internal/engine"
	"github.com/dwg7318/codex-network-doctor/internal/evidence"
	"github.com/dwg7318/codex-network-doctor/internal/providers"
)

func Run(ctx context.Context, args []string, stdout, stderr io.Writer, version string) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "version", "--version", "-version":
		_, _ = fmt.Fprintf(stdout, "cnd %s\n", version)
		return 0
	case "catalog":
		return runCatalog(args[1:], stdout, stderr)
	case "cases", "case":
		return casebook.Run(args, stdout, stderr)
	case "collect":
		return runCollect(ctx, args[1:], stdout, stderr, version)
	case "sample":
		return runSample(ctx, args[1:], stdout, stderr, version)
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "help", "--help", "-h":
		printUsage(stdout)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runCatalog(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("catalog", flag.ContinueOnError)
	flags.SetOutput(stderr)
	provider := flags.String("provider", "auto", "provider: auto, osquery, or powershell")
	osqueryPath := flags.String("osquery", "", "path to osqueryi.exe")
	powerShellPath := flags.String("powershell", "", "path to powershell.exe or pwsh.exe")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		_, _ = fmt.Fprintln(stderr, "catalog does not accept positional arguments")
		return 2
	}

	selection, err := providers.Prepare(providers.Options{
		Provider:       *provider,
		OsqueryPath:    *osqueryPath,
		PowerShellPath: *powerShellPath,
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	document := struct {
		SchemaVersion string `json:"schema_version"`
		Provider      string `json:"provider"`
		BinaryPath    string `json:"binary_path"`
		Collectors    any    `json:"collectors"`
	}{
		SchemaVersion: "1.0.0",
		Provider:      selection.Provider,
		BinaryPath:    selection.BinaryPath,
		Collectors:    selection.Catalog,
	}
	if err := printJSON(stdout, document); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runCollect(ctx context.Context, args []string, stdout, stderr io.Writer, version string) int {
	flags := flag.NewFlagSet("collect", flag.ContinueOnError)
	flags.SetOutput(stderr)
	outputPath := flags.String("output", "", "destination Evidence ZIP path")
	provider := flags.String("provider", "auto", "provider: auto, osquery, or powershell")
	osqueryPath := flags.String("osquery", "", "path to osqueryi.exe")
	powerShellPath := flags.String("powershell", "", "path to powershell.exe or pwsh.exe")
	force := flags.Bool("force", false, "replace an existing output ZIP")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		_, _ = fmt.Fprintln(stderr, "collect does not accept positional arguments")
		return 2
	}
	if *outputPath == "" {
		_, _ = fmt.Fprintln(stderr, "collect requires --output")
		return 2
	}

	selection, err := providers.Prepare(providers.Options{
		Provider:       *provider,
		OsqueryPath:    *osqueryPath,
		PowerShellPath: *powerShellPath,
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	summary, err := app.Package(ctx, app.PackageOptions{
		OutputPath: *outputPath,
		Force:      *force,
		Tool: evidence.ToolInfo{
			Name:    "Codex Network Doctor",
			Version: version,
		},
		Provider:   selection.Provider,
		Collectors: selection.Collectors,
		Catalog:    selection.Catalog,
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if err := printJSON(stdout, summary); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runValidate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "validate requires exactly one Evidence ZIP path")
		return 2
	}
	report, err := evidence.ValidateZip(flags.Arg(0))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if err := printJSON(stdout, report); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if !report.Valid {
		return 2
	}
	return 0
}

func runSample(ctx context.Context, args []string, stdout, stderr io.Writer, version string) int {
	flags := flag.NewFlagSet("sample", flag.ContinueOnError)
	flags.SetOutput(stderr)
	outputPath := flags.String("output", "", "destination synthetic Evidence ZIP path")
	force := flags.Bool("force", false, "replace an existing output ZIP")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *outputPath == "" {
		_, _ = fmt.Fprintln(stderr, "sample requires --output and no positional arguments")
		return 2
	}

	type sampleCatalogEntry struct {
		ID          string `json:"id"`
		Category    string `json:"category"`
		Description string `json:"description"`
		TimeoutMS   int64  `json:"timeout_ms"`
		Privilege   string `json:"privilege"`
		Mode        string `json:"mode"`
	}
	definitions := []struct {
		catalog sampleCatalogEntry
		path    string
		payload string
	}{
		{
			catalog: sampleCatalogEntry{ID: "synthetic_system", Category: "system", Description: "Synthetic host metadata.", TimeoutMS: 1000, Privilege: "none", Mode: "passive"},
			path:    "evidence/synthetic/system.json",
			payload: "{\"hostname\":\"synthetic-host\",\"os\":\"Windows\",\"build\":\"sample\"}\n",
		},
		{
			catalog: sampleCatalogEntry{ID: "synthetic_adapter", Category: "adapter", Description: "Synthetic network adapter.", TimeoutMS: 1000, Privilege: "none", Mode: "passive"},
			path:    "evidence/synthetic/adapter.json",
			payload: "[{\"name\":\"Ethernet Sample\",\"address\":\"192.0.2.10\",\"prefix_length\":24,\"mac\":\"00:00:5E:00:53:01\"}]\n",
		},
		{
			catalog: sampleCatalogEntry{ID: "synthetic_route", Category: "route", Description: "Synthetic documentation-only route.", TimeoutMS: 1000, Privilege: "none", Mode: "passive"},
			path:    "evidence/synthetic/route.json",
			payload: "[{\"destination\":\"0.0.0.0/0\",\"gateway\":\"192.0.2.1\",\"interface\":\"Ethernet Sample\"}]\n",
		},
	}
	collectors := make([]engine.Collector, 0, len(definitions))
	catalog := make([]sampleCatalogEntry, 0, len(definitions))
	for _, definition := range definitions {
		current := definition
		catalog = append(catalog, current.catalog)
		collectors = append(collectors, engine.CollectorFunc{
			Spec: engine.Definition{
				ID:          current.catalog.ID,
				Category:    current.catalog.Category,
				Description: current.catalog.Description,
				Provider:    "synthetic",
				Timeout:     time.Second,
				Privilege:   current.catalog.Privilege,
				Mode:        current.catalog.Mode,
			},
			Run: func(_ context.Context, store *evidence.Store) ([]evidence.Artifact, error) {
				artifact, err := store.Write(current.path, []byte(current.payload), "application/json")
				return []evidence.Artifact{artifact}, err
			},
		})
	}

	summary, err := app.Package(ctx, app.PackageOptions{
		OutputPath: *outputPath,
		Force:      *force,
		Tool:       evidence.ToolInfo{Name: "Codex Network Doctor", Version: version},
		Provider:   "synthetic",
		Collectors: collectors,
		Catalog:    catalog,
		Hostname:   "synthetic-host",
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if err := printJSON(stdout, summary); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func printJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func printUsage(writer io.Writer) {
	_, _ = fmt.Fprintln(writer, "Codex Network Doctor (cnd)")
	_, _ = fmt.Fprintln(writer, "usage:")
	_, _ = fmt.Fprintln(writer, "  cnd collect --output evidence.zip [--provider auto|osquery|powershell]")
	_, _ = fmt.Fprintln(writer, "  cnd validate evidence.zip")
	_, _ = fmt.Fprintln(writer, "  cnd catalog [--provider auto|osquery|powershell]")
	_, _ = fmt.Fprintln(writer, "  cnd sample --output sample.zip")
	_, _ = fmt.Fprintln(writer, "  cnd cases")
	_, _ = fmt.Fprintln(writer, "  cnd case <case-id>")
	_, _ = fmt.Fprintln(writer, "  cnd version")
}
