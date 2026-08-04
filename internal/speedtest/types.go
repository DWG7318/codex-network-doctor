package speedtest

import "time"

type Path string

const (
	PathSystem      Path = "system_path"
	PathHTTPProxy   Path = "http_proxy"
	PathSOCKS5Proxy Path = "socks5_proxy"
)

type Sample struct {
	Path           Path          `json:"path"`
	Attempt        int           `json:"attempt"`
	StatusCode     int           `json:"status_code,omitempty"`
	Bytes          int64         `json:"bytes"`
	BytesPerSecond float64       `json:"bytes_per_second"`
	DNS            time.Duration `json:"dns"`
	Connect        time.Duration `json:"connect"`
	TLS            time.Duration `json:"tls"`
	TTFB           time.Duration `json:"ttfb"`
	Total          time.Duration `json:"total"`
	Error          string        `json:"error,omitempty"`
}

type Summary struct {
	Path                  Path    `json:"path"`
	SuccessfulSamples     int     `json:"successful_samples"`
	FailedSamples         int     `json:"failed_samples"`
	MedianBytesPerSecond  float64 `json:"median_bytes_per_second"`
	MinimumBytesPerSecond float64 `json:"minimum_bytes_per_second"`
	MaximumBytesPerSecond float64 `json:"maximum_bytes_per_second"`
}

type FindingCode string

const (
	FindingTunPathSlow             FindingCode = "TUN_PATH_SLOW"
	FindingTunPathUnreliable       FindingCode = "TUN_PATH_UNRELIABLE"
	FindingHTTPProxyPathSlow       FindingCode = "HTTP_PROXY_PATH_SLOW"
	FindingHTTPProxyPathUnreliable FindingCode = "HTTP_PROXY_PATH_UNRELIABLE"
	FindingSharedPathUnstable      FindingCode = "SHARED_REMOTE_PATH_UNSTABLE"
	FindingInsufficientEvidence    FindingCode = "INSUFFICIENT_EVIDENCE"
	FindingNoPathSpecificIssue     FindingCode = "NO_PATH_SPECIFIC_ISSUE"
)

type Finding struct {
	Code        FindingCode `json:"code"`
	Scope       string      `json:"scope"`
	Explanation string      `json:"explanation"`
	Action      string      `json:"action"`
}

type Report struct {
	Samples   []Sample  `json:"samples"`
	Summaries []Summary `json:"summaries"`
	Findings  []Finding `json:"findings"`
}
