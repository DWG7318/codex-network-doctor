package main

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestResolveInvocationTurnsNoArgumentsIntoOneClickCollection(t *testing.T) {
	desktop := filepath.Join(t.TempDir(), "Desktop")
	resolved, oneClick, output := resolveInvocation(
		nil,
		desktop,
		"OFFLINE-PC",
		time.Date(2026, 7, 30, 9, 8, 7, 0, time.Local),
	)

	if !oneClick {
		t.Fatal("oneClick = false, want true")
	}
	wantOutput := filepath.Join(desktop, "cnd诊断结果-OFFLINE-PC-20260730-090807.zip")
	wantArgs := []string{"collect", "--provider", "powershell", "--output", wantOutput}
	if !reflect.DeepEqual(resolved, wantArgs) {
		t.Fatalf("resolved args = %v, want %v", resolved, wantArgs)
	}
	if output != wantOutput {
		t.Fatalf("output = %q, want %q", output, wantOutput)
	}
}

func TestResolveInvocationPreservesExplicitArguments(t *testing.T) {
	args := []string{"help"}
	resolved, oneClick, output := resolveInvocation(args, "cnd.exe", "host", time.Now())

	if oneClick {
		t.Fatal("oneClick = true, want false")
	}
	if output != "" {
		t.Fatalf("output = %q, want empty", output)
	}
	if !reflect.DeepEqual(resolved, args) {
		t.Fatalf("resolved args = %v, want %v", resolved, args)
	}
}
