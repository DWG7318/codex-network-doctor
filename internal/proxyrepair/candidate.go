package proxyrepair

import (
	"net"
	"sort"
	"strconv"
	"strings"
)

type Candidate struct {
	Host          string
	Port          int
	PID           uint32
	ProcessName   string
	V2RayRelated  bool
	DiscoveryNote string
}

type Process struct {
	PID       uint32
	ParentPID uint32
	Name      string
}

type Listener struct {
	Address string
	Port    int
	PID     uint32
}

func BuildCandidates(processes []Process, listeners []Listener, currentProxy string) []Candidate {
	processByPID := make(map[uint32]Process, len(processes))
	for _, process := range processes {
		processByPID[process.PID] = process
	}
	related := MarkV2RayRelatedProcesses(processes)
	currentPort := proxyPort(currentProxy)
	byPort := make(map[int]Candidate)
	order := make([]int, 0)
	for _, listener := range listeners {
		address := net.ParseIP(listener.Address)
		if address == nil || (!address.IsLoopback() && !address.IsUnspecified()) {
			continue
		}
		isRelated := related[listener.PID]
		if !isRelated && listener.Port != currentPort && !isCommonProxyPort(listener.Port) {
			continue
		}
		candidate := Candidate{
			Host:         "127.0.0.1",
			Port:         listener.Port,
			PID:          listener.PID,
			ProcessName:  processByPID[listener.PID].Name,
			V2RayRelated: isRelated,
		}
		existing, exists := byPort[listener.Port]
		if !exists {
			order = append(order, listener.Port)
		}
		if !exists || (!existing.V2RayRelated && candidate.V2RayRelated) {
			byPort[listener.Port] = candidate
		}
	}
	candidates := make([]Candidate, 0, len(order))
	for _, port := range order {
		candidates = append(candidates, byPort[port])
	}
	return candidates
}

func MarkV2RayRelatedProcesses(processes []Process) map[uint32]bool {
	related := make(map[uint32]bool)
	for _, process := range processes {
		if isV2RayProcessName(process.Name) {
			related[process.PID] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for _, process := range processes {
			if !related[process.PID] && related[process.ParentPID] {
				related[process.PID] = true
				changed = true
			}
		}
	}
	return related
}

func isV2RayProcessName(name string) bool {
	normalized := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(name), ".exe"))
	switch normalized {
	case "v2rayn", "xray", "v2ray", "sing-box", "mihomo", "clash", "hysteria",
		"tuic-client", "juicity-client", "naive", "brook", "mieru":
		return true
	default:
		return false
	}
}

func RankCandidates(candidates []Candidate, currentProxy string) []Candidate {
	ranked := append([]Candidate(nil), candidates...)
	currentPort := proxyPort(currentProxy)
	sort.SliceStable(ranked, func(i, j int) bool {
		return candidateScore(ranked[i], currentPort) > candidateScore(ranked[j], currentPort)
	})
	return ranked
}

func candidateScore(candidate Candidate, currentPort int) int {
	score := 0
	if currentPort > 0 && candidate.Port == currentPort {
		score += 100
	}
	if candidate.V2RayRelated {
		score += 50
	}
	if isCommonProxyPort(candidate.Port) {
		score += 10
	}
	return score
}

func isCommonProxyPort(port int) bool {
	switch port {
	case 10808, 10809, 10802, 7890, 7891:
		return true
	default:
		return false
	}
}

func proxyPort(value string) int {
	for _, part := range strings.Split(value, ";") {
		endpoint := strings.TrimSpace(part)
		if equals := strings.IndexByte(endpoint, '='); equals >= 0 {
			endpoint = endpoint[equals+1:]
		}
		endpoint = strings.TrimPrefix(endpoint, "http://")
		host, portText, err := net.SplitHostPort(endpoint)
		if err != nil || !isLoopbackHost(host) {
			continue
		}
		port, err := strconv.Atoi(portText)
		if err == nil && port > 0 && port <= 65535 {
			return port
		}
	}
	return 0
}

func isLoopbackHost(host string) bool {
	return strings.EqualFold(host, "localhost") || net.ParseIP(host).IsLoopback()
}
