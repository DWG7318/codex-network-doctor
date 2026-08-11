package networkcli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dwg7318/codex-network-doctor/internal/networkcore"
	"github.com/dwg7318/codex-network-doctor/internal/networkengine/singbox"
)

const maxInputBytes = 1 << 20

var errInputTooLarge = errors.New("input too large")

type operationSummary struct {
	SchemaVersion string                        `json:"schema_version"`
	Command       string                        `json:"command"`
	State         string                        `json:"state"`
	Digest        string                        `json:"digest,omitempty"`
	PlanID        string                        `json:"plan_id,omitempty"`
	Scenario      networkcore.Scenario          `json:"scenario,omitempty"`
	EvidenceMode  networkcore.EvidenceMode      `json:"evidence_mode,omitempty"`
	HealthStatus  networkcore.HealthStatus      `json:"health_status,omitempty"`
	Render        *singbox.RenderSummary        `json:"render,omitempty"`
	Validation    *networkcore.ValidationReport `json:"validation,omitempty"`
}

func Run(ctx context.Context, args []string, stdout, stderr io.Writer, version string) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	if err := ctx.Err(); err != nil {
		printError(stderr, "OPERATION_CANCELLED")
		return 1
	}

	switch args[0] {
	case "version", "--version", "-version":
		_, _ = fmt.Fprintf(stdout, "cnd-network %s\n", version)
		return 0
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "plan":
		return runPlan(args[1:], stdout, stderr, version)
	case "render":
		return runRender(args[1:], stdout, stderr)
	case "simulate":
		return runSimulate(args[1:], stdout, stderr)
	case "report":
		return runReport(args[1:], stdout, stderr)
	case "help", "--help", "-h":
		printUsage(stdout)
		return 0
	default:
		_, _ = io.WriteString(stderr, "unknown command\n")
		printUsage(stderr)
		return 2
	}
}

func runValidate(args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("validate")
	kind := flags.String("kind", "", "contract kind")
	input := flags.String("input", "", "input file")
	format := flags.String("format", "text", "output format")
	if !parseFlags(flags, args, stderr) || !validInputPath(*input) || !validFormat(*format) {
		printError(stderr, "INVALID_ARGUMENTS")
		return 2
	}

	var validationErr error
	switch *kind {
	case "node":
		_, _, validationErr = decodeContractFile(*input, "NODE", networkcore.DecodeNode)
	case "target-set":
		_, _, validationErr = decodeContractFile(*input, "TARGET_SET", networkcore.DecodeTargetSet)
	case "route-profile":
		_, _, validationErr = decodeContractFile(*input, "ROUTE_PROFILE", networkcore.DecodeRouteProfile)
	default:
		printError(stderr, "KIND_UNSUPPORTED")
		return 2
	}

	if validationErr != nil {
		var contractErr networkcore.ValidationError
		if errors.As(validationErr, &contractErr) {
			report := networkcore.ValidationReport{SchemaVersion: networkcore.SchemaVersion, Kind: *kind, Valid: false, Issues: contractErr.Issues}
			writeValidation(stdout, *format, report)
			return 2
		}
		printError(stderr, safeCode(validationErr))
		if isReadError(validationErr) {
			return 1
		}
		return 2
	}
	report := networkcore.ValidationReport{SchemaVersion: networkcore.SchemaVersion, Kind: *kind, Valid: true, Issues: []networkcore.ValidationIssue{}}
	writeValidation(stdout, *format, report)
	return 0
}

