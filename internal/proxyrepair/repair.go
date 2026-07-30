package proxyrepair

import (
	"context"
	"fmt"
	"time"
)

type SettingValue struct {
	Present bool   `json:"present"`
	Kind    string `json:"kind,omitempty"`
	String  string `json:"string,omitempty"`
	DWord   uint32 `json:"dword,omitempty"`
}

type SettingsSnapshot struct {
	SchemaVersion    string                  `json:"schema_version"`
	CreatedAt        time.Time               `json:"created_at"`
	InternetSettings map[string]SettingValue `json:"internet_settings"`
	Environment      map[string]SettingValue `json:"environment"`
}

type Check struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code,omitempty"`
	Error      string `json:"error,omitempty"`
}

type Result struct {
	Proxy        Candidate
	BackupPath   string
	Checks       []Check
	Warnings     []string
	RolledBack   bool
	RestoredFrom string
}

type Options struct {
	VerificationURLs []string
}

func DefaultOptions() Options {
	return Options{
		VerificationURLs: []string{
			"https://chatgpt.com/",
			"https://api.openai.com/v1/models",
		},
	}
}

type Host interface {
	CurrentProxy() (string, error)
	DiscoverCandidates(context.Context) ([]Candidate, error)
	Probe(context.Context, Candidate) error
	Capture(time.Time) (SettingsSnapshot, error)
	SaveBackup(SettingsSnapshot) (string, error)
	Apply(Candidate) error
	Refresh() error
	FlushDNS() error
	Verify(context.Context, Candidate, string) (int, error)
	Restore(SettingsSnapshot) error
	LoadLatestBackup() (SettingsSnapshot, string, error)
}

func Execute(ctx context.Context, host Host, options Options, now time.Time) (Result, error) {
	currentProxy, err := host.CurrentProxy()
	if err != nil {
		return Result{}, fmt.Errorf("read current proxy: %w", err)
	}
	candidates, err := host.DiscoverCandidates(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("discover local proxy listeners: %w", err)
	}

	var selected Candidate
	found := false
	for _, candidate := range RankCandidates(candidates, currentProxy) {
		if !validCandidate(candidate) {
			continue
		}
		if err := host.Probe(ctx, candidate); err == nil {
			selected = candidate
			found = true
			break
		}
	}
	if !found {
		return Result{}, fmt.Errorf("no working v2rayN HTTP or Mixed proxy listener was found")
	}

	snapshot, err := host.Capture(now)
	if err != nil {
		return Result{}, fmt.Errorf("capture current settings: %w", err)
	}
	backupPath, err := host.SaveBackup(snapshot)
	if err != nil {
		return Result{}, fmt.Errorf("save rollback backup: %w", err)
	}
	result := Result{Proxy: selected, BackupPath: backupPath}
	if err := host.Apply(selected); err != nil {
		result.RolledBack = true
		if restoreErr := host.Restore(snapshot); restoreErr != nil {
			return result, fmt.Errorf("apply proxy settings: %v; rollback failed: %w", err, restoreErr)
		}
		if refreshErr := host.Refresh(); refreshErr != nil {
			return result, fmt.Errorf("apply proxy settings: %v; settings restored but refresh failed: %w", err, refreshErr)
		}
		return result, fmt.Errorf("apply proxy settings: %w", err)
	}
	if err := host.Refresh(); err != nil {
		result.RolledBack = true
		if restoreErr := host.Restore(snapshot); restoreErr != nil {
			return result, fmt.Errorf("refresh proxy settings: %v; rollback failed: %w", err, restoreErr)
		}
		if refreshErr := host.Refresh(); refreshErr != nil {
			return result, fmt.Errorf("refresh proxy settings: %v; settings restored but rollback refresh failed: %w", err, refreshErr)
		}
		return result, fmt.Errorf("refresh proxy settings: %w", err)
	}

	if err := host.FlushDNS(); err != nil {
		result.Warnings = append(result.Warnings, "DNS cache flush failed: "+err.Error())
	}
	successfulChecks := 0
	for _, target := range options.VerificationURLs {
		statusCode, verifyErr := host.Verify(ctx, selected, target)
		check := Check{URL: target, StatusCode: statusCode}
		if verifyErr != nil {
			check.Error = verifyErr.Error()
		} else {
			successfulChecks++
		}
		result.Checks = append(result.Checks, check)
	}
	if successfulChecks == 0 {
		result.RolledBack = true
		if restoreErr := host.Restore(snapshot); restoreErr != nil {
			return result, fmt.Errorf("OpenAI verification failed and rollback failed: %w", restoreErr)
		}
		if refreshErr := host.Refresh(); refreshErr != nil {
			return result, fmt.Errorf("OpenAI verification failed; settings were restored but refresh failed: %w", refreshErr)
		}
		return result, fmt.Errorf("OpenAI verification failed; original proxy settings were restored")
	}
	return result, nil
}

func RollbackLatest(host Host) (Result, error) {
	snapshot, backupPath, err := host.LoadLatestBackup()
	if err != nil {
		return Result{}, fmt.Errorf("load latest rollback backup: %w", err)
	}
	result := Result{RestoredFrom: backupPath}
	if err := host.Restore(snapshot); err != nil {
		return result, fmt.Errorf("restore proxy settings: %w", err)
	}
	if err := host.Refresh(); err != nil {
		return result, fmt.Errorf("refresh restored proxy settings: %w", err)
	}
	if err := host.FlushDNS(); err != nil {
		result.Warnings = append(result.Warnings, "DNS cache flush failed: "+err.Error())
	}
	return result, nil
}

func validCandidate(candidate Candidate) bool {
	return candidate.Port > 0 && candidate.Port <= 65535 && isLoopbackHost(candidate.Host)
}
