//go:build !windows

package oneclick

import "errors"

func DesktopDirectory() (string, error) {
	return "", errors.New("Windows known folders are unavailable on this platform")
}
