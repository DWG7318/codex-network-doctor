package proxyrepair

import (
	"fmt"
	"strings"
	"time"
)

const (
	WindowsProxyBypass     = "<local>;localhost;127.*;10.*;172.16.*;172.17.*;172.18.*;172.19.*;172.20.*;172.21.*;172.22.*;172.23.*;172.24.*;172.25.*;172.26.*;172.27.*;172.28.*;172.29.*;172.30.*;172.31.*;192.168.*"
	EnvironmentProxyBypass = "localhost,127.0.0.1,::1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"
)

type valueStore interface {
	Read(name string) (SettingValue, error)
	Write(name string, value SettingValue) error
	Delete(name string) error
}

var internetSettingNames = []string{"ProxyEnable", "ProxyServer", "ProxyOverride", "AutoConfigURL"}
var environmentSettingNames = []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY"}

func captureSettings(internet, environment valueStore, now time.Time) (SettingsSnapshot, error) {
	snapshot := SettingsSnapshot{
		SchemaVersion:    "1.0.0",
		CreatedAt:        now,
		InternetSettings: make(map[string]SettingValue, len(internetSettingNames)),
		Environment:      make(map[string]SettingValue, len(environmentSettingNames)),
	}
	for _, name := range internetSettingNames {
		value, err := internet.Read(name)
		if err != nil {
			return SettingsSnapshot{}, fmt.Errorf("read %s: %w", name, err)
		}
		snapshot.InternetSettings[name] = value
	}
	for _, name := range environmentSettingNames {
		value, err := environment.Read(name)
		if err != nil {
			return SettingsSnapshot{}, fmt.Errorf("read %s: %w", name, err)
		}
		snapshot.Environment[name] = value
	}
	return snapshot, nil
}

func restoreSettings(internet, environment valueStore, snapshot SettingsSnapshot) error {
	if snapshot.SchemaVersion != "1.0.0" {
		return fmt.Errorf("unsupported backup schema %q", snapshot.SchemaVersion)
	}
	for _, name := range internetSettingNames {
		if err := restoreValue(internet, name, snapshot.InternetSettings[name]); err != nil {
			return err
		}
	}
	for _, name := range environmentSettingNames {
		if err := restoreValue(environment, name, snapshot.Environment[name]); err != nil {
			return err
		}
	}
	return nil
}

func restoreValue(store valueStore, name string, value SettingValue) error {
	var err error
	if value.Present {
		err = store.Write(name, value)
	} else {
		err = store.Delete(name)
	}
	if err != nil {
		return fmt.Errorf("restore %s: %w", name, err)
	}
	return nil
}

func applyProxySettings(internet, environment valueStore, candidate Candidate) error {
	if !validCandidate(candidate) {
		return fmt.Errorf("invalid local proxy endpoint %s:%d", candidate.Host, candidate.Port)
	}
	proxyBypass, err := mergedBypassValue(internet, "ProxyOverride", WindowsProxyBypass, ";")
	if err != nil {
		return err
	}
	environmentBypass, err := mergedBypassValue(environment, "NO_PROXY", EnvironmentProxyBypass, ",")
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("127.0.0.1:%d", candidate.Port)
	proxyURL := "http://" + endpoint
	updates := []struct {
		store valueStore
		name  string
		value SettingValue
	}{
		{internet, "ProxyEnable", SettingValue{Present: true, Kind: "dword", DWord: 1}},
		{internet, "ProxyServer", SettingValue{Present: true, Kind: "string", String: endpoint}},
		{internet, "ProxyOverride", SettingValue{Present: true, Kind: "string", String: proxyBypass}},
		{internet, "AutoConfigURL", SettingValue{Present: true, Kind: "string", String: ""}},
		{environment, "HTTP_PROXY", SettingValue{Present: true, Kind: "string", String: proxyURL}},
		{environment, "HTTPS_PROXY", SettingValue{Present: true, Kind: "string", String: proxyURL}},
		{environment, "NO_PROXY", SettingValue{Present: true, Kind: "string", String: environmentBypass}},
	}
	for _, update := range updates {
		if err := update.store.Write(update.name, update.value); err != nil {
			return fmt.Errorf("write %s: %w", update.name, err)
		}
	}
	return nil
}

func mergedBypassValue(store valueStore, name, defaults, separator string) (string, error) {
	existing, err := store.Read(name)
	if err != nil {
		return "", fmt.Errorf("read %s before update: %w", name, err)
	}
	if existing.Present && existing.Kind != "string" && existing.Kind != "expand-string" {
		return "", fmt.Errorf("%s has unsupported type %q", name, existing.Kind)
	}
	return mergeDelimited(existing.String, defaults, separator), nil
}

func mergeDelimited(existing, defaults, separator string) string {
	seen := make(map[string]bool)
	var merged []string
	for _, source := range []string{existing, defaults} {
		for _, item := range strings.Split(source, separator) {
			item = strings.TrimSpace(item)
			key := strings.ToLower(item)
			if item == "" || seen[key] {
				continue
			}
			seen[key] = true
			merged = append(merged, item)
		}
	}
	return strings.Join(merged, separator)
}
