//go:build windows

package proxyrepair

import (
	"fmt"
	"os"
	"testing"
	"time"

	"golang.org/x/sys/windows/registry"
)

func TestRegistryValueStoreRoundTripsTypesAndMissingValues(t *testing.T) {
	t.Parallel()

	path := fmt.Sprintf(
		`Software\CND\Tests\proxyrepair-%d-%d`,
		os.Getpid(),
		time.Now().UnixNano(),
	)
	store, err := openRegistryValueStore(registry.CURRENT_USER, path)
	if err != nil {
		t.Fatalf("openRegistryValueStore: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
		_ = registry.DeleteKey(registry.CURRENT_USER, path)
	})

	if err := store.Write("DWord", SettingValue{Present: true, Kind: "dword", DWord: 7}); err != nil {
		t.Fatalf("write dword: %v", err)
	}
	if err := store.Write("String", SettingValue{Present: true, Kind: "string", String: "value"}); err != nil {
		t.Fatalf("write string: %v", err)
	}

	assertSetting(t, store, "DWord", SettingValue{Present: true, Kind: "dword", DWord: 7})
	assertSetting(t, store, "String", SettingValue{Present: true, Kind: "string", String: "value"})
	assertSetting(t, store, "Missing", SettingValue{Present: false})
	if err := store.Delete("String"); err != nil {
		t.Fatalf("delete string: %v", err)
	}
	assertSetting(t, store, "String", SettingValue{Present: false})
}

func TestWindowsHostAppliesAndRestoresUsingIsolatedRegistryKeys(t *testing.T) {
	t.Parallel()

	suffix := fmt.Sprintf(`%d-%d`, os.Getpid(), time.Now().UnixNano())
	internetPath := `Software\CND\Tests\internet-` + suffix
	environmentPath := `Software\CND\Tests\environment-` + suffix
	host, err := newWindowsHostWithPaths(t.TempDir(), internetPath, environmentPath)
	if err != nil {
		t.Fatalf("newWindowsHostWithPaths: %v", err)
	}
	t.Cleanup(func() {
		_ = host.Close()
		_ = registry.DeleteKey(registry.CURRENT_USER, internetPath)
		_ = registry.DeleteKey(registry.CURRENT_USER, environmentPath)
	})
	if err := host.internet.Write("ProxyEnable", SettingValue{Present: true, Kind: "dword", DWord: 0}); err != nil {
		t.Fatalf("seed ProxyEnable: %v", err)
	}
	if err := host.internet.Write("ProxyServer", SettingValue{Present: true, Kind: "string", String: "127.0.0.1:9000"}); err != nil {
		t.Fatalf("seed ProxyServer: %v", err)
	}

	current, err := host.CurrentProxy()
	if err != nil {
		t.Fatalf("CurrentProxy: %v", err)
	}
	if current != "127.0.0.1:9000" {
		t.Fatalf("current proxy = %q", current)
	}
	snapshot, err := host.Capture(time.Now())
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if err := host.Apply(Candidate{Host: "127.0.0.1", Port: 10808}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	assertSetting(t, host.internet, "ProxyServer", SettingValue{Present: true, Kind: "string", String: "127.0.0.1:10808"})
	if err := host.Restore(snapshot); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	assertSetting(t, host.internet, "ProxyServer", SettingValue{Present: true, Kind: "string", String: "127.0.0.1:9000"})
	assertSetting(t, host.environment, "HTTPS_PROXY", SettingValue{Present: false})
}
