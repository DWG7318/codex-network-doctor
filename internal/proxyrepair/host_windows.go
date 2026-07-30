//go:build windows

package proxyrepair

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	internetSettingsRegistryPath = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`
	environmentRegistryPath      = `Environment`
)

var (
	internetSetOptionW    = windows.NewLazySystemDLL("wininet.dll").NewProc("InternetSetOptionW")
	dnsFlushResolverCache = windows.NewLazySystemDLL("dnsapi.dll").NewProc("DnsFlushResolverCache")
	sendMessageTimeoutW   = windows.NewLazySystemDLL("user32.dll").NewProc("SendMessageTimeoutW")
)

type WindowsHost struct {
	internet    *registryValueStore
	environment *registryValueStore
	backups     fileBackupStore
}

func NewWindowsHost(backupDirectory string) (*WindowsHost, error) {
	if backupDirectory == "" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			var err error
			base, err = os.UserConfigDir()
			if err != nil {
				return nil, fmt.Errorf("locate user configuration directory: %w", err)
			}
		}
		backupDirectory = filepath.Join(base, "CND", "proxy-repair", "backups")
	}
	return newWindowsHostWithPaths(
		backupDirectory,
		internetSettingsRegistryPath,
		environmentRegistryPath,
	)
}

func newWindowsHostWithPaths(
	backupDirectory string,
	internetPath string,
	environmentPath string,
) (*WindowsHost, error) {
	internet, err := openRegistryValueStore(registry.CURRENT_USER, internetPath)
	if err != nil {
		return nil, fmt.Errorf("open Internet Settings registry: %w", err)
	}
	environment, err := openRegistryValueStore(registry.CURRENT_USER, environmentPath)
	if err != nil {
		_ = internet.Close()
		return nil, fmt.Errorf("open user Environment registry: %w", err)
	}
	return &WindowsHost{
		internet:    internet,
		environment: environment,
		backups:     fileBackupStore{directory: backupDirectory},
	}, nil
}

func (host *WindowsHost) Close() error {
	internetErr := host.internet.Close()
	environmentErr := host.environment.Close()
	if internetErr != nil {
		return internetErr
	}
	return environmentErr
}

func (host *WindowsHost) CurrentProxy() (string, error) {
	value, err := host.internet.Read("ProxyServer")
	if err != nil {
		return "", err
	}
	if !value.Present {
		return "", nil
	}
	if value.Kind != "string" && value.Kind != "expand-string" {
		return "", fmt.Errorf("ProxyServer has unexpected type %q", value.Kind)
	}
	return value.String, nil
}

func (host *WindowsHost) DiscoverCandidates(context.Context) ([]Candidate, error) {
	currentProxy, err := host.CurrentProxy()
	if err != nil {
		return nil, err
	}
	processes, err := listProcessesWindows()
	if err != nil {
		return nil, err
	}
	listeners, err := listTCPListenersWindows()
	if err != nil {
		return nil, err
	}
	return BuildCandidates(processes, listeners, currentProxy), nil
}

func (host *WindowsHost) Probe(ctx context.Context, candidate Candidate) error {
	probeContext, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	return probeHTTPProxy(probeContext, candidate)
}

func (host *WindowsHost) Capture(now time.Time) (SettingsSnapshot, error) {
	return captureSettings(host.internet, host.environment, now)
}

func (host *WindowsHost) SaveBackup(snapshot SettingsSnapshot) (string, error) {
	return host.backups.Save(snapshot)
}

func (host *WindowsHost) Apply(candidate Candidate) error {
	return applyProxySettings(host.internet, host.environment, candidate)
}

func (host *WindowsHost) Refresh() error {
	if result, _, callErr := internetSetOptionW.Call(0, 39, 0, 0); result == 0 {
		return fmt.Errorf("InternetSetOptionW(SETTINGS_CHANGED): %w", callErr)
	}
	if result, _, callErr := internetSetOptionW.Call(0, 37, 0, 0); result == 0 {
		return fmt.Errorf("InternetSetOptionW(REFRESH): %w", callErr)
	}
	environment, err := windows.UTF16PtrFromString("Environment")
	if err == nil {
		var messageResult uintptr
		_, _, _ = sendMessageTimeoutW.Call(
			0xffff,
			0x001a,
			0,
			uintptr(unsafe.Pointer(environment)),
			0x0002,
			5000,
			uintptr(unsafe.Pointer(&messageResult)),
		)
	}
	return nil
}

func (host *WindowsHost) FlushDNS() error {
	if result, _, callErr := dnsFlushResolverCache.Call(); result == 0 {
		return fmt.Errorf("DnsFlushResolverCache: %w", callErr)
	}
	return nil
}

func (host *WindowsHost) Verify(
	ctx context.Context,
	candidate Candidate,
	target string,
) (int, error) {
	verifyContext, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return verifyThroughProxy(verifyContext, candidate, target, nil)
}

func (host *WindowsHost) Restore(snapshot SettingsSnapshot) error {
	return restoreSettings(host.internet, host.environment, snapshot)
}

func (host *WindowsHost) LoadLatestBackup() (SettingsSnapshot, string, error) {
	return host.backups.LoadLatest()
}
