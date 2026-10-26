//go:build linux

package networkhealth

import (
	"io"
	"net"
	"os"
	"runtime"
	"sort"
	"strings"
)

const hostFileLimit = 64 << 10

// Snapshot reads fixed, bounded host facts. No external commands or privilege changes.
func Snapshot() Host { return snapshot(boundedHostRead, net.Interfaces) }

func boundedHostRead(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, hostFileLimit+1))
	if len(b) > hostFileLimit {
		return nil, io.ErrShortBuffer
	}
	return b, err
}

func snapshot(read func(string) ([]byte, error), interfaces func() ([]net.Interface, error)) Host {
	h := Host{Platform: "linux", Status: "complete", Interfaces: []Interface{}, Nameservers: []string{}, IPv4Routes: []string{}, IPv6Routes: []string{}, Codes: []string{}}
	osRelease, err := read("/etc/os-release")
	if err != nil {
		h.Codes = append(h.Codes, "OS_RELEASE_UNAVAILABLE")
	} else {
		values := map[string]string{}
		for _, line := range strings.Split(string(osRelease), "\n") {
			k, v, ok := strings.Cut(line, "=")
			if ok {
				values[k] = strings.Trim(v, "\"")
			}
		}
		if values["ID"] == "ubuntu" && values["VERSION_ID"] == "24.04" && runtime.GOARCH == "amd64" {
			h.Platform = "ubuntu-24.04-amd64"
			h.Supported = true
		}
	}
	if !h.Supported {
		h.Codes = append(h.Codes, "PLATFORM_NOT_VALIDATED")
	}
	ints, err := interfaces()
	if err != nil {
		h.Codes = append(h.Codes, "INTERFACES_UNAVAILABLE")
	} else {
		for _, i := range ints {
			entry := Interface{Name: i.Name, MTU: i.MTU, Up: i.Flags&net.FlagUp != 0, Loopback: i.Flags&net.FlagLoopback != 0, Addresses: []string{}}
			addresses, err := i.Addrs()
			if err != nil {
				h.Codes = append(h.Codes, "INTERFACE_ADDRESSES_UNAVAILABLE")
			}
			for _, a := range addresses {
				entry.Addresses = append(entry.Addresses, a.String())
			}
			sort.Strings(entry.Addresses)
			h.Interfaces = append(h.Interfaces, entry)
		}
		sort.Slice(h.Interfaces, func(i, j int) bool { return h.Interfaces[i].Name < h.Interfaces[j].Name })
	}
	resolv, err := read("/etc/resolv.conf")
	if err != nil {
		h.Codes = append(h.Codes, "RESOLVERS_UNAVAILABLE")
	} else {
		for _, line := range strings.Split(string(resolv), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[0] == "nameserver" && net.ParseIP(fields[1]) != nil {
				h.Nameservers = append(h.Nameservers, fields[1])
			}
		}
		if len(h.Nameservers) == 0 {
			h.Codes = append(h.Codes, "NO_RESOLVER_OBSERVED")
		}
	}
	for _, item := range []struct {
		path, code string
		dest       *[]string
		header     bool
	}{{"/proc/net/route", "IPV4_ROUTES_UNAVAILABLE", &h.IPv4Routes, true}, {"/proc/net/ipv6_route", "IPV6_ROUTES_UNAVAILABLE", &h.IPv6Routes, false}} {
		b, err := read(item.path)
		if err != nil {
			h.Codes = append(h.Codes, item.code)
			continue
		}
		for n, line := range strings.Split(string(b), "\n") {
			if item.header && n == 0 {
				continue
			}
			if strings.TrimSpace(line) != "" {
				*item.dest = append(*item.dest, line)
			}
		}
	}
	if len(h.Codes) > 0 {
		h.Status = "partial"
	}
	sort.Strings(h.Codes)
	return h
}
