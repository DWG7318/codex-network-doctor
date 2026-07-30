//go:build !windows

package main

func ensurePlatformElevation(operationMode) (bool, error) {
	return false, nil
}
