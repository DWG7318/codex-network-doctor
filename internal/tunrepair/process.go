package tunrepair

import (
	"errors"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var v2rayNGUIName = regexp.MustCompile(`(?i)^v2rayn(?:[-_.]?\d+)?$`)

type Process struct {
	PID       uint32
	ParentPID uint32
	Name      string
	Path      string
	StartedAt time.Time
}

type Listener struct {
	Address string
	Port    int
	PID     uint32
}

type TerminationPlan struct {
	Terminate     []Process
	UnknownOwners []Process
}

func IsV2RayNGUIName(name string) bool {
	base := strings.TrimSuffix(strings.TrimSpace(name), filepath.Ext(name))
	return v2rayNGUIName.MatchString(base)
}

func SelectKeeper(processes []Process) (Process, []Process, error) {
	var instances []Process
	for _, process := range processes {
		if IsV2RayNGUIName(process.Name) && process.Path != "" {
			instances = append(instances, process)
		}
	}
	if len(instances) == 0 {
		return Process{}, nil, errors.New("未发现正在运行的 v2rayN")
	}
	sort.SliceStable(instances, func(i, j int) bool {
		if instances[i].StartedAt.Equal(instances[j].StartedAt) {
			return instances[i].PID > instances[j].PID
		}
		return instances[i].StartedAt.After(instances[j].StartedAt)
	})
	return instances[0], append([]Process(nil), instances[1:]...), nil
}

func PlanTermination(
	processes []Process,
	listeners []Listener,
	keeper Process,
	controllerPorts []int,
) TerminationPlan {
	processByPID := make(map[uint32]Process, len(processes))
	guiDirectories := make(map[string]bool)
	related := make(map[uint32]bool)
	for _, process := range processes {
		processByPID[process.PID] = process
		if IsV2RayNGUIName(process.Name) && process.Path != "" {
			guiDirectories[strings.ToLower(filepath.Clean(filepath.Dir(process.Path)))] = true
			related[process.PID] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for _, process := range processes {
			if !related[process.PID] &&
				related[process.ParentPID] &&
				isKnownCoreName(process.Name) {
				related[process.PID] = true
				changed = true
			}
		}
	}
	for _, process := range processes {
		if related[process.PID] || !isKnownCoreName(process.Name) || process.Path == "" {
			continue
		}
		for directory := range guiDirectories {
			if pathWithin(process.Path, filepath.Join(directory, "bin")) {
				related[process.PID] = true
				break
			}
		}
	}

	var plan TerminationPlan
	for _, process := range processes {
		if related[process.PID] {
			plan.Terminate = append(plan.Terminate, process)
		}
	}
	sort.SliceStable(plan.Terminate, func(i, j int) bool {
		if plan.Terminate[i].PID == keeper.PID {
			return false
		}
		if plan.Terminate[j].PID == keeper.PID {
			return true
		}
		return plan.Terminate[i].PID < plan.Terminate[j].PID
	})

	targetPorts := make(map[int]bool, len(controllerPorts))
	for _, port := range controllerPorts {
		targetPorts[port] = true
	}
	unknownSeen := make(map[uint32]bool)
	for _, listener := range listeners {
		if !targetPorts[listener.Port] || related[listener.PID] || unknownSeen[listener.PID] {
			continue
		}
		if owner, exists := processByPID[listener.PID]; exists {
			plan.UnknownOwners = append(plan.UnknownOwners, owner)
			unknownSeen[listener.PID] = true
		}
	}
	return plan
}

func isKnownCoreName(name string) bool {
	normalized := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(name), filepath.Ext(name)))
	switch normalized {
	case "xray", "v2ray", "sing-box", "mihomo", "clash", "hysteria", "hysteria2",
		"tuic-client", "juicity-client", "naive", "brook", "mieru":
		return true
	default:
		return false
	}
}

func pathWithin(path string, directory string) bool {
	cleanPath := strings.ToLower(filepath.Clean(path))
	cleanDirectory := strings.TrimSuffix(strings.ToLower(filepath.Clean(directory)), string(filepath.Separator))
	return cleanPath == cleanDirectory || strings.HasPrefix(cleanPath, cleanDirectory+string(filepath.Separator))
}
