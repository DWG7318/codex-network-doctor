package runner

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestOSRunnerCapturesStdoutStderrAndExitCode(t *testing.T) {
	resultStdout, resultStderr, exitCode, err := (OSRunner{}).Run(
		context.Background(),
		os.Args[0],
		"-test.run=TestCommandHelperProcess",
		"--",
		"exit-seven",
	)

	if err == nil {
		t.Fatal("Run() error = nil, want process failure")
	}
	if exitCode != 7 {
		t.Fatalf("exit code = %d, want 7", exitCode)
	}
	if string(resultStdout) != "stdout evidence\n" {
		t.Fatalf("stdout = %q", resultStdout)
	}
	if string(resultStderr) != "stderr evidence\n" {
		t.Fatalf("stderr = %q", resultStderr)
	}
}

func TestOSRunnerHonorsContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	_, _, _, err := (OSRunner{}).Run(
		ctx,
		os.Args[0],
		"-test.run=TestCommandHelperProcess",
		"--",
		"wait",
	)

	if err != context.DeadlineExceeded {
		t.Fatalf("Run() error = %v, want context deadline exceeded", err)
	}
}

func TestCommandHelperProcess(t *testing.T) {
	separator := -1
	for index, argument := range os.Args {
		if argument == "--" {
			separator = index
			break
		}
	}
	if separator < 0 || separator+1 >= len(os.Args) {
		return
	}

	switch os.Args[separator+1] {
	case "exit-seven":
		_, _ = fmt.Fprintln(os.Stdout, "stdout evidence")
		_, _ = fmt.Fprintln(os.Stderr, "stderr evidence")
		os.Exit(7)
	case "wait":
		time.Sleep(5 * time.Second)
		os.Exit(0)
	default:
		os.Exit(9)
	}
}
