package casebook

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func TestEmbeddedTunReservedPortCaseCanBeListedAndRead(t *testing.T) {
	t.Parallel()

	cases := List()
	if len(cases) == 0 {
		t.Fatal("List() returned no embedded cases")
	}

	const id = "CND-CASE-2026-001"
	content, ok := Lookup(id)
	if !ok {
		t.Fatalf("Lookup(%q) did not find the embedded case", id)
	}
	for _, required := range []string{
		"v2rayN",
		"TUN",
		"excluded port range",
		"HTTP 401",
		"脱敏",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("case content does not contain %q", required)
		}
	}
}

func TestEmbeddedCasesDoNotContainIncidentIdentifiers(t *testing.T) {
	t.Parallel()

	for _, incidentIdentifier := range []string{
		"wxid_",
		"xwechat_files",
		"C:\\Users\\",
	} {
		for _, entry := range List() {
			content, ok := Lookup(entry.ID)
			if !ok {
				t.Fatalf("listed case %q cannot be read", entry.ID)
			}
			if strings.Contains(content, incidentIdentifier) {
				t.Errorf("case %q contains incident identifier %q", entry.ID, incidentIdentifier)
			}
		}
	}

	publicIPv4 := regexp.MustCompile(
		`\b(?:(?:[1-9]\d?|1\d\d|2[0-4]\d|25[0-5])\.){3}` +
			`(?:[1-9]\d?|1\d\d|2[0-4]\d|25[0-5])\b`,
	)
	for _, entry := range List() {
		content, ok := Lookup(entry.ID)
		if !ok {
			t.Fatalf("listed case %q cannot be read", entry.ID)
		}
		for _, candidate := range publicIPv4.FindAllString(content, -1) {
			if candidate != "127.0.0.1" {
				t.Errorf("case %q contains public IPv4 candidate %q", entry.ID, candidate)
			}
		}
	}
}

func TestRunPrintsCaseListAndSelectedCase(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"cases"}, &stdout, &stderr); code != 0 {
		t.Fatalf("cases exit code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "CND-CASE-2026-001") {
		t.Fatalf("cases output = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"case", "CND-CASE-2026-001"}, &stdout, &stderr); code != 0 {
		t.Fatalf("case exit code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "HTTP 401") {
		t.Fatalf("case output = %q", stdout.String())
	}
}
