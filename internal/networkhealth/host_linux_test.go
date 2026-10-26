//go:build linux

package networkhealth

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHostSnapshotPartialAndBounded(t *testing.T) {
	reads := []string{}
	h := snapshot(func(path string) ([]byte, error) {
		reads = append(reads, path)
		switch path {
		case "/etc/os-release":
			return []byte("ID=ubuntu\nVERSION_ID=\"24.04\""), nil
		case "/etc/resolv.conf":
			return []byte("nameserver 127.0.0.53\nsearch private.invalid"), nil
		}
		return nil, os.ErrPermission
	}, func() ([]net.Interface, error) { return nil, nil })
	if !h.Supported || h.Status != "partial" || len(reads) != 4 || len(h.Nameservers) != 1 || h.Nameservers[0] != "127.0.0.53" {
		t.Fatalf("host=%+v reads=%v", h, reads)
	}
	p := filepath.Join(t.TempDir(), "oversized")
	if err := os.WriteFile(p, []byte(strings.Repeat("x", hostFileLimit+1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := boundedHostRead(p); err != io.ErrShortBuffer {
		t.Fatalf("err=%v", err)
	}
}

func TestLocalLinuxSnapshot(t *testing.T) {
	h := Snapshot()
	if h.Platform == "unsupported" || len(h.Interfaces) == 0 {
		t.Fatalf("host observations unavailable: %v", h.Codes)
	}
	// Report only non-sensitive counts to test output.
	t.Logf("platform=%s supported=%v status=%s interfaces=%d resolvers=%d", h.Platform, h.Supported, h.Status, len(h.Interfaces), len(h.Nameservers))
}
