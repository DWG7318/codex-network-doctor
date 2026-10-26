package networkhealth

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"sync"
	"time"
)

// TLSConfig is only an embedding/test trust store. The CLI exposes no TLS bypass.
type Prober struct {
	TLSConfig *tls.Config
	resolver  hostResolver
	dial      func(context.Context, string, string) (net.Conn, error)
}

type hostResolver interface {
	LookupHost(context.Context, string) ([]string, error)
}

func (p *Prober) Run(ctx context.Context, c Config) []Sample {
	results := []Sample{}
	if c.Validate() != nil {
		return results
	}
	for attempt := 1; attempt <= c.Samples; attempt++ {
		for _, target := range c.Targets {
			for _, route := range c.Routes {
				if ctx.Err() != nil {
					return results
				}
				results = append(results, p.sample(ctx, c, target, route, attempt))
			}
		}
	}
	return results
}

func (p *Prober) sample(parent context.Context, c Config, target Target, route Route, attempt int) Sample {
	s := Sample{TargetID: target.ID, RouteID: route.ID, Attempt: attempt, TCPScope: "target", DNS: Phase{Status: "blocked"}, TCP: Phase{Status: "blocked"}, TLS: Phase{Status: "blocked"}, HTTP: Phase{Status: "blocked"}}
	started := time.Now()
	ctx, cancel := context.WithTimeout(parent, time.Duration(c.TimeoutMS)*time.Millisecond)
	defer cancel()
	finish := func() Sample {
		s.TotalMS = time.Since(started).Milliseconds()
		s.Complete = parent.Err() == nil
		return s
	}
	u, _ := url.Parse(target.URL)
	transport := &http.Transport{Proxy: nil, DisableKeepAlives: true, ForceAttemptHTTP2: false, MaxResponseHeaderBytes: 32 << 10, TLSHandshakeTimeout: time.Duration(c.TimeoutMS) * time.Millisecond}
	if p.TLSConfig != nil {
		transport.TLSClientConfig = p.TLSConfig.Clone()
	}
	defer transport.CloseIdleConnections()
	var mu sync.Mutex
	if route.ID == "direct" {
		dnsStarted := time.Now()
		addresses := []string{u.Hostname()}
		if net.ParseIP(u.Hostname()) != nil {
			s.DNS.Status = "not_applicable"
		} else {
			var err error
			resolver := p.resolver
			if resolver == nil {
				resolver = net.DefaultResolver
			}
			addresses, err = resolver.LookupHost(ctx, u.Hostname())
			s.DNS.DurationMS = time.Since(dnsStarted).Milliseconds()
			if err != nil || len(addresses) == 0 {
				s.DNS.Status = "fail"
				s.DNS.Code = errorCode(ctx, "DNS_FAILED")
				return finish()
			}
			s.DNS.Status = "pass"
		}
		// Pin this attempt to its observed resolver answers, with IPv4/IPv6 fallback.
		// Transport detaches the request deadline from its dialing context. Keep
		// the original sample context for every address attempt and cancellation.
		transport.DialContext = func(_ context.Context, network, address string) (net.Conn, error) {
			_, port, _ := net.SplitHostPort(address)
			var last error
			tcpStarted := time.Now()
			for i, ip := range addresses {
				if i >= 16 {
					break
				}
				deadline, _ := ctx.Deadline()
				budget := time.Until(deadline)
				remaining := min(len(addresses)-i, 16-i)
				dialCtx, cancel := context.WithTimeout(ctx, budget/time.Duration(remaining))
				dial := p.dial
				if dial == nil {
					dial = (&net.Dialer{}).DialContext
				}
				conn, err := dial(dialCtx, "tcp", net.JoinHostPort(ip, port))
				cancel()
				if err == nil {
					mu.Lock()
					s.TCP = Phase{Status: "pass", DurationMS: time.Since(tcpStarted).Milliseconds()}
					mu.Unlock()
					return conn, nil
				}
				last = err
			}
			mu.Lock()
			s.TCP = Phase{Status: "fail", Code: errorCode(ctx, "TCP_FAILED"), DurationMS: time.Since(tcpStarted).Milliseconds()}
			mu.Unlock()
			return nil, last
		}
	} else {
		s.DNS = Phase{Status: "not_observed", Code: "PROXY_RESOLVES_TARGET"}
		s.TCPScope = "proxy_endpoint"
		proxy, _ := url.Parse(route.ProxyURL)
		transport.Proxy = http.ProxyURL(proxy)
		transport.DialContext = (&net.Dialer{Timeout: time.Duration(c.TimeoutMS) * time.Millisecond}).DialContext
	}
	var tcpStarted, tlsStarted time.Time
	trace := &httptrace.ClientTrace{
		ConnectStart: func(_, _ string) { mu.Lock(); defer mu.Unlock(); tcpStarted = time.Now() },
		ConnectDone: func(_, _ string, err error) {
			if route.ID == "direct" {
				return
			}
			mu.Lock()
			defer mu.Unlock()
			s.TCP = Phase{Status: "pass", DurationMS: time.Since(tcpStarted).Milliseconds()}
			if err != nil {
				s.TCP.Status = "fail"
				s.TCP.Code = errorCode(ctx, "PROXY_TCP_FAILED")
			}
		},
		TLSHandshakeStart: func() { mu.Lock(); defer mu.Unlock(); tlsStarted = time.Now() },
		TLSHandshakeDone: func(_ tls.ConnectionState, err error) {
			mu.Lock()
			defer mu.Unlock()
			s.TLS = Phase{Status: "pass", DurationMS: time.Since(tlsStarted).Milliseconds()}
			if err != nil {
				s.TLS.Status = "fail"
				s.TLS.Code = errorCode(ctx, "TLS_FAILED")
			}
		},
	}
	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(ctx, trace), http.MethodHead, target.URL, nil)
	if err != nil {
		s.HTTP = Phase{Status: "fail", Code: "REQUEST_INVALID"}
		return finish()
	}
	req.Header.Set("User-Agent", "LCnd-Network-Health/1")
	client := &http.Client{Transport: transport, Timeout: time.Duration(c.TimeoutMS) * time.Millisecond, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	mu.Lock()
	defer mu.Unlock()
	if err != nil {
		switch {
		case s.TCP.Status == "blocked":
			s.TCP = Phase{Status: "fail", Code: errorCode(ctx, "TCP_FAILED")}
		case s.TCP.Status == "pass" && s.TLS.Status == "blocked":
			s.TLS = Phase{Status: "fail", Code: errorCode(ctx, "PROXY_TUNNEL_FAILED")}
		case s.TLS.Status == "pass":
			s.HTTP = Phase{Status: "fail", Code: errorCode(ctx, "HTTP_FAILED"), DurationMS: time.Since(started).Milliseconds()}
		}
		return finish()
	}
	_ = response.Body.Close() // HEAD only: no payload download, cookies, headers or body persisted.
	s.HTTPStatus = response.StatusCode
	s.HTTP = Phase{Status: "pass", DurationMS: time.Since(started).Milliseconds()}
	if response.StatusCode < 100 || response.StatusCode > 599 {
		s.HTTPStatus = 0
		s.HTTP.Status = "fail"
		s.HTTP.Code = "HTTP_INVALID_STATUS"
	} else if response.StatusCode != target.ExpectedStatus {
		s.HTTP.Status = "fail"
		s.HTTP.Code = "HTTP_UNEXPECTED_STATUS"
	}
	return finish()
}
func errorCode(ctx context.Context, fallback string) string {
	if ctx.Err() == context.DeadlineExceeded {
		return "TIMEOUT"
	}
	if ctx.Err() != nil {
		return "CANCELLED"
	}
	return fallback
}
