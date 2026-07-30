//go:build windows

package tunrepair

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

func IsElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

func RelaunchElevated(arguments []string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	escaped := make([]string, len(arguments))
	for index, argument := range arguments {
		escaped[index] = syscall.EscapeArg(argument)
	}
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(executable)
	if err != nil {
		return err
	}
	parameters, err := windows.UTF16PtrFromString(strings.Join(escaped, " "))
	if err != nil {
		return err
	}
	directory, err := windows.UTF16PtrFromString(filepath.Dir(executable))
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, verb, file, parameters, directory, windows.SW_SHOWNORMAL)
}
