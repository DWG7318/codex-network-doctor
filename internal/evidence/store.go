package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/dwg7318/codex-network-doctor/internal/model"
)

// Artifact describes one immutable file in an evidence package.
type Artifact = model.Artifact

// Store writes evidence files below a single staging root.
type Store struct {
	root string
}

func NewStore(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("evidence root is required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return nil, err
	}
	return &Store{root: absolute}, nil
}

func (s *Store) Root() string {
	return s.root
}

func (s *Store) Write(relativePath string, data []byte, mediaType string) (Artifact, error) {
	clean, err := safeRelativePath(relativePath)
	if err != nil {
		return Artifact{}, err
	}
	destination := filepath.Join(s.root, clean)
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return Artifact{}, err
	}
	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return Artifact{}, err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return Artifact{}, err
	}
	if err := file.Close(); err != nil {
		return Artifact{}, err
	}

	sum := sha256.Sum256(data)
	return Artifact{
		Path:      filepath.ToSlash(clean),
		Size:      int64(len(data)),
		SHA256:    hex.EncodeToString(sum[:]),
		MediaType: mediaType,
	}, nil
}

func safeRelativePath(path string) (string, error) {
	if path == "" || strings.Contains(path, ":") || filepath.IsAbs(path) || filepath.VolumeName(path) != "" ||
		strings.HasPrefix(path, "/") || strings.HasPrefix(path, `\`) {
		return "", errors.New("evidence path must be relative")
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("evidence path escapes the staging root")
	}
	return clean, nil
}
