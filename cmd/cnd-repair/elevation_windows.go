//go:build windows

package main

import "github.com/dwg7318/codex-network-doctor/internal/tunrepair"

func ensurePlatformElevation(mode operationMode) (bool, error) {
	if mode == modeInspect {
		return false, nil
	}
	if tunrepair.IsElevated() {
		return false, nil
	}
	if err := tunrepair.RelaunchElevated([]string{string(mode)}); err != nil {
		return false, err
	}
	return true, nil
}
