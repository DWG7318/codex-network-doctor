package evidence

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dwg7318/codex-network-doctor/internal/model"
)

const (
	ManifestSchemaVersion = "1.0.0"
	maxValidationFileSize = 256 << 20
)

var (
	requiredDirectories = []string{"collectors/", "evidence/", "logs/", "validation/"}
	requiredFiles       = []string{"collectors/catalog.json", "manifest.json", "timeline.json", "validation/preflight.json"}
	fixedZipTime        = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
)

type ToolInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type CaptureInfo struct {
	ID         string    `json:"id"`
	Hostname   string    `json:"hostname"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Status     string    `json:"status"`
}

type Manifest struct {
	SchemaVersion string                  `json:"schema_version"`
	Tool          ToolInfo                `json:"tool"`
	Capture       CaptureInfo             `json:"capture"`
	Collectors    []model.CollectorResult `json:"collectors"`
	Files         []model.Artifact        `json:"files"`
}

type ZipInfo struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type ValidationIssue struct {
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

type ValidationReport struct {
	Valid        bool              `json:"valid"`
	CheckedFiles int               `json:"checked_files"`
	Issues       []ValidationIssue `json:"issues"`
}

func BuildManifest(tool ToolInfo, capture CaptureInfo, collectors []model.CollectorResult, files []model.Artifact) Manifest {
	sortedCollectors := append([]model.CollectorResult(nil), collectors...)
	for index := range sortedCollectors {
		sortedCollectors[index].Artifacts = append([]model.Artifact(nil), sortedCollectors[index].Artifacts...)
		sort.Slice(sortedCollectors[index].Artifacts, func(i, j int) bool {
			return sortedCollectors[index].Artifacts[i].Path < sortedCollectors[index].Artifacts[j].Path
		})
	}
	sort.Slice(sortedCollectors, func(i, j int) bool {
		return sortedCollectors[i].ID < sortedCollectors[j].ID
	})

	sortedFiles := append([]model.Artifact(nil), files...)
	sort.Slice(sortedFiles, func(i, j int) bool {
		return sortedFiles[i].Path < sortedFiles[j].Path
	})

	return Manifest{
		SchemaVersion: ManifestSchemaVersion,
		Tool:          tool,
		Capture:       capture,
		Collectors:    sortedCollectors,
		Files:         sortedFiles,
	}
}

func MarshalManifest(manifest Manifest) ([]byte, error) {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func BuildZip(stagingRoot, destination string) (ZipInfo, error) {
	root, err := filepath.Abs(stagingRoot)
	if err != nil {
		return ZipInfo{}, err
	}
	destinationAbsolute, err := filepath.Abs(destination)
	if err != nil {
		return ZipInfo{}, err
	}
	destinationRelative, err := filepath.Rel(root, destinationAbsolute)
	if err != nil {
		return ZipInfo{}, err
	}
	if destinationRelative == "." ||
		(destinationRelative != ".." && !strings.HasPrefix(destinationRelative, ".."+string(filepath.Separator))) {
		return ZipInfo{}, errors.New("ZIP destination cannot be inside the staging root")
	}

	entries := make(map[string]string)
	for _, directory := range requiredDirectories {
		entries[directory] = ""
	}
	err = filepath.WalkDir(root, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if filePath == root {
			return nil
		}
		relative, relativeErr := filepath.Rel(root, filePath)
		if relativeErr != nil {
			return relativeErr
		}
		name := filepath.ToSlash(relative)
		if entry.IsDir() {
			name += "/"
		}
		entries[name] = filePath
		return nil
	})
	if err != nil {
		return ZipInfo{}, err
	}

	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)

	if err := os.MkdirAll(filepath.Dir(destinationAbsolute), 0o700); err != nil {
		return ZipInfo{}, err
	}
	output, err := os.Create(destinationAbsolute)
	if err != nil {
		return ZipInfo{}, err
	}
	zipWriter := zip.NewWriter(output)
	for _, name := range names {
		if strings.HasSuffix(name, "/") {
			header := &zip.FileHeader{Name: name, Method: zip.Store}
			header.Modified = fixedZipTime
			header.SetMode(0o755 | fs.ModeDir)
			if _, err := zipWriter.CreateHeader(header); err != nil {
				_ = zipWriter.Close()
				_ = output.Close()
				return ZipInfo{}, err
			}
			continue
		}

		data, err := os.ReadFile(entries[name])
		if err != nil {
			_ = zipWriter.Close()
			_ = output.Close()
			return ZipInfo{}, err
		}
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.Modified = fixedZipTime
		header.SetMode(0o644)
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			_ = zipWriter.Close()
			_ = output.Close()
			return ZipInfo{}, err
		}
		if _, err := writer.Write(data); err != nil {
			_ = zipWriter.Close()
			_ = output.Close()
			return ZipInfo{}, err
		}
	}
	if err := zipWriter.Close(); err != nil {
		_ = output.Close()
		return ZipInfo{}, err
	}
	if err := output.Close(); err != nil {
		return ZipInfo{}, err
	}

	data, err := os.ReadFile(destinationAbsolute)
	if err != nil {
		return ZipInfo{}, err
	}
	sum := sha256.Sum256(data)
	return ZipInfo{
		Path:   destinationAbsolute,
		Size:   int64(len(data)),
		SHA256: hex.EncodeToString(sum[:]),
	}, nil
}

func ValidateZip(zipPath string) (ValidationReport, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return ValidationReport{}, err
	}
	defer func() {
		_ = reader.Close()
	}()

	report := ValidationReport{Valid: true, Issues: []ValidationIssue{}}
	entries := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		if !safeZipName(file.Name) {
			report.addIssue("UNSAFE_PATH", file.Name, "ZIP entry is absolute or escapes the package root")
			continue
		}
		if _, exists := entries[file.Name]; exists {
			report.addIssue("DUPLICATE_ENTRY", file.Name, "ZIP contains a duplicate entry")
			continue
		}
		entries[file.Name] = file
	}

	for _, directory := range requiredDirectories {
		if _, exists := entries[directory]; !exists {
			report.addIssue("MISSING_DIRECTORY", directory, "required directory entry is missing")
		}
	}
	for _, file := range requiredFiles {
		if _, exists := entries[file]; !exists {
			report.addIssue("MISSING_FILE", file, "required package file is missing")
		}
	}

	manifestFile, exists := entries["manifest.json"]
	if !exists {
		return report, nil
	}
	manifestData, readErr := readZipFile(manifestFile)
	if readErr != nil {
		report.addIssue("MANIFEST_READ_ERROR", "manifest.json", readErr.Error())
		return report, nil
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		report.addIssue("MANIFEST_INVALID_JSON", "manifest.json", err.Error())
		return report, nil
	}
	if manifest.SchemaVersion != ManifestSchemaVersion {
		report.addIssue("SCHEMA_VERSION_UNSUPPORTED", "manifest.json", fmt.Sprintf("schema version %q is not supported", manifest.SchemaVersion))
	}

	manifested := make(map[string]struct{}, len(manifest.Files))
	for _, artifact := range manifest.Files {
		if _, duplicate := manifested[artifact.Path]; duplicate {
			report.addIssue("DUPLICATE_MANIFEST_PATH", artifact.Path, "manifest contains the same file more than once")
			continue
		}
		manifested[artifact.Path] = struct{}{}
		file, found := entries[artifact.Path]
		if !found {
			report.addIssue("MISSING_ARTIFACT", artifact.Path, "manifested artifact is missing from ZIP")
			continue
		}
		data, err := readZipFile(file)
		if err != nil {
			report.addIssue("ARTIFACT_READ_ERROR", artifact.Path, err.Error())
			continue
		}
		report.CheckedFiles++
		if int64(len(data)) != artifact.Size {
			report.addIssue("SIZE_MISMATCH", artifact.Path, fmt.Sprintf("got %d bytes, manifest declares %d", len(data), artifact.Size))
		}
		sum := sha256.Sum256(data)
		actualHash := hex.EncodeToString(sum[:])
		if actualHash != strings.ToLower(artifact.SHA256) {
			report.addIssue("HASH_MISMATCH", artifact.Path, fmt.Sprintf("got %s, manifest declares %s", actualHash, artifact.SHA256))
		}
	}

	for name, file := range entries {
		if file.FileInfo().IsDir() || name == "manifest.json" {
			continue
		}
		if _, found := manifested[name]; !found {
			report.addIssue("UNMANIFESTED_FILE", name, "file is not declared in manifest")
		}
	}
	return report, nil
}

func (r *ValidationReport) addIssue(code, filePath, message string) {
	r.Valid = false
	r.Issues = append(r.Issues, ValidationIssue{Code: code, Path: filePath, Message: message})
}

func safeZipName(name string) bool {
	if name == "" || strings.Contains(name, `\`) || strings.Contains(name, ":") || strings.HasPrefix(name, "/") {
		return false
	}
	clean := path.Clean(name)
	return clean != ".." && !strings.HasPrefix(clean, "../")
}

func readZipFile(file *zip.File) ([]byte, error) {
	if file.UncompressedSize64 > maxValidationFileSize {
		return nil, fmt.Errorf("file exceeds validation limit of %d bytes", maxValidationFileSize)
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = reader.Close()
	}()
	return io.ReadAll(io.LimitReader(reader, maxValidationFileSize+1))
}
