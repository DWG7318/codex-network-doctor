package speedtest

import "testing"

func TestAnalyzeIdentifiesTunIngressSlowdown(t *testing.T) {
	t.Parallel()

	report := Analyze([]Sample{
		success(PathSystem, 1, 80_000),
		success(PathSystem, 2, 90_000),
		success(PathSystem, 3, 100_000),
		success(PathSOCKS5Proxy, 1, 800_000),
		success(PathSOCKS5Proxy, 2, 900_000),
		success(PathSOCKS5Proxy, 3, 1_000_000),
	})

	if !hasFinding(report, FindingTunPathSlow) {
		t.Fatalf("findings = %#v, want %q", report.Findings, FindingTunPathSlow)
	}
	if hasFinding(report, FindingSharedPathUnstable) {
		t.Fatalf("stable samples were classified as shared instability: %#v", report.Findings)
	}
}

func TestAnalyzeIdentifiesHTTPIngressSlowdownAcrossMixedPorts(t *testing.T) {
	t.Parallel()

	report := Analyze([]Sample{
		success(PathHTTPProxy, 1, 70_000),
		success(PathHTTPProxy, 2, 80_000),
		success(PathHTTPProxy, 3, 90_000),
		success(PathSOCKS5Proxy, 1, 700_000),
		success(PathSOCKS5Proxy, 2, 800_000),
		success(PathSOCKS5Proxy, 3, 900_000),
	})

	if !hasFinding(report, FindingHTTPProxyPathSlow) {
		t.Fatalf("findings = %#v, want %q", report.Findings, FindingHTTPProxyPathSlow)
	}
}

func TestAnalyzeIdentifiesSharedInstabilityAfterIngressPathsConverge(t *testing.T) {
	t.Parallel()

	report := Analyze([]Sample{
		success(PathSystem, 1, 50_000),
		success(PathSystem, 2, 500_000),
		success(PathSystem, 3, 60_000),
		success(PathSOCKS5Proxy, 1, 55_000),
		success(PathSOCKS5Proxy, 2, 550_000),
		success(PathSOCKS5Proxy, 3, 65_000),
	})

	if !hasFinding(report, FindingSharedPathUnstable) {
		t.Fatalf("findings = %#v, want %q", report.Findings, FindingSharedPathUnstable)
	}
	if hasFinding(report, FindingTunPathSlow) {
		t.Fatalf("similar unstable paths were misclassified as TUN-only: %#v", report.Findings)
	}
}

func TestAnalyzeRequiresThreeSuccessfulSamplesForComparativeFinding(t *testing.T) {
	t.Parallel()

	report := Analyze([]Sample{
		success(PathSystem, 1, 10_000),
		success(PathSOCKS5Proxy, 1, 1_000_000),
	})

	if !hasFinding(report, FindingInsufficientEvidence) {
		t.Fatalf("findings = %#v, want %q", report.Findings, FindingInsufficientEvidence)
	}
	if hasFinding(report, FindingTunPathSlow) {
		t.Fatalf("single samples must not produce TUN finding: %#v", report.Findings)
	}
}

func TestAnalyzeIdentifiesTunPathWithTwoOfThreeFailures(t *testing.T) {
	t.Parallel()

	report := Analyze([]Sample{
		success(PathSystem, 1, 100_000),
		failure(PathSystem, 2),
		failure(PathSystem, 3),
		success(PathSOCKS5Proxy, 1, 600_000),
		success(PathSOCKS5Proxy, 2, 700_000),
		success(PathSOCKS5Proxy, 3, 800_000),
	})

	if !hasFinding(report, FindingTunPathUnreliable) {
		t.Fatalf("findings = %#v, want %q", report.Findings, FindingTunPathUnreliable)
	}
	if hasFinding(report, FindingInsufficientEvidence) {
		t.Fatalf("three paired attempts are not insufficient: %#v", report.Findings)
	}
}

func TestAnalyzeIdentifiesHTTPPathWithTwoOfThreeFailures(t *testing.T) {
	t.Parallel()

	report := Analyze([]Sample{
		failure(PathHTTPProxy, 1),
		success(PathHTTPProxy, 2, 100_000),
		failure(PathHTTPProxy, 3),
		success(PathSOCKS5Proxy, 1, 600_000),
		success(PathSOCKS5Proxy, 2, 700_000),
		success(PathSOCKS5Proxy, 3, 800_000),
	})

	if !hasFinding(report, FindingHTTPProxyPathUnreliable) {
		t.Fatalf("findings = %#v, want %q", report.Findings, FindingHTTPProxyPathUnreliable)
	}
}

func success(path Path, attempt int, bytesPerSecond float64) Sample {
	return Sample{
		Path:           path,
		Attempt:        attempt,
		Bytes:          500_000,
		BytesPerSecond: bytesPerSecond,
		StatusCode:     200,
	}
}

func failure(path Path, attempt int) Sample {
	return Sample{Path: path, Attempt: attempt, Error: "timeout"}
}

func hasFinding(report Report, code FindingCode) bool {
	for _, finding := range report.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
