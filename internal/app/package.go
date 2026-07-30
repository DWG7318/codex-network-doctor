package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dwg7318/codex-network-doctor/internal/engine"
	"github.com/dwg7318/codex-network-doctor/internal/evidence"
)

type PackageOptions struct {
	OutputPath string
	Force      bool
	Tool       evidence.ToolInfo
	Provider   string
	Collectors []engine.Collector
	Catalog    any
	Hostname   string
}

type PackageSummary struct {
	CaptureID            string                    `json:"capture_id"`
	Status               string                    `json:"status"`
	Provider             string                    `json:"provider"`
	OutputPath           string                    `json:"output_path"`
	Size                 int64                     `json:"size"`
	SHA256               string                    `json:"sha256"`
	SuccessfulCollectors int                       `json:"successful_collectors"`
	FailedCollectors     int                       `json:"failed_collectors"`
	TimedOutCollectors   int                       `json:"timed_out_collectors"`
	Validation           evidence.ValidationReport `json:"validation"`
}

type catalogDocument struct {
	SchemaVersion string `json:"schema_version"`
	Provider      string `json:"provider"`
	Collectors    any    `json:"collectors"`
}

type timelineDocument struct {
	SchemaVersion string `json:"schema_version"`
	Events        []any  `json:"events"`
}

type preflightDocument struct {
	SchemaVersion          string `json:"schema_version"`
	Valid                  bool   `json:"valid"`
	CollectorResultsStored int    `json:"collector_results_stored"`
	ArtifactCount          int    `json:"artifact_count"`
}

func Package(ctx context.Context, options PackageOptions) (PackageSummary, error) {
	if options.OutputPath == "" {
		return PackageSummary{}, errors.New("output path is required")
	}
	outputPath, err := filepath.Abs(options.OutputPath)
	if err != nil {
		return PackageSummary{}, err
	}
	if info, statErr := os.Stat(outputPath); statErr == nil {
		if info.IsDir() {
			return PackageSummary{}, fmt.Errorf("output path %q is a directory", outputPath)
		}
		if !options.Force {
			return PackageSummary{}, fmt.Errorf("output path %q already exists; use force to replace it", outputPath)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return PackageSummary{}, statErr
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o700); err != nil {
		return PackageSummary{}, err
	}

	stagingRoot, err := os.MkdirTemp(filepath.Dir(outputPath), ".cnd-staging-")
	if err != nil {
		return PackageSummary{}, err
	}
	defer func() {
		_ = os.RemoveAll(stagingRoot)
	}()

	store, err := evidence.NewStore(stagingRoot)
	if err != nil {
		return PackageSummary{}, err
	}
	captureID, err := newCaptureID()
	if err != nil {
		return PackageSummary{}, err
	}
	hostname := options.Hostname
	if hostname == "" {
		hostname, err = os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
	}
	startedAt := time.Now().UTC()
	results := engine.New(options.Collectors).Run(ctx, store)
	finishedAt := time.Now().UTC()

	status := "complete"
	successful := 0
	failed := 0
	timedOut := 0
	files := make([]evidence.Artifact, 0)
	for _, result := range results {
		files = append(files, result.Artifacts...)
		switch result.Status {
		case engine.StatusSuccess:
			successful++
		case engine.StatusTimedOut:
			status = "partial"
			timedOut++
		default:
			status = "partial"
			failed++
		}
	}

	catalogArtifact, err := writeJSON(store, "collectors/catalog.json", catalogDocument{
		SchemaVersion: "1.0.0",
		Provider:      options.Provider,
		Collectors:    options.Catalog,
	})
	if err != nil {
		return PackageSummary{}, err
	}
	files = append(files, catalogArtifact)

	timelineArtifact, err := writeJSON(store, "timeline.json", timelineDocument{
		SchemaVersion: "1.0.0",
		Events:        []any{},
	})
	if err != nil {
		return PackageSummary{}, err
	}
	files = append(files, timelineArtifact)

	preflightArtifact, err := writeJSON(store, "validation/preflight.json", preflightDocument{
		SchemaVersion:          "1.0.0",
		Valid:                  true,
		CollectorResultsStored: len(results),
		ArtifactCount:          len(files),
	})
	if err != nil {
		return PackageSummary{}, err
	}
	files = append(files, preflightArtifact)

	manifest := evidence.BuildManifest(
		options.Tool,
		evidence.CaptureInfo{
			ID:         captureID,
			Hostname:   hostname,
			StartedAt:  startedAt,
			FinishedAt: finishedAt,
			Status:     status,
		},
		results,
		files,
	)
	manifestBytes, err := evidence.MarshalManifest(manifest)
	if err != nil {
		return PackageSummary{}, err
	}
	if _, err := store.Write("manifest.json", manifestBytes, "application/json"); err != nil {
		return PackageSummary{}, err
	}

	zipInfo, err := evidence.BuildZip(stagingRoot, outputPath)
	if err != nil {
		return PackageSummary{}, err
	}
	validation, err := evidence.ValidateZip(outputPath)
	if err != nil {
		return PackageSummary{}, err
	}
	if !validation.Valid {
		return PackageSummary{}, fmt.Errorf("built evidence ZIP failed validation: %+v", validation.Issues)
	}

	return PackageSummary{
		CaptureID:            captureID,
		Status:               status,
		Provider:             options.Provider,
		OutputPath:           zipInfo.Path,
		Size:                 zipInfo.Size,
		SHA256:               zipInfo.SHA256,
		SuccessfulCollectors: successful,
		FailedCollectors:     failed,
		TimedOutCollectors:   timedOut,
		Validation:           validation,
	}, nil
}

func writeJSON(store *evidence.Store, relativePath string, value any) (evidence.Artifact, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return evidence.Artifact{}, err
	}
	data = append(data, '\n')
	return store.Write(relativePath, data, "application/json")
}

func newCaptureID() (string, error) {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return hex.EncodeToString(random), nil
}
