// Package networkhealth implements explicit, bounded live observations. It never repairs hosts.
package networkhealth

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const MaxInputBytes = 1 << 20

var ErrConfig = errors.New("HEALTH_INPUT_INVALID")
var idPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

type Config struct {
	SchemaVersion string   `json:"schema_version"`
	Samples       int      `json:"samples"`
	TimeoutMS     int      `json:"timeout_ms"`
	Targets       []Target `json:"targets"`
	Routes        []Route  `json:"routes"`
}
type Target struct {
	ID             string `json:"id"`
	URL            string `json:"url"`
	ExpectedStatus int    `json:"expected_status"`
}
type Route struct {
	ID       string `json:"id"`
	ProxyURL string `json:"proxy_url,omitempty"`
}

func Decode(r io.Reader) (Config, error) {
	var c Config
	data, err := io.ReadAll(io.LimitReader(r, MaxInputBytes+1))
	if err != nil || len(data) > MaxInputBytes {
		return c, ErrConfig
	}
	d := json.NewDecoder(strings.NewReader(string(data)))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil {
		return Config{}, ErrConfig
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return Config{}, ErrConfig
	}
	return c, c.Validate()
}
func (c Config) Validate() error {
	if c.SchemaVersion != "1.0.0" || c.Samples < 3 || c.Samples > 5 || c.TimeoutMS < 100 || c.TimeoutMS > 10000 || len(c.Targets) < 1 || len(c.Targets) > 8 || len(c.Routes) < 1 || len(c.Routes) > 3 {
		return ErrConfig
	}
	if c.TimeoutMS*c.Samples*len(c.Targets)*len(c.Routes) > 120000 {
		return ErrConfig
	}
	ids := map[string]bool{}
	for _, t := range c.Targets {
		u, err := url.Parse(t.URL)
		if !idPattern.MatchString(t.ID) || ids[t.ID] || len(t.URL) > 2048 || err != nil || u == nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(t.URL, "#") || t.ExpectedStatus < 200 || t.ExpectedStatus > 299 {
			return ErrConfig
		}
		if p := u.Port(); p != "" {
			n, e := strconv.Atoi(p)
			if e != nil || n < 1 || n > 65535 {
				return ErrConfig
			}
		}
		ids[t.ID] = true
	}
	ids = map[string]bool{}
	for _, r := range c.Routes {
		if ids[r.ID] {
			return ErrConfig
		}
		ids[r.ID] = true
		if r.ID == "direct" {
			if r.ProxyURL != "" {
				return ErrConfig
			}
			continue
		}
		if r.ID != "lcnd-egress-cn" && r.ID != "lcnd-egress-global" {
			return ErrConfig
		}
		u, err := url.Parse(r.ProxyURL)
		if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "socks5" && u.Scheme != "socks5h") || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(r.ProxyURL, "#") || (u.Path != "" && u.Path != "/") {
			return ErrConfig
		}
		ip := net.ParseIP(u.Hostname())
		port, err := strconv.Atoi(u.Port())
		if ip == nil || (!ip.IsLoopback() && !ip.IsPrivate()) || err != nil || port < 1 || port > 65535 {
			return ErrConfig
		}
	}
	if !ids["direct"] {
		return ErrConfig
	}
	return nil
}

type Phase struct {
	Status     string `json:"status"`
	Code       string `json:"code,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}
type Sample struct {
	TargetID   string `json:"target_id"`
	RouteID    string `json:"route_id"`
	Attempt    int    `json:"attempt"`
	Complete   bool   `json:"complete"`
	DNS        Phase  `json:"dns"`
	TCP        Phase  `json:"tcp"`
	TLS        Phase  `json:"tls"`
	HTTP       Phase  `json:"http"`
	TCPScope   string `json:"tcp_scope"`
	HTTPStatus int    `json:"http_status,omitempty"`
	TotalMS    int64  `json:"total_ms"`
}
type Assessment struct {
	Status           string   `json:"status"`
	Score            *int     `json:"score"`
	ExpectedSamples  int      `json:"expected_samples"`
	CompletedSamples int      `json:"completed_samples"`
	PassedChecks     int      `json:"passed_checks"`
	RequiredChecks   int      `json:"required_checks"`
	Codes            []string `json:"codes"`
}
type Interface struct {
	Name      string   `json:"name"`
	Up        bool     `json:"up"`
	Loopback  bool     `json:"loopback"`
	MTU       int      `json:"mtu"`
	Addresses []string `json:"addresses"`
}
type Host struct {
	Platform    string      `json:"platform"`
	Supported   bool        `json:"supported"`
	Status      string      `json:"status"`
	Interfaces  []Interface `json:"interfaces"`
	Nameservers []string    `json:"nameservers"`
	IPv4Routes  []string    `json:"ipv4_routes"`
	IPv6Routes  []string    `json:"ipv6_routes"`
	Codes       []string    `json:"codes"`
}
type Report struct {
	SchemaVersion string       `json:"schema_version"`
	ToolVersion   string       `json:"tool_version"`
	EvidenceMode  string       `json:"evidence_mode"`
	StartedAt     string       `json:"started_at"`
	FinishedAt    string       `json:"finished_at"`
	Scope         string       `json:"scope"`
	Sensitive     bool         `json:"sensitive"`
	Host          Host         `json:"host"`
	Samples       []Sample     `json:"samples"`
	Assessment    Assessment   `json:"assessment"`
	Paths         []PathHealth `json:"paths"`
	Limitations   []string     `json:"limitations"`
}

type PathHealth struct {
	TargetID   string     `json:"target_id"`
	RouteID    string     `json:"route_id"`
	HTTPPassed int        `json:"http_passed"`
	MedianMS   *int64     `json:"median_success_ms"`
	Assessment Assessment `json:"assessment"`
}
