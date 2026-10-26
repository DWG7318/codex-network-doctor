package networkcli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/dwg7318/codex-network-doctor/internal/networkhealth"
)

// runDiagnose is the only live entry point. Existing synthetic commands are unchanged.
func runDiagnose(ctx context.Context, args []string, stdout, stderr io.Writer, version string) int {
	flags := newFlagSet("diagnose")
	input := flags.String("input", "", "explicit targets and paths")
	output := flags.String("output", "", "private JSON report")
	textOutput := flags.String("text-output", "", "Chinese text report")
	if !parseFlags(flags, args, stderr) || !validInputPath(*input) || !validOutputPath(*output) || !validOutputPath(*textOutput) {
		printError(stderr, "INVALID_ARGUMENTS")
		return 2
	}
	f, err := os.Open(*input)
	if err != nil {
		printError(stderr, "HEALTH_INPUT_READ_FAILED")
		return 1
	}
	config, err := networkhealth.Decode(f)
	_ = f.Close()
	if err != nil {
		printError(stderr, "HEALTH_INPUT_INVALID")
		return 2
	}
	if runtime.GOOS != "linux" {
		printError(stderr, "LINUX_REQUIRED")
		return 2
	}
	// Reserve both destinations before any host or network observation.
	outputs, err := reserveHealthOutputs(*output, *textOutput)
	if err != nil {
		printError(stderr, "HEALTH_OUTPUT_CREATE_FAILED")
		return 1
	}
	written := false
	defer func() {
		for _, f := range outputs {
			_ = f.Close()
			if !written {
				_ = os.Remove(f.Name())
			}
		}
	}()
	started := time.Now()
	host := networkhealth.Snapshot()
	// A total wall-clock budget bounds all serial attempts, in addition to per-attempt deadlines.
	probeCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	samples := (&networkhealth.Prober{}).Run(probeCtx, config)
	report := networkhealth.NewReport(version, started, host, config, samples)
	enc := json.NewEncoder(outputs[0])
	enc.SetIndent("", "  ")
	if enc.Encode(report) != nil || networkhealth.WriteText(outputs[1], report) != nil {
		printError(stderr, "HEALTH_OUTPUT_WRITE_FAILED")
		return 1
	}
	for _, f := range outputs {
		if f.Close() != nil {
			printError(stderr, "HEALTH_OUTPUT_CLOSE_FAILED")
			return 1
		}
	}
	written = true
	_, _ = fmt.Fprintf(stdout, "diagnose %s samples=%d/%d host=%s\n", report.Assessment.Status, report.Assessment.CompletedSamples, report.Assessment.ExpectedSamples, host.Status)
	if !host.Supported || host.Status != "complete" || report.Assessment.Status != "healthy" {
		return 3
	}
	return 0
}

func reserveHealthOutputs(paths ...string) ([]*os.File, error) {
	seen := map[string]bool{}
	out := []*os.File{}
	fail := func() ([]*os.File, error) {
		for _, f := range out {
			_ = f.Close()
			_ = os.Remove(f.Name())
		}
		return nil, errors.New("HEALTH_OUTPUT_CREATE_FAILED")
	}
	for _, p := range paths {
		absolute, err := filepath.Abs(p)
		if err != nil || seen[absolute] {
			return fail()
		}
		seen[absolute] = true
		f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return fail()
		}
		out = append(out, f)
		if runtime.GOOS == "linux" {
			info, err := f.Stat()
			if err != nil || !privateHealthMode(info.Mode()) {
				return fail()
			}
		}
	}
	return out, nil
}

func privateHealthMode(mode os.FileMode) bool {
	return mode.IsRegular() && mode.Perm() == 0600
}
