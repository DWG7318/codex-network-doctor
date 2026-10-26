package networkhealth

import (
	"fmt"
	"io"
	"sort"
	"time"
)

func Assess(c Config, samples []Sample) Assessment {
	return assess(c, samples, c.Validate() == nil)
}

func assess(c Config, samples []Sample, valid bool) Assessment {
	a := Assessment{Status: "insufficient_evidence", ExpectedSamples: c.Samples * len(c.Targets) * len(c.Routes), Codes: []string{}}
	codes := map[string]bool{}
	seen := map[string]bool{}
	targets := map[string]bool{}
	routes := map[string]bool{}
	for _, t := range c.Targets {
		targets[t.ID] = true
	}
	for _, r := range c.Routes {
		routes[r.ID] = true
	}
	for _, s := range samples {
		key := fmt.Sprintf("%s/%s/%d", s.TargetID, s.RouteID, s.Attempt)
		if seen[key] || !targets[s.TargetID] || !routes[s.RouteID] || s.Attempt < 1 || s.Attempt > c.Samples {
			valid = false
		}
		seen[key] = true
		if s.Complete {
			a.CompletedSamples++
		}
		for _, phase := range []Phase{s.DNS, s.TCP, s.TLS, s.HTTP} {
			if phase.Status == "not_observed" || phase.Status == "not_applicable" {
				continue
			}
			a.RequiredChecks++
			if phase.Status == "pass" {
				a.PassedChecks++
			}
			if phase.Code != "" {
				codes[phase.Code] = true
			}
		}
	}
	if valid && a.CompletedSamples == a.ExpectedSamples && len(samples) == a.ExpectedSamples && a.RequiredChecks > 0 {
		score := 100 * a.PassedChecks / a.RequiredChecks
		a.Score = &score
		a.Status = "degraded"
		if score == 100 {
			a.Status = "healthy"
		}
		if a.PassedChecks == 0 {
			a.Status = "unhealthy"
		}
	} else {
		codes["INSUFFICIENT_EVIDENCE"] = true
	}
	for code := range codes {
		a.Codes = append(a.Codes, code)
	}
	sort.Strings(a.Codes)
	return a
}

func NewReport(version string, started time.Time, host Host, c Config, samples []Sample) Report {
	return Report{SchemaVersion: "1.0.0", ToolVersion: version, EvidenceMode: "live", StartedAt: started.UTC().Format(time.RFC3339Nano), FinishedAt: time.Now().UTC().Format(time.RFC3339Nano), Scope: "selected_targets_and_explicit_paths", Sensitive: true, Host: host, Samples: samples, Assessment: Assess(c, samples), Paths: summarizePaths(c, samples), Limitations: []string{
		"Score is the percentage of successful required DNS/TCP/TLS/HTTP checks; blocked downstream checks are not successes.",
		"Score is not a whole-network rating, bandwidth measurement or root-cause proof; HTTP timing is time to headers, not throughput.",
		"direct bypasses environment proxies only; existing host routes, VPN and TUN may still carry traffic.",
		"Proxy target DNS is not observed; TCP success for a proxy route only confirms its proxy endpoint.",
		"Host observations are a bounded snapshot; policy routes, firewall and all intermediate hops are not exhaustively evaluated.",
	}}
}

func summarizePaths(c Config, samples []Sample) []PathHealth {
	out := []PathHealth{}
	for _, target := range c.Targets {
		for _, route := range c.Routes {
			selected := []Sample{}
			times := []int64{}
			passed := 0
			for _, s := range samples {
				if s.TargetID == target.ID && s.RouteID == route.ID {
					selected = append(selected, s)
					if s.Complete && s.HTTP.Status == "pass" {
						passed++
						times = append(times, s.TotalMS)
					}
				}
			}
			sub := c
			sub.Targets = []Target{target}
			sub.Routes = []Route{route}
			item := PathHealth{TargetID: target.ID, RouteID: route.ID, HTTPPassed: passed, Assessment: assess(sub, selected, c.Validate() == nil)}
			if len(times) > 0 {
				sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
				v := times[len(times)/2]
				if len(times)%2 == 0 {
					v = (v + times[len(times)/2-1]) / 2
				}
				item.MedianMS = &v
			}
			out = append(out, item)
		}
	}
	return out
}

func WriteText(w io.Writer, r Report) error {
	score := "无（证据不足）"
	if r.Assessment.Score != nil {
		score = fmt.Sprintf("%d/100", *r.Assessment.Score)
	}
	_, err := fmt.Fprintf(w, "LCnd Ubuntu 网络健康报告\n版本：%s\n证据：%s\n平台：%s\n主机观察：%s\n检测范围：指定目标与显式路径\n状态：%s\n评分：%s\n完整样本：%d/%d\n成功检查：%d/%d\n", r.ToolVersion, r.EvidenceMode, r.Host.Platform, r.Host.Status, r.Assessment.Status, score, r.Assessment.CompletedSamples, r.Assessment.ExpectedSamples, r.Assessment.PassedChecks, r.Assessment.RequiredChecks)
	if err != nil {
		return err
	}
	for _, p := range r.Paths {
		score := "无"
		if p.Assessment.Score != nil {
			score = fmt.Sprint(*p.Assessment.Score)
		}
		median := "无成功样本"
		if p.MedianMS != nil {
			median = fmt.Sprintf("%dms", *p.MedianMS)
		}
		if _, err = fmt.Fprintf(w, "路径汇总：目标=%s 路径=%s 状态=%s 评分=%s HTTP成功=%d 成功样本耗时中位数=%s\n", p.TargetID, p.RouteID, p.Assessment.Status, score, p.HTTPPassed, median); err != nil {
			return err
		}
	}
	for _, s := range r.Samples {
		if _, err = fmt.Fprintf(w, "目标=%s 路径=%s 样本=%d DNS=%s(%s) TCP=%s(%s) TLS=%s(%s) HTTP=%s(%s) 状态码=%d 总耗时=%dms\n", s.TargetID, s.RouteID, s.Attempt, s.DNS.Status, s.DNS.Code, s.TCP.Status, s.TCP.Code, s.TLS.Status, s.TLS.Code, s.HTTP.Status, s.HTTP.Code, s.HTTPStatus, s.TotalMS); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(w, "评分只表示本次目标/路径的分层检查成功率。不能据此认定整机网络完全健康或断言故障根因。\n未做带宽测速；未修改网络配置或服务。代理目标 DNS 未观测，直连仍可能经过宿主已有 VPN/TUN。\nJSON 包含敏感本机地址/路由快照，请私下保存。")
	return err
}
