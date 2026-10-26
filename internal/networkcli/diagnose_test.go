package networkcli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiagnoseRejectsInputBeforeNetwork(t *testing.T) {
	p := filepath.Join(t.TempDir(), "targets.json")
	if err := os.WriteFile(p, []byte(`{"token":"DO_NOT_ECHO_PRIVATE"}`), 0600); err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	code := Run(context.Background(), []string{"diagnose", "--input", p, "--output", filepath.Join(t.TempDir(), "report.json"), "--text-output", filepath.Join(t.TempDir(), "report.txt")}, &out, &errs, "test")
	if code != 2 || !strings.Contains(errs.String(), "HEALTH_INPUT_INVALID") || strings.Contains(out.String()+errs.String(), "DO_NOT_ECHO_PRIVATE") {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, &out, &errs)
	}
}

func TestHealthOutputReservationNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "report.json")
	second := filepath.Join(dir, "report.txt")
	if err := os.WriteFile(second, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := reserveHealthOutputs(p, second); err == nil {
		t.Fatal("accepted existing output")
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("left partial output")
	}
	b, _ := os.ReadFile(second)
	if string(b) != "preserve" {
		t.Fatal("overwrote output")
	}
	if _, err := reserveHealthOutputs(p, p); err == nil {
		t.Fatal("accepted duplicate output")
	}
}

func TestHealthReportsRequirePrivateRegularFileMode(t *testing.T) {
	for _, mode := range []os.FileMode{0644, 0666, 0777, 0400, os.ModeDir | 0600, os.ModeSymlink | 0600} {
		if privateHealthMode(mode) {
			t.Errorf("accepted mode %v", mode)
		}
	}
	if !privateHealthMode(0600) {
		t.Fatal("rejected private regular file")
	}
}
