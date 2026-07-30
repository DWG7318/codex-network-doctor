package runner

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
)

type OSRunner struct{}

func (OSRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, int, error) {
	command := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return stdout.Bytes(), stderr.Bytes(), -1, ctxErr
	}
	if err == nil {
		return stdout.Bytes(), stderr.Bytes(), 0, nil
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return stdout.Bytes(), stderr.Bytes(), exitError.ExitCode(), err
	}
	return stdout.Bytes(), stderr.Bytes(), -1, err
}
