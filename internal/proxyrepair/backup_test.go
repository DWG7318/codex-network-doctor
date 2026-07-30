package proxyrepair

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestFileBackupStoreSavesHistoryAndLoadsLatest(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	store := fileBackupStore{directory: directory}
	snapshot := SettingsSnapshot{
		SchemaVersion: "1.0.0",
		CreatedAt:     time.Date(2026, 7, 30, 9, 1, 2, 123, time.UTC),
		InternetSettings: map[string]SettingValue{
			"ProxyEnable": {Present: true, Kind: "dword", DWord: 0},
		},
		Environment: map[string]SettingValue{
			"HTTPS_PROXY": {Present: false},
		},
	}

	historyPath, err := store.Save(snapshot)
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if filepath.Dir(historyPath) != directory {
		t.Fatalf("history directory = %q, want %q", filepath.Dir(historyPath), directory)
	}

	loaded, latestPath, err := store.LoadLatest()
	if err != nil {
		t.Fatalf("LoadLatest returned error: %v", err)
	}
	if latestPath != filepath.Join(directory, "latest.json") {
		t.Fatalf("latest path = %q", latestPath)
	}
	if !reflect.DeepEqual(loaded, snapshot) {
		t.Fatalf("loaded snapshot = %#v, want %#v", loaded, snapshot)
	}
}
