//go:build windows

package proxyrepair

import (
	"net"
	"os"
	"testing"
)

func TestListProcessesIncludesCurrentProcess(t *testing.T) {
	t.Parallel()

	processes, err := listProcessesWindows()
	if err != nil {
		t.Fatalf("listProcessesWindows returned error: %v", err)
	}
	currentPID := uint32(os.Getpid())
	for _, process := range processes {
		if process.PID == currentPID {
			return
		}
	}
	t.Fatalf("current PID %d was not returned", currentPID)
}

func TestListTCPListenersIncludesCurrentLoopbackListener(t *testing.T) {
	t.Parallel()

	socket, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = socket.Close() }()
	port := socket.Addr().(*net.TCPAddr).Port

	listeners, err := listTCPListenersWindows()
	if err != nil {
		t.Fatalf("listTCPListenersWindows returned error: %v", err)
	}
	for _, listener := range listeners {
		if listener.Port == port && listener.PID == uint32(os.Getpid()) {
			return
		}
	}
	t.Fatalf("listener port %d owned by PID %d was not returned", port, os.Getpid())
}
