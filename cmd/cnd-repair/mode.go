package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

type operationMode string

const (
	modeRepair   operationMode = "repair"
	modeRollback operationMode = "rollback"
	modeInspect  operationMode = "inspect"
	modeSpeed    operationMode = "speed"
)

func resolveMode(args []string, executableName string) (operationMode, error) {
	if len(args) == 0 {
		baseName := strings.ToLower(filepath.Base(executableName))
		if strings.Contains(baseName, "撤销") {
			return modeRollback, nil
		}
		if strings.Contains(baseName, "网速检测") || strings.Contains(baseName, "speed") {
			return modeSpeed, nil
		}
		return modeRepair, nil
	}
	if len(args) != 1 {
		return "", fmt.Errorf("expected at most one operation argument")
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "repair":
		return modeRepair, nil
	case "rollback", "undo":
		return modeRollback, nil
	case "inspect":
		return modeInspect, nil
	case "speed", "speed-test":
		return modeSpeed, nil
	default:
		return "", fmt.Errorf("unknown operation %q", args[0])
	}
}
