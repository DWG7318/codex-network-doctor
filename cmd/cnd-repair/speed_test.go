package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dwg7318/codex-network-doctor/internal/proxyrepair"
	"github.com/dwg7318/codex-network-doctor/internal/speedtest"
)

func TestRunSpeedPlanPairsIngressPathsAndWritesFindingCodes(t *testing.T) {
	t.Parallel()

	var paths []speedtest.Path
	measure := func(_ context.Context, request speedtest.Request) speedtest.Sample {
		paths = append(paths, request.Path)
		bytesPerSecond := 100_000.0
		if request.Path == speedtest.PathSOCKS5Proxy {
			bytesPerSecond = 1_000_000
		}
		return speedtest.Sample{
			Path: request.Path, Attempt: request.Attempt,
			StatusCode: 200, Bytes: 500_000,
			BytesPerSecond: bytesPerSecond, Total: time.Second,
		}
	}
	var output bytes.Buffer
	report := runSpeedPlan(
		context.Background(),
		speedPlan{
			Endpoint:  "127.0.0.1:20218",
			TargetURL: "https://speed.example/download?bytes=500000",
			Attempts:  3,
			Timeout:   time.Second,
		},
		measure,
		&output,
	)

	wantPaths := []speedtest.Path{
		speedtest.PathSystem, speedtest.PathHTTPProxy, speedtest.PathSOCKS5Proxy,
		speedtest.PathSystem, speedtest.PathHTTPProxy, speedtest.PathSOCKS5Proxy,
		speedtest.PathSystem, speedtest.PathHTTPProxy, speedtest.PathSOCKS5Proxy,
	}
	if len(paths) != len(wantPaths) {
		t.Fatalf("paths = %v, want %v", paths, wantPaths)
	}
	for index := range wantPaths {
		if paths[index] != wantPaths[index] {
			t.Fatalf("paths[%d] = %q, want %q", index, paths[index], wantPaths[index])
		}
	}
	if !hasSpeedFinding(report, speedtest.FindingHTTPProxyPathSlow) {
		t.Fatalf("findings = %#v", report.Findings)
	}
	for _, required := range []string{"system_path", "http_proxy", "socks5_proxy", "HTTP_PROXY_PATH_SLOW"} {
		if !strings.Contains(output.String(), required) {
			t.Errorf("output does not contain %q:\n%s", required, output.String())
		}
	}
}

func TestSelectSpeedCandidateSkipsNonProxyControlListener(t *testing.T) {
	t.Parallel()

	candidates := []proxyrepair.Candidate{
		{Host: "127.0.0.1", Port: 20224, ProcessName: "sing-box.exe", V2RayRelated: true},
		{Host: "127.0.0.1", Port: 20218, ProcessName: "sing-box.exe", V2RayRelated: true},
	}
	selected, err := selectSpeedCandidate(
		context.Background(),
		candidates,
		func(_ context.Context, candidate proxyrepair.Candidate) error {
			if candidate.Port == 20218 {
				return nil
			}
			return errors.New("CONNECT rejected")
		},
	)
	if err != nil {
		t.Fatalf("selectSpeedCandidate returned error: %v", err)
	}
	if selected.Port != 20218 {
		t.Fatalf("selected port = %d, want 20218", selected.Port)
	}
}

func hasSpeedFinding(report speedtest.Report, code speedtest.FindingCode) bool {
	for _, finding := range report.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