func runPlan(args []string, stdout, stderr io.Writer, version string) int {
	flags := newFlagSet("plan")
	nodePath := flags.String("node", "", "node descriptor")
	targetsPath := flags.String("targets", "", "target set")
	protectedPath := flags.String("protected", "", "protected target set")
	profilePath := flags.String("profile", "", "route profile")
	outputPath := flags.String("output", "", "draft output")
	format := flags.String("format", "text", "output format")
	if !parseFlags(flags, args, stderr) || !allInputPaths(*nodePath, *targetsPath, *protectedPath, *profilePath) || !validOutputPath(*outputPath) || !validFormat(*format) {
		printError(stderr, "INVALID_ARGUMENTS")
		return 2
	}
	if !preflightAbsent(*outputPath) {
		printError(stderr, "DRAFT_OUTPUT_EXISTS")
		return 1
	}

	node, exit, err := decodeContractFile(*nodePath, "NODE", networkcore.DecodeNode)
	if err != nil {
		return reportCommandError(stderr, exit, err)
	}
	targets, exit, err := decodeContractFile(*targetsPath, "TARGET_SET", networkcore.DecodeTargetSet)
	if err != nil {
		return reportCommandError(stderr, exit, err)
	}
	protected, exit, err := decodeContractFile(*protectedPath, "PROTECTED_TARGET_SET", networkcore.DecodeTargetSet)
	if err != nil {
		return reportCommandError(stderr, exit, err)
	}
	profile, exit, err := decodeContractFile(*profilePath, "ROUTE_PROFILE", networkcore.DecodeRouteProfile)
	if err != nil {
		return reportCommandError(stderr, exit, err)
	}
	draft, err := networkcore.BuildDraft(node, targets, protected, profile, version)
	if err != nil {
		printError(stderr, safeCode(err))
		return 2
	}
	digest, err := writeCanonicalExclusive(*outputPath, draft, 0o644)
	if err != nil {
		printError(stderr, "DRAFT_WRITE_FAILED")
		return 1
	}
	return writeSummary(stdout, *format, operationSummary{SchemaVersion: networkcore.SchemaVersion, Command: "plan", State: string(draft.State), Digest: digest})
}

func runRender(args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("render")
	draftPath := flags.String("draft", "", "draft input")
	materialPath := flags.String("material", "", "protected material input")
	candidatePath := flags.String("candidate", "", "protected candidate output")
	planPath := flags.String("plan", "", "finalized plan output")
	format := flags.String("format", "text", "output format")
	if !parseFlags(flags, args, stderr) || !allInputPaths(*draftPath, *materialPath) || !validOutputPath(*candidatePath) || !validOutputPath(*planPath) || !validFormat(*format) {
		printError(stderr, "INVALID_ARGUMENTS")
		return 2
	}
	if !preflightAbsent(*candidatePath) || !preflightAbsent(*planPath) {
		printError(stderr, "RENDER_OUTPUT_EXISTS")
		return 1
	}

	draft, exit, err := decodeJSONFile[networkcore.PlanDraft](*draftPath, "DRAFT", validateDraft)
	if err != nil {
		return reportCommandError(stderr, exit, err)
	}
	material, exit, err := decodeContractFile(*materialPath, "MATERIAL", singbox.DecodeMaterial)
	if err != nil {
		return reportCommandError(stderr, exit, err)
	}
	plan, summary, err := singbox.RenderToFile(draft, material, *candidatePath)
	if err != nil {
		printError(stderr, safeCode(err))
		if strings.HasPrefix(err.Error(), "CANDIDATE_") {
			return 1
		}
		return 2
	}
	if _, err := writeCanonicalExclusive(*planPath, plan, 0o644); err != nil {
		_ = os.Remove(*candidatePath)
		printError(stderr, "PLAN_WRITE_FAILED")
		return 1
	}
	return writeSummary(stdout, *format, operationSummary{SchemaVersion: networkcore.SchemaVersion, Command: "render", State: string(plan.State), PlanID: plan.PlanID, Render: &summary})
}

