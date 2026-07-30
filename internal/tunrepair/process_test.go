package tunrepair

import (
	"testing"
	"time"
)

func TestIsV2RayNGUINameAcceptsRenamedNumericExecutable(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"v2rayN.exe", "v2rayN2.exe", "V2RAYN-3.EXE"} {
		if !IsV2RayNGUIName(name) {
			t.Fatalf("expected %q to be recognized as a v2rayN GUI executable", name)
		}
	}
	for _, name := range []string{"v2ray.exe", "v2rayN-helper.exe", "not-v2rayN.exe"} {
		if IsV2RayNGUIName(name) {
			t.Fatalf("did not expect %q to be recognized as a v2rayN GUI executable", name)
		}
	}
}

func TestSelectKeeperUsesNewestV2RayNGUIInstance(t *testing.T) {
	t.Parallel()

	oldStart := time.Date(2026, 7, 30, 8, 0, 0, 0, time.Local)
	newStart := oldStart.Add(10 * time.Minute)
	processes := []Process{
		{PID: 10, Name: "v2rayN.exe", Path: `C:\v2rayN\v2rayN.exe`, StartedAt: oldStart},
		{PID: 20, Name: "v2rayN2.exe", Path: `C:\v2rayN\v2rayN2.exe`, StartedAt: newStart},
		{PID: 30, ParentPID: 20, Name: "sing-box.exe", Path: `C:\v2rayN\bin\sing_box\sing-box.exe`},
	}

	keeper, duplicates, err := SelectKeeper(processes)
	if err != nil {
		t.Fatalf("SelectKeeper() error = %v", err)
	}
	if keeper.PID != 20 {
		t.Fatalf("keeper PID = %d, want 20", keeper.PID)
	}
	if len(duplicates) != 1 || duplicates[0].PID != 10 {
		t.Fatalf("duplicates = %#v, want only PID 10", duplicates)
	}
}

func TestPlanTerminationNeverIncludesUnknownPortOwner(t *testing.T) {
	t.Parallel()

	processes := []Process{
		{PID: 20, Name: "v2rayN2.exe", Path: `C:\v2rayN\v2rayN2.exe`},
		{PID: 30, ParentPID: 20, Name: "sing-box.exe", Path: `C:\v2rayN\bin\sing_box\sing-box.exe`},
		{PID: 40, ParentPID: 20, Name: "database.exe", Path: `C:\Database\database.exe`},
	}
	keeper := processes[0]
	listeners := []Listener{
		{Address: "127.0.0.1", Port: 10807, PID: 40},
		{Address: "127.0.0.1", Port: 10808, PID: 30},
	}

	plan := PlanTermination(processes, listeners, keeper, []int{10807, 10808})
	if containsPID(plan.Terminate, 40) {
		t.Fatalf("termination plan must not include unknown port owner: %#v", plan.Terminate)
	}
	if !containsPID(plan.UnknownOwners, 40) {
		t.Fatalf("unknown owner list = %#v, want PID 40", plan.UnknownOwners)
	}
	if !containsPID(plan.Terminate, 20) || !containsPID(plan.Terminate, 30) {
		t.Fatalf("termination plan = %#v, want keeper and its related core", plan.Terminate)
	}
}

func TestPlanTerminationIncludesOrphanCoreOnlyInsideV2RayNInstallation(t *testing.T) {
	t.Parallel()

	processes := []Process{
		{PID: 20, Name: "v2rayN2.exe", Path: `C:\v2rayN\v2rayN2.exe`},
		{PID: 30, Name: "sing-box.exe", Path: `C:\v2rayN\bin\sing_box\sing-box.exe`},
		{PID: 31, Name: "sing-box.exe", Path: `C:\Other\sing-box.exe`},
	}

	plan := PlanTermination(processes, nil, processes[0], nil)
	if !containsPID(plan.Terminate, 30) {
		t.Fatalf("expected same-installation orphan core to be terminated: %#v", plan.Terminate)
	}
	if containsPID(plan.Terminate, 31) {
		t.Fatalf("must not terminate a same-named core outside the installation: %#v", plan.Terminate)
	}
}

func containsPID(processes []Process, pid uint32) bool {
	for _, process := range processes {
		if process.PID == pid {
			return true
		}
	}
	return false
}
