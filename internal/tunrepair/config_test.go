package tunrepair

import (
	"bytes"
	"testing"
)

func TestEnableTunConfigChangesOnlyEnableTunValue(t *testing.T) {
	t.Parallel()

	input := []byte("{\r\n  \"IndexId\": \"keep-me\",\r\n  \"TunModeItem\": {\r\n    \"EnableTun\": false,\r\n    \"AutoRoute\": true,\r\n    \"StrictRoute\": true\r\n  },\r\n  \"Custom\": \"EnableTun false must stay\"\r\n}\r\n")

	updated, changed, err := EnableTunConfig(input)
	if err != nil {
		t.Fatalf("EnableTunConfig() error = %v", err)
	}
	if !changed {
		t.Fatal("EnableTunConfig() changed = false, want true")
	}
	if bytes.Count(updated, []byte(`"EnableTun": true`)) != 1 {
		t.Fatalf("updated config does not contain exactly one enabled TUN value:\n%s", updated)
	}
	if !bytes.Contains(updated, []byte(`"Custom": "EnableTun false must stay"`)) {
		t.Fatalf("unrelated config content changed:\n%s", updated)
	}
}

func TestEnableTunConfigLeavesAlreadyEnabledConfigByteIdentical(t *testing.T) {
	t.Parallel()

	input := []byte(`{"TunModeItem":{"EnableTun":true,"AutoRoute":true}}`)
	updated, changed, err := EnableTunConfig(input)
	if err != nil {
		t.Fatalf("EnableTunConfig() error = %v", err)
	}
	if changed {
		t.Fatal("EnableTunConfig() changed = true, want false")
	}
	if !bytes.Equal(updated, input) {
		t.Fatalf("already enabled config was modified: got %q want %q", updated, input)
	}
}

func TestEnableTunConfigRejectsMissingOrMalformedSetting(t *testing.T) {
	t.Parallel()

	for _, input := range [][]byte{
		[]byte(`{"TunModeItem":{"AutoRoute":true}}`),
		[]byte(`{"TunModeItem":{"EnableTun":"yes"}}`),
		[]byte(`not-json`),
	} {
		if _, _, err := EnableTunConfig(input); err == nil {
			t.Fatalf("EnableTunConfig(%q) error = nil, want error", input)
		}
	}
}

func TestCandidateConfigPathsCoverPortableAndLocalAppDataLayouts(t *testing.T) {
	t.Parallel()

	got := CandidateConfigPaths(
		`C:\v2rayN-windows-64\v2rayN2.exe`,
		`C:\Users\ExampleUser\AppData\Local`,
	)
	want := []string{
		`C:\v2rayN-windows-64\guiConfigs\guiNConfig.json`,
		`C:\Users\ExampleUser\AppData\Local\v2rayN\guiConfigs\guiNConfig.json`,
	}
	if len(got) != len(want) {
		t.Fatalf("CandidateConfigPaths() = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("CandidateConfigPaths()[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}
