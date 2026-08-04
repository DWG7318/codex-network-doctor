package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/dwg7318/codex-network-doctor/internal/proxyrepair"
	"github.com/dwg7318/codex-network-doctor/internal/speedtest"
)

type speedPlan struct {
	Endpoint  string
	TargetURL string
	Attempts  int
	Timeout   time.Duration
	Delay     time.Duration
}

type speedMeasureFunc func(context.Context, speedtest.Request) speedtest.Sample
type speedProbeFunc func(context.Context, proxyrepair.Candidate) error

func selectSpeedCandidate(
	ctx context.Context,
	candidates []proxyrepair.Candidate,
	probe speedProbeFunc,
) (proxyrepair.Candidate, error) {
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return proxyrepair.Candidate{}, err
		}
		if err := probe(ctx, candidate); err == nil {
			return candidate, nil
		}
	}
	return proxyrepair.Candidate{}, fmt.Errorf("no local candidate accepted HTTP CONNECT")
}

func runSpeedPlan(
	ctx context.Context,
	plan speedPlan,
	measure speedMeasureFunc,
	output io.Writer,
) speedtest.Report {
	proxyURL := &url.URL{Scheme: "http", Host: plan.Endpoint}
	httpTransport := http.DefaultTransport.(*http.Transport).Clone()
	httpTransport.Proxy = http.ProxyURL(proxyURL)
	socksTransport := http.DefaultTransport.(*http.Transport).Clone()
	socksTransport.Proxy = nil
	socksTransport.DialContext = speedtest.SOCKS5DialContext(plan.Endpoint, 8*time.Second)
	systemTransport := http.DefaultTransport.(*http.Transport).Clone()
	systemTransport.Proxy = nil

	paths := []struct {
		path      speedtest.Path
		transport *http.Transport
	}{
		{path: speedtest.PathSystem, transport: systemTransport},
		{path: speedtest.PathHTTPProxy, transport: httpTransport},
		{path: speedtest.PathSOCKS5Proxy, transport: socksTransport},
	}
	var samples []speedtest.Sample
	for attempt := 1; attempt <= plan.Attempts; attempt++ {
		for _, item := range paths {
			requestContext, cancel := context.WithTimeout(ctx, plan.Timeout)
			sample := measure(requestContext, speedtest.Request{
				Path: item.path, Attempt: attempt, URL: plan.TargetURL,
				Transport: item.transport, MaximumResponseBytes: 2 << 20,
			})
			cancel()
			samples = append(samples, sample)
			writeSpeedSample(output, sample)
			if plan.Delay > 0 {
				select {
				case <-ctx.Done():
					plan.Delay = 0
				case <-time.After(plan.Delay):
				}
			}
		}
	}
	report := speedtest.Analyze(samples)
	writeSpeedReport(output, report)
	return report
}

func writeSpeedSample(output io.Writer, sample speedtest.Sample) {
	if sample.Error != "" {
		_, _ = fmt.Fprintf(
			output,
			"[%s #%d] FAIL total=%s error=%s\n",
			sample.Path,
			sample.Attempt,
			sample.Total.Round(time.Millisecond),
			sample.Error,
		)
		return
	}
	_, _ = fmt.Fprintf(
		output,
		"[%s #%d] HTTP=%d bytes=%d total=%s TTFB=%s speed=%.2f Mbps\n",
		sample.Path,
		sample.Attempt,
		sample.StatusCode,
		sample.Bytes,
		sample.Total.Round(time.Millisecond),
		sample.TTFB.Round(time.Millisecond),
		sample.BytesPerSecond*8/1_000_000,
	)
}

func writeSpeedReport(output io.Writer, report speedtest.Report) {
	_, _ = fmt.Fprintln(output, "\n路径汇总：")
	for _, summary := range report.Summaries {
		_, _ = fmt.Fprintf(
			output,
			"- %s: success=%d fail=%d median=%.2f Mbps min=%.2f Mbps max=%.2f Mbps\n",
			summary.Path,
			summary.SuccessfulSamples,
			summary.FailedSamples,
			summary.MedianBytesPerSecond*8/1_000_000,
			summary.MinimumBytesPerSecond*8/1_000_000,
			summary.MaximumBytesPerSecond*8/1_000_000,
		)
	}
	_, _ = fmt.Fprintln(output, "\n分层判断：")
	for _, finding := range report.Findings {
		_, _ = fmt.Fprintf(
			output,
			"- [%s] %s\n  范围：%s\n  建议：%s\n",
			finding.Code,
			finding.Explanation,
			finding.Scope,
			finding.Action,
		)
	}
}