func runSimulate(args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("simulate")
	planPath := flags.String("plan", "", "finalized plan input")
	scenarioValue := flags.String("scenario", "", "simulation scenario")
	outputPath := flags.String("output", "", "receipt output")
	format := flags.String("format", "text", "output format")
	if !parseFlags(flags, args, stderr) || !validInputPath(*planPath) || !validOutputPath(*outputPath) || !validFormat(*format) {
		printError(stderr, "INVALID_ARGUMENTS")
		return 2
	}
	if !preflightAbsent(*outputPath) {
		printError(stderr, "RECEIPT_OUTPUT_EXISTS")
		return 1
	}
	plan, exit, err := decodeJSONFile[networkcore.NetworkChangePlan](*planPath, "PLAN", validatePlan)
	if err != nil {
		return reportCommandError(stderr, exit, err)
	}
	scenario := networkcore.Scenario(*scenarioValue)
	receipt, err := networkcore.Simulate(plan, scenario)
	if err != nil {
		printError(stderr, safeCode(err))
		return 2
	}
	if _, err := writeCanonicalExclusive(*outputPath, receipt, 0o644); err != nil {
		printError(stderr, "RECEIPT_WRITE_FAILED")
		return 1
	}
	return writeSummary(stdout, *format, operationSummary{SchemaVersion: networkcore.SchemaVersion, Command: "simulate", State: string(receipt.State), PlanID: receipt.PlanID, Scenario: receipt.Scenario, EvidenceMode: networkcore.EvidenceSynthetic})
}

func runReport(args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("report")
	planPath := flags.String("plan", "", "finalized plan input")
	receiptPath := flags.String("receipt", "", "execution receipt input")
	outputPath := flags.String("output", "", "report output")
	format := flags.String("format", "text", "output format")
	if !parseFlags(flags, args, stderr) || !allInputPaths(*planPath, *receiptPath) || !validOutputPath(*outputPath) || !validFormat(*format) {
		printError(stderr, "INVALID_ARGUMENTS")
		return 2
	}
	if !preflightAbsent(*outputPath) {
		printError(stderr, "REPORT_OUTPUT_EXISTS")
		return 1
	}
	plan, exit, err := decodeJSONFile[networkcore.NetworkChangePlan](*planPath, "PLAN", validatePlan)
	if err != nil {
		return reportCommandError(stderr, exit, err)
	}
	receipt, exit, err := decodeJSONFile[networkcore.ExecutionReceipt](*receiptPath, "RECEIPT", validateReceipt)
	if err != nil {
		return reportCommandError(stderr, exit, err)
	}
	report, err := networkcore.BuildReport(plan, receipt)
	if err != nil {
		printError(stderr, safeCode(err))
		return 2
	}
	if _, err := writeCanonicalExclusive(*outputPath, report, 0o644); err != nil {
		printError(stderr, "REPORT_WRITE_FAILED")
		return 1
	}
	return writeSummary(stdout, *format, operationSummary{SchemaVersion: networkcore.SchemaVersion, Command: "report", State: string(report.Status), PlanID: report.PlanID, EvidenceMode: report.EvidenceMode, HealthStatus: report.Status})
}

func decodeContractFile[T any](path, role string, decode func(io.Reader) (T, error)) (T, int, error) {
	var zero T
	file, err := os.Open(path)
	if err != nil {
		return zero, 1, errors.New(role + "_READ_FAILED")
	}
	defer file.Close()
	data, err := readBounded(file)
	if errors.Is(err, errInputTooLarge) {
		return zero, 2, errors.New(role + "_INPUT_TOO_LARGE")
	}
	if err != nil {
		return zero, 1, errors.New(role + "_READ_FAILED")
	}
	value, err := decode(bytes.NewReader(data))
	if err != nil {
		return value, 2, err
	}
	return value, 0, nil
}

func decodeJSONFile[T any](path, role string, validate func(T) error) (T, int, error) {
	var value T
	file, err := os.Open(path)
	if err != nil {
		return value, 1, errors.New(role + "_READ_FAILED")
	}
	defer file.Close()
	data, err := readBounded(file)
	if errors.Is(err, errInputTooLarge) {
		return value, 2, errors.New(role + "_INPUT_TOO_LARGE")
	}
	if err != nil {
		return value, 1, errors.New(role + "_READ_FAILED")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, 2, errors.New(role + "_DECODE_FAILED")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return value, 2, errors.New(role + "_TRAILING_DATA")
	}
	if err := validate(value); err != nil {
		return value, 2, err
	}
	return value, 0, nil
}

func readBounded(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxInputBytes+1))
	if err != nil {
		return nil, errors.New("input read failed")
	}
	if len(data) > maxInputBytes {
		return nil, errInputTooLarge
	}
	return data, nil
}

