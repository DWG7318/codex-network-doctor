//go:build windows

package oneclick

import (
	"fmt"
	"syscall"
	"unsafe"
)

var folderIDDesktop = syscall.GUID{
	Data1: 0xB4BFCC3A,
	Data2: 0xDB2C,
	Data3: 0x424C,
	Data4: [8]byte{0xB0, 0x29, 0x7F, 0xE9, 0x9A, 0x87, 0xC6, 0x41},
}

func DesktopDirectory() (string, error) {
	shell32 := syscall.NewLazyDLL("shell32.dll")
	ole32 := syscall.NewLazyDLL("ole32.dll")
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getKnownFolderPath := shell32.NewProc("SHGetKnownFolderPath")
	coTaskMemFree := ole32.NewProc("CoTaskMemFree")
	lstrlenW := kernel32.NewProc("lstrlenW")

	var rawPath *uint16
	result, _, _ := getKnownFolderPath.Call(
		uintptr(unsafe.Pointer(&folderIDDesktop)),
		0,
		0,
		uintptr(unsafe.Pointer(&rawPath)),
	)
	if result != 0 {
		return "", fmt.Errorf("SHGetKnownFolderPath(FOLDERID_Desktop) failed with HRESULT 0x%08x", uint32(result))
	}
	if rawPath == nil {
		return "", fmt.Errorf("SHGetKnownFolderPath(FOLDERID_Desktop) returned an empty path")
	}
	defer func() {
		_, _, _ = coTaskMemFree.Call(uintptr(unsafe.Pointer(rawPath)))
	}()

	pathLength, _, _ := lstrlenW.Call(uintptr(unsafe.Pointer(rawPath)))
	return syscall.UTF16ToString(unsafe.Slice(rawPath, int(pathLength))), nil
}
