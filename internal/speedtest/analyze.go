package speedtest

import "sort"

const (
	minimumComparativeSamples = 3
	slowPathRatio             = 0.5
	unstableSpreadRatio       = 4.0
)

func Analyze(samples []Sample) Report {
	report := Report{Samples: append([]Sample(nil), samples...)}
	byPath := make(map[Path][]float64)
	failures := make(map[Path]int)
	for _, sample := range samples {
		if sample.Error != "" || sample.StatusCode < 200 || sample.StatusCode >= 400 || sample.BytesPerSecond <= 0 {
			failures[sample.Path]++
			continue
		}
		byPath[sample.Path] = append(byPath[sample.Path], sample.BytesPerSecond)
	}

	paths := []Path{PathSystem, PathHTTPProxy, PathSOCKS5Proxy}
	summaries := make(map[Path]Summary, len(paths))
	for _, path := range paths {
		values := append([]float64(nil), byPath[path]...)
		if len(values) == 0 && failures[path] == 0 {
			continue
		}
		sort.Float64s(values)
		summary := Summary{Path: path, SuccessfulSamples: len(values), FailedSamples: failures[path]}
		if len(values) > 0 {
			summary.MinimumBytesPerSecond = values[0]
			summary.MaximumBytesPerSecond = values[len(values)-1]
			summary.MedianBytesPerSecond = median(values)
		}
		summaries[path] = summary
		report.Summaries = append(report.Summaries, summary)
	}

	comparisons := 0
	if unreliableComparedWith(summaries, PathSystem, PathSOCKS5Proxy) {
		comparisons++
		report.Findings = append(report.Findings, Finding{
			Code: FindingTunPathUnreliable, Scope: "system/TUN ingress",
			Explanation: "三次配对中系统/TUN 路径至少失败两次，而同出口 SOCKS5 三次成功。",
			Action:      "检查 TUN 栈、MTU、路由/VPN 冲突；保持节点不变后复测。",
		})
	}
	if unreliableComparedWith(summaries, PathHTTPProxy, PathSOCKS5Proxy) {
		comparisons++
		report.Findings = append(report.Findings, Finding{
			Code: FindingHTTPProxyPathUnreliable, Scope: "local HTTP proxy ingress",
			Explanation: "三次配对中 HTTP 代理路径至少失败两次，而同端点 SOCKS5 三次成功。",
			Action:      "检查 mixed 入站的 HTTP CONNECT 路径；支持 SOCKS5 的应用可用 SOCKS5 对照。",
		})
	}
	if comparable(summaries, PathSystem, PathSOCKS5Proxy) {
		comparisons++
		system := summaries[PathSystem]
		socks := summaries[PathSOCKS5Proxy]
		if system.MedianBytesPerSecond < socks.MedianBytesPerSecond*slowPathRatio {
			report.Findings = append(report.Findings, Finding{
				Code: FindingTunPathSlow, Scope: "system/TUN ingress",
				Explanation: "系统路径吞吐持续低于同一出口的本地 SOCKS5 路径。",
				Action:      "检查 TUN 栈、MTU、路由冲突和其他 VPN；不要据此修改远端节点。",
			})
		}
		if unstable(system) && unstable(socks) {
			report.Findings = append(report.Findings, Finding{
				Code: FindingSharedPathUnstable, Scope: "shared remote/upstream path",
				Explanation: "系统/TUN 与 SOCKS5 路径同时出现四倍以上吞吐波动，瓶颈位于两条入口汇合之后。",
				Action:      "检查远端节点、上游代理、出口带宽和 TCP 重传；本地 TUN 调参不是首选。",
			})
		}
	}
	if comparable(summaries, PathHTTPProxy, PathSOCKS5Proxy) {
		comparisons++
		httpSummary := summaries[PathHTTPProxy]
		socks := summaries[PathSOCKS5Proxy]
		if httpSummary.MedianBytesPerSecond < socks.MedianBytesPerSecond*slowPathRatio {
			report.Findings = append(report.Findings, Finding{
				Code: FindingHTTPProxyPathSlow, Scope: "local HTTP proxy ingress",
				Explanation: "HTTP 代理吞吐持续低于同一端点的 SOCKS5 协议。",
				Action:      "支持 SOCKS5 的应用优先使用 SOCKS5；检查 mixed 入站的 HTTP CONNECT 路径。",
			})
		}
	}

	if comparisons == 0 {
		report.Findings = append(report.Findings, Finding{
			Code: FindingInsufficientEvidence, Scope: "measurement",
			Explanation: "每条对比路径至少需要三个成功样本。",
			Action:      "保持节点和 TUN 状态不变后重新运行测速。",
		})
	} else if len(report.Findings) == 0 {
		report.Findings = append(report.Findings, Finding{
			Code: FindingNoPathSpecificIssue, Scope: "measured ingress paths",
			Explanation: "当前样本未显示某一入口持续慢于对照路径。",
			Action:      "如仍有体感问题，保存结果并在故障时段复测。",
		})
	}
	return report
}

func comparable(summaries map[Path]Summary, left, right Path) bool {
	return summaries[left].SuccessfulSamples >= minimumComparativeSamples &&
		summaries[right].SuccessfulSamples >= minimumComparativeSamples
}

func unstable(summary Summary) bool {
	return summary.SuccessfulSamples >= minimumComparativeSamples &&
		summary.MinimumBytesPerSecond > 0 &&
		summary.MaximumBytesPerSecond/summary.MinimumBytesPerSecond >= unstableSpreadRatio
}

func unreliableComparedWith(summaries map[Path]Summary, path, reference Path) bool {
	summary := summaries[path]
	referenceSummary := summaries[reference]
	return summary.SuccessfulSamples+summary.FailedSamples >= minimumComparativeSamples &&
		summary.FailedSamples >= 2 &&
		referenceSummary.SuccessfulSamples >= minimumComparativeSamples
}

func median(sortedValues []float64) float64 {
	middle := len(sortedValues) / 2
	if len(sortedValues)%2 == 1 {
		return sortedValues[middle]
	}
	return (sortedValues[middle-1] + sortedValues[middle]) / 2
}