func validateDraft(draft networkcore.PlanDraft) error {
	if draft.SchemaVersion != networkcore.SchemaVersion || draft.State != networkcore.PlanValidated || !draft.SimulationOnly || draft.Platform != "ubuntu-24.04" || draft.Architecture != "amd64" || draft.Candidate.Engine != "sing-box" || draft.Candidate.EngineVersion != networkcore.SingBoxVersion || draft.Candidate.SecretRef == "" {
		return errors.New("DRAFT_INVALID")
	}
	return nil
}

func validatePlan(plan networkcore.NetworkChangePlan) error {
	if plan.SchemaVersion != networkcore.SchemaVersion || plan.State != networkcore.PlanReady || !plan.SimulationOnly || len(plan.PlanID) != 64 || len(plan.CandidateSHA256) != 64 || plan.Platform != "ubuntu-24.04" || plan.Architecture != "amd64" {
		return errors.New("PLAN_INVALID")
	}
	return nil
}

func validateReceipt(receipt networkcore.ExecutionReceipt) error {
	if receipt.SchemaVersion != networkcore.SchemaVersion || receipt.EvidenceMode != string(networkcore.EvidenceSynthetic) || len(receipt.PlanID) != 64 || len(receipt.CandidateSHA256) != 64 || (receipt.State != networkcore.ExecutionVerified && receipt.State != networkcore.ExecutionRolledBack) {
		return errors.New("RECEIPT_INVALID")
	}
	return nil
}

func writeCanonicalExclusive(path string, value any, mode os.FileMode) (string, error) {
	data, digest, err := networkcore.CanonicalJSON(value)
	if err != nil {
		return "", err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return "", errors.New("OUTPUT_CREATE_FAILED")
	}
	written := false
	defer func() {
		if !written {
			_ = os.Remove(path)
		}
	}()
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return "", errors.New("OUTPUT_WRITE_FAILED")
	}
	if err := file.Close(); err != nil {
		return "", errors.New("OUTPUT_CLOSE_FAILED")
	}
	written = true
	return digest, nil
}

func preflightAbsent(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

func newFlagSet(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	return flags
}

func parseFlags(flags *flag.FlagSet, args []string, stderr io.Writer) bool {
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return false
	}
	return true
}

func allInputPaths(paths ...string) bool {
	for _, path := range paths {
		if !validInputPath(path) {
			return false
		}
	}
	return true
}

func validInputPath(path string) bool {
	return path != "" && path != "-"
}

func validOutputPath(path string) bool {
	return path != "" && path != "-"
}

func validFormat(format string) bool {
	return format == "text" || format == "json"
}

func writeValidation(output io.Writer, format string, report networkcore.ValidationReport) {
	if format == "json" {
		_ = json.NewEncoder(output).Encode(report)
		return
	}
	if report.Valid {
		_, _ = fmt.Fprintf(output, "valid %s\n", report.Kind)
		return
	}
	_, _ = fmt.Fprintf(output, "invalid %s\n", report.Kind)
}

func writeSummary(output io.Writer, format string, summary operationSummary) int {
	if format == "json" {
		if err := json.NewEncoder(output).Encode(summary); err != nil {
			return 1
		}
		return 0
	}
	_, _ = fmt.Fprintf(output, "%s %s\n", summary.Command, summary.State)
	return 0
}

func reportCommandError(stderr io.Writer, exit int, err error) int {
	printError(stderr, safeCode(err))
	return exit
}

func safeCode(err error) string {
	if err == nil {
		return "UNKNOWN_ERROR"
	}
	text := err.Error()
	if index := strings.IndexByte(text, ':'); index >= 0 {
		text = text[:index]
	}
	if index := strings.IndexByte(text, ','); index >= 0 {
		text = text[:index]
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "UNKNOWN_ERROR"
	}
	return text
}

func isReadError(err error) bool {
	return strings.HasSuffix(safeCode(err), "_READ_FAILED")
}

func printError(output io.Writer, code string) {
	_, _ = fmt.Fprintf(output, "error: %s\n", code)
}

func printUsage(output io.Writer) {
	_, _ = io.WriteString(output, "usage: cnd-network <version|validate|plan|render|simulate|report> [options]\n")
}
