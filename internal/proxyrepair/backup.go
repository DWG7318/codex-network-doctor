package proxyrepair

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type fileBackupStore struct {
	directory string
}

func (store fileBackupStore) Save(snapshot SettingsSnapshot) (string, error) {
	if err := os.MkdirAll(store.directory, 0o700); err != nil {
		return "", fmt.Errorf("create backup directory: %w", err)
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode backup: %w", err)
	}
	data = append(data, '\n')

	baseName := "proxy-backup-" + snapshot.CreatedAt.Format("20060102-150405.000000000")
	historyPath := ""
	for suffix := 0; suffix < 100; suffix++ {
		name := baseName + ".json"
		if suffix > 0 {
			name = fmt.Sprintf("%s-%d.json", baseName, suffix)
		}
		candidate := filepath.Join(store.directory, name)
		file, openErr := os.OpenFile(candidate, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if os.IsExist(openErr) {
			continue
		}
		if openErr != nil {
			return "", fmt.Errorf("create history backup: %w", openErr)
		}
		if _, writeErr := file.Write(data); writeErr != nil {
			_ = file.Close()
			return "", fmt.Errorf("write history backup: %w", writeErr)
		}
		if closeErr := file.Close(); closeErr != nil {
			return "", fmt.Errorf("close history backup: %w", closeErr)
		}
		historyPath = candidate
		break
	}
	if historyPath == "" {
		return "", fmt.Errorf("could not allocate a unique history backup name")
	}
	if err := os.WriteFile(filepath.Join(store.directory, "latest.json"), data, 0o600); err != nil {
		return "", fmt.Errorf("write latest backup: %w", err)
	}
	return historyPath, nil
}

func (store fileBackupStore) LoadLatest() (SettingsSnapshot, string, error) {
	path := filepath.Join(store.directory, "latest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return SettingsSnapshot{}, path, fmt.Errorf("read latest backup: %w", err)
	}
	var snapshot SettingsSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return SettingsSnapshot{}, path, fmt.Errorf("decode latest backup: %w", err)
	}
	if snapshot.SchemaVersion != "1.0.0" {
		return SettingsSnapshot{}, path, fmt.Errorf("unsupported backup schema %q", snapshot.SchemaVersion)
	}
	return snapshot, path, nil
}
