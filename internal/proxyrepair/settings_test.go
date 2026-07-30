package proxyrepair

import (
	"strings"
	"testing"
	"time"
)

func TestApplyProxySettingsUsesV2RayNCompatibleValues(t *testing.T) {
	t.Parallel()

	internet := newMemoryValueStore()
	environment := newMemoryValueStore()

	err := applyProxySettings(
		internet,
		environment,
		Candidate{Host: "127.0.0.1", Port: 10808},
	)
	if err != nil {
		t.Fatalf("applyProxySettings returned error: %v", err)
	}

	assertSetting(t, internet, "ProxyEnable", SettingValue{Present: true, Kind: "dword", DWord: 1})
	assertSetting(t, internet, "ProxyServer", SettingValue{Present: true, Kind: "string", String: "127.0.0.1:10808"})
	assertSetting(t, internet, "ProxyOverride", SettingValue{Present: true, Kind: "string", String: WindowsProxyBypass})
	assertSetting(t, internet, "AutoConfigURL", SettingValue{Present: true, Kind: "string", String: ""})
	assertSetting(t, environment, "HTTP_PROXY", SettingValue{Present: true, Kind: "string", String: "http://127.0.0.1:10808"})
	assertSetting(t, environment, "HTTPS_PROXY", SettingValue{Present: true, Kind: "string", String: "http://127.0.0.1:10808"})
	assertSetting(t, environment, "NO_PROXY", SettingValue{Present: true, Kind: "string", String: EnvironmentProxyBypass})
}

func TestRestoreSettingsPreservesPresentAndAbsentValues(t *testing.T) {
	t.Parallel()

	internet := newMemoryValueStore()
	environment := newMemoryValueStore()
	internet.values["ProxyEnable"] = SettingValue{Present: true, Kind: "dword", DWord: 0}
	internet.values["ProxyServer"] = SettingValue{Present: true, Kind: "string", String: "old:9000"}
	environment.values["HTTP_PROXY"] = SettingValue{Present: true, Kind: "string", String: "http://old:9000"}
	now := time.Date(2026, 7, 30, 8, 30, 0, 0, time.UTC)

	snapshot, err := captureSettings(internet, environment, now)
	if err != nil {
		t.Fatalf("captureSettings returned error: %v", err)
	}
	if err := applyProxySettings(internet, environment, Candidate{Host: "127.0.0.1", Port: 10808}); err != nil {
		t.Fatalf("applyProxySettings returned error: %v", err)
	}
	if err := restoreSettings(internet, environment, snapshot); err != nil {
		t.Fatalf("restoreSettings returned error: %v", err)
	}

	assertSetting(t, internet, "ProxyEnable", SettingValue{Present: true, Kind: "dword", DWord: 0})
	assertSetting(t, internet, "ProxyServer", SettingValue{Present: true, Kind: "string", String: "old:9000"})
	assertSetting(t, internet, "ProxyOverride", SettingValue{Present: false})
	assertSetting(t, internet, "AutoConfigURL", SettingValue{Present: false})
	assertSetting(t, environment, "HTTP_PROXY", SettingValue{Present: true, Kind: "string", String: "http://old:9000"})
	assertSetting(t, environment, "HTTPS_PROXY", SettingValue{Present: false})
	assertSetting(t, environment, "NO_PROXY", SettingValue{Present: false})
}

func TestApplyProxySettingsPreservesExistingBypassEntries(t *testing.T) {
	t.Parallel()

	internet := newMemoryValueStore()
	environment := newMemoryValueStore()
	internet.values["ProxyOverride"] = SettingValue{
		Present: true,
		Kind:    "string",
		String:  "*.corp.example;<local>",
	}
	environment.values["NO_PROXY"] = SettingValue{
		Present: true,
		Kind:    "string",
		String:  ".corp.example,localhost",
	}

	if err := applyProxySettings(internet, environment, Candidate{Host: "127.0.0.1", Port: 10808}); err != nil {
		t.Fatalf("applyProxySettings returned error: %v", err)
	}

	override, _ := internet.Read("ProxyOverride")
	for _, entry := range []string{"*.corp.example", "<local>", "192.168.*"} {
		if !containsDelimitedEntry(override.String, ";", entry) {
			t.Errorf("ProxyOverride %q does not preserve %q", override.String, entry)
		}
	}
	noProxy, _ := environment.Read("NO_PROXY")
	for _, entry := range []string{".corp.example", "localhost", "192.168.0.0/16"} {
		if !containsDelimitedEntry(noProxy.String, ",", entry) {
			t.Errorf("NO_PROXY %q does not preserve %q", noProxy.String, entry)
		}
	}
}

type memoryValueStore struct {
	values map[string]SettingValue
}

func newMemoryValueStore() *memoryValueStore {
	return &memoryValueStore{values: make(map[string]SettingValue)}
}

func (store *memoryValueStore) Read(name string) (SettingValue, error) {
	value, ok := store.values[name]
	if !ok {
		return SettingValue{Present: false}, nil
	}
	return value, nil
}

func (store *memoryValueStore) Write(name string, value SettingValue) error {
	store.values[name] = value
	return nil
}

func (store *memoryValueStore) Delete(name string) error {
	delete(store.values, name)
	return nil
}

func assertSetting(t *testing.T, store valueStore, name string, want SettingValue) {
	t.Helper()
	got, err := store.Read(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	if got != want {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func containsDelimitedEntry(value, separator, want string) bool {
	for _, entry := range strings.Split(value, separator) {
		if strings.EqualFold(strings.TrimSpace(entry), want) {
			return true
		}
	}
	return false
}
