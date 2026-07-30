//go:build windows

package proxyrepair

import (
	"encoding/binary"
	"fmt"
	"net"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	tcpTableOwnerPIDAll = 5
	mibTCPStateListen   = 2
	afINET              = 2
	tcpOwnerPIDRowSize  = 24
)

var getExtendedTCPTable = windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetExtendedTcpTable")

func listProcessesWindows() ([]Process, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot: %w", err)
	}
	defer func() { _ = windows.CloseHandle(snapshot) }()

	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return nil, fmt.Errorf("Process32First: %w", err)
	}
	var processes []Process
	for {
		processes = append(processes, Process{
			PID:       entry.ProcessID,
			ParentPID: entry.ParentProcessID,
			Name:      windows.UTF16ToString(entry.ExeFile[:]),
		})
		err := windows.Process32Next(snapshot, &entry)
		if err == nil {
			continue
		}
		if err == windows.ERROR_NO_MORE_FILES {
			break
		}
		return nil, fmt.Errorf("Process32Next: %w", err)
	}
	return processes, nil
}

func listTCPListenersWindows() ([]Listener, error) {
	var size uint32
	result, _, _ := getExtendedTCPTable.Call(
		0,
		uintptr(unsafe.Pointer(&size)),
		0,
		afINET,
		tcpTableOwnerPIDAll,
		0,
	)
	if result != uintptr(windows.ERROR_INSUFFICIENT_BUFFER) && result != 0 {
		return nil, fmt.Errorf("GetExtendedTcpTable size query: %w", syscall.Errno(result))
	}
	if size < 4 {
		return nil, fmt.Errorf("GetExtendedTcpTable returned invalid size %d", size)
	}
	buffer := make([]byte, size)
	result, _, _ = getExtendedTCPTable.Call(
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
		afINET,
		tcpTableOwnerPIDAll,
		0,
	)
	if result != 0 {
		return nil, fmt.Errorf("GetExtendedTcpTable: %w", syscall.Errno(result))
	}

	rowCount := int(binary.LittleEndian.Uint32(buffer[:4]))
	if 4+(rowCount*tcpOwnerPIDRowSize) > len(buffer) {
		return nil, fmt.Errorf("GetExtendedTcpTable returned truncated data")
	}
	listeners := make([]Listener, 0, rowCount)
	for index := 0; index < rowCount; index++ {
		offset := 4 + index*tcpOwnerPIDRowSize
		state := binary.LittleEndian.Uint32(buffer[offset : offset+4])
		if state != mibTCPStateListen {
			continue
		}
		address := net.IP(append([]byte(nil), buffer[offset+4:offset+8]...)).String()
		port := int(binary.BigEndian.Uint16(buffer[offset+8 : offset+10]))
		pid := binary.LittleEndian.Uint32(buffer[offset+20 : offset+24])
		listeners = append(listeners, Listener{Address: address, Port: port, PID: pid})
	}
	return listeners, nil
}
