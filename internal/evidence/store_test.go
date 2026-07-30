package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreWritePersistsArtifactAndComputesSHA256(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	payload := []byte(`{"status":"ok"}` + "\n")
	artifact, err := store.Write("evidence/system.json", payload, "application/json")
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(root, "evidence", "system.json"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("persisted payload = %q, want %q", got, payload)
	}

	sum := sha256.Sum256(payload)
	if artifact.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("SHA256 = %q, want %q", artifact.SHA256, hex.EncodeToString(sum[:]))
	}
	if artifact.Path != "evidence/system.json" {
		t.Fatalf("Path = %q, want evidence/system.json", artifact.Path)
	}
	if artifact.Size != int64(len(payload)) {
		t.Fatalf("Size = %d, want %d", artifact.Size, len(payload))
	}
	if artifact.MediaType != "application/json" {
		t.Fatalf("MediaType = %q, want application/json", artifact.MediaType)
	}
}

func TestStoreWriteRejectsPathsOutsideRoot(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	for _, path := range []string{
		"../escape.txt",
		"/absolute.txt",
		`C:\absolute.txt`,
		`C:relative.txt`,
		"evidence/file.txt:stream",
		"evidence/../../escape.txt",
	} {
		t.Run(path, func(t *testing.T) {
			if _, err := store.Write(path, []byte("unsafe"), "text/plain"); err == nil {
				t.Fatalf("Write(%q) error = nil, want rejection", path)
			}
		})
	}
}

func TestStoreWriteRejectsOverwritingExistingEvidence(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if _, err := store.Write("evidence/immutable.txt", []byte("first"), "text/plain"); err != nil {
		t.Fatalf("first Write() error = %v", err)
	}

	if _, err := store.Write("evidence/immutable.txt", []byte("second"), "text/plain"); err == nil {
		t.Fatal("second Write() error = nil, want overwrite rejection")
	}
	data, err := os.ReadFile(filepath.Join(store.Root(), "evidence", "immutable.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "first" {
		t.Fatalf("existing evidence was modified: %q", data)
	}
}
