package proxyrepair

import "testing"

func TestRankCandidatesPrefersCurrentV2RayNEndpoint(t *testing.T) {
	t.Parallel()

	candidates := []Candidate{
		{Host: "127.0.0.1", Port: 7890, PID: 10, ProcessName: "other.exe"},
		{Host: "127.0.0.1", Port: 10808, PID: 20, ProcessName: "xray.exe", V2RayRelated: true},
		{Host: "127.0.0.1", Port: 10809, PID: 21, ProcessName: "xray.exe", V2RayRelated: true},
	}

	ranked := RankCandidates(candidates, "127.0.0.1:10809")

	if len(ranked) != 3 {
		t.Fatalf("RankCandidates returned %d candidates, want 3", len(ranked))
	}
	if ranked[0].Port != 10809 {
		t.Fatalf("first candidate port = %d, want current v2rayN port 10809", ranked[0].Port)
	}
	if ranked[1].Port != 10808 {
		t.Fatalf("second candidate port = %d, want related port 10808", ranked[1].Port)
	}
}

func TestMarkV2RayRelatedProcessesIncludesDescendantsAndKnownCores(t *testing.T) {
	t.Parallel()

	processes := []Process{
		{PID: 100, ParentPID: 1, Name: "v2rayN.exe"},
		{PID: 101, ParentPID: 100, Name: "AmazTool.exe"},
		{PID: 102, ParentPID: 101, Name: "custom-core.exe"},
		{PID: 200, ParentPID: 1, Name: "xray.exe"},
		{PID: 300, ParentPID: 1, Name: "unrelated.exe"},
	}

	related := MarkV2RayRelatedProcesses(processes)

	for _, pid := range []uint32{100, 101, 102, 200} {
		if !related[pid] {
			t.Errorf("PID %d was not marked as v2rayN-related", pid)
		}
	}
	if related[300] {
		t.Error("unrelated PID 300 was marked as v2rayN-related")
	}
}

func TestBuildCandidatesKeepsV2RayCurrentAndCommonListeners(t *testing.T) {
	t.Parallel()

	processes := []Process{
		{PID: 100, ParentPID: 1, Name: "v2rayN.exe"},
		{PID: 101, ParentPID: 100, Name: "custom-core.exe"},
		{PID: 200, ParentPID: 1, Name: "browser.exe"},
		{PID: 300, ParentPID: 1, Name: "database.exe"},
	}
	listeners := []Listener{
		{Address: "127.0.0.1", Port: 25000, PID: 101},
		{Address: "0.0.0.0", Port: 10802, PID: 200},
		{Address: "127.0.0.1", Port: 7890, PID: 200},
		{Address: "127.0.0.1", Port: 5432, PID: 300},
	}

	candidates := BuildCandidates(processes, listeners, "127.0.0.1:10802")

	gotPorts := make(map[int]bool)
	for _, candidate := range candidates {
		gotPorts[candidate.Port] = true
		if candidate.Host != "127.0.0.1" {
			t.Errorf("candidate host = %q, want loopback dial address", candidate.Host)
		}
	}
	for _, port := range []int{25000, 10802, 7890} {
		if !gotPorts[port] {
			t.Errorf("port %d was not retained", port)
		}
	}
	if gotPorts[5432] {
		t.Error("unrelated database listener was retained")
	}
}
