package networkhealth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testConfig(url string) Config {
	return Config{SchemaVersion: "1.0.0", Samples: 3, TimeoutMS: 1000, Targets: []Target{{ID: "test", URL: url, ExpectedStatus: 204}}, Routes: []Route{{ID: "direct"}}}
}

func TestDecodeRejectsUnsafeInput(t *testing.T) {
	for _, url := range []string{"http://example.invalid/", "https://user:password@example.invalid/", "https://example.invalid/?token=secret", "https://example.invalid/#secret"} {
		c := testConfig(url)
		b, _ := json.Marshal(c)
		if _, err := Decode(strings.NewReader(string(b))); err == nil {
			t.Errorf("accepted unsafe URL")
		}
	}
	c := testConfig("https://example.invalid/")
	b, _ := json.Marshal(c)
	for _, b := range []string{string(b) + "{}", strings.Replace(string(b), `"samples":3`, `"samples":3,"token":"private"`, 1), strings.Repeat(" ", MaxInputBytes+1)} {
		if _, err := Decode(strings.NewReader(b)); err == nil {
			t.Error("accepted malformed input")
		}
	}
	c.Routes = append(c.Routes, Route{ID: "lcnd-egress-cn", ProxyURL: "http://user:secret@127.0.0.1:1080"})
	if err := c.Validate(); err == nil {
		t.Fatal("accepted proxy credentials")
	}
}

func TestRealTLSHeadAndScore(t *testing.T) {
	calls := 0
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "HEAD" {
			t.Error("not HEAD")
		}
		w.WriteHeader(204)
	}))
	defer srv.Close()
	roots := x509.NewCertPool()
	roots.AddCert(srv.Certificate())
	p := Prober{TLSConfig: &tls.Config{RootCAs: roots}}
	c := testConfig(srv.URL)
	results := p.Run(context.Background(), c)
	if calls != 3 || len(results) != 3 {
		t.Fatalf("calls=%d results=%d", calls, len(results))
	}
	a := Assess(c, results)
	if a.Status != "healthy" || a.Score == nil || *a.Score != 100 {
		t.Fatalf("assessment=%+v results=%+v", a, results)
	}
	if results[0].DNS.Status != "not_applicable" || results[0].TLS.Status != "pass" {
		t.Fatalf("phases=%+v", results[0])
	}
}

func TestTLSFailureIsEvidenceNotHealthy(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("untrusted TLS reached HTTP") }))
	defer srv.Close()
	c := testConfig(srv.URL)
	results := (&Prober{}).Run(context.Background(), c)
	a := Assess(c, results)
	if a.Status == "healthy" || a.Score == nil || results[0].TLS.Code != "TLS_FAILED" {
		t.Fatalf("assessment=%+v results=%+v", a, results)
	}
}

func TestRedirectNotFollowedAndNoEnvironmentProxy(t *testing.T) {
	calls := 0
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Location", "https://example.invalid/private")
		w.WriteHeader(302)
	}))
	defer srv.Close()
	roots := x509.NewCertPool()
	roots.AddCert(srv.Certificate())
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("ALL_PROXY", "socks5://127.0.0.1:1")
	c := testConfig(srv.URL)
	p := Prober{TLSConfig: &tls.Config{RootCAs: roots}}
	r := p.Run(context.Background(), c)
	if calls != 3 || r[0].HTTP.Code != "HTTP_UNEXPECTED_STATUS" || r[0].HTTPStatus != 302 {
		t.Fatalf("calls=%d results=%+v", calls, r)
	}
}

func TestTimeoutAndCancelledEvidence(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer srv.Close()
	roots := x509.NewCertPool()
	roots.AddCert(srv.Certificate())
	c := testConfig(srv.URL)
	c.TimeoutMS = 100
	p := Prober{TLSConfig: &tls.Config{RootCAs: roots}}
	started := time.Now()
	r := p.Run(context.Background(), c)
	if time.Since(started) > 2*time.Second || r[0].HTTP.Code != "TIMEOUT" {
		t.Fatalf("results=%+v", r)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a := Assess(c, p.Run(ctx, c))
	if a.Status != "insufficient_evidence" || a.Score != nil {
		t.Fatalf("assessment=%+v", a)
	}
}

func TestAssessmentRequiresEveryPairAndUniqueSamples(t *testing.T) {
	c := testConfig("https://example.invalid/")
	r := Sample{TargetID: "test", RouteID: "direct", Attempt: 1, Complete: true, DNS: Phase{Status: "pass"}, TCP: Phase{Status: "pass"}, TLS: Phase{Status: "pass"}, HTTP: Phase{Status: "pass"}}
	if a := Assess(c, []Sample{r, r, r}); a.Score != nil {
		t.Fatal("duplicate attempts invented evidence")
	}
}

type failingResolver struct{}

func (failingResolver) LookupHost(context.Context, string) ([]string, error) {
	return nil, errors.New("DO_NOT_ECHO_PRIVATE_ERROR")
}

func TestDNSFailureIsBoundedAndRedacted(t *testing.T) {
	c := testConfig("https://example.invalid/")
	results := (&Prober{resolver: failingResolver{}}).Run(context.Background(), c)
	a := Assess(c, results)
	if a.Score == nil || *a.Score != 0 || a.Status != "unhealthy" || results[0].DNS.Code != "DNS_FAILED" || results[0].TCP.Status != "blocked" {
		t.Fatalf("assessment=%+v results=%+v", a, results)
	}
	b, _ := json.Marshal(results)
	if strings.Contains(string(b), "DO_NOT_ECHO") {
		t.Fatal("raw resolver error leaked")
	}
}

func TestConfigHasHardBudgetAndDuplicateIDProtection(t *testing.T) {
	c := testConfig("https://example.invalid/")
	c.Targets = append(c.Targets, c.Targets[0])
	if c.Validate() == nil {
		t.Fatal("duplicate target accepted")
	}
	c = testConfig("https://example.invalid/")
	c.Samples = 5
	c.TimeoutMS = 10000
	c.Routes = append(c.Routes, Route{ID: "lcnd-egress-cn", ProxyURL: "http://127.0.0.1:10080"}, Route{ID: "lcnd-egress-global", ProxyURL: "http://127.0.0.1:10081"})
	if c.Validate() == nil {
		t.Fatal("unbounded total budget accepted")
	}
}

type fixedResolver []string

func (r fixedResolver) LookupHost(context.Context, string) ([]string, error) { return r, nil }

func TestAddressFallbackUsesRemainingSampleDeadline(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(204) }))
	defer srv.Close()
	roots := x509.NewCertPool()
	roots.AddCert(srv.Certificate())
	u, _ := url.Parse(srv.URL)
	u.Host = net.JoinHostPort(srv.Certificate().DNSNames[0], u.Port())
	c := testConfig(u.String())
	p := Prober{TLSConfig: &tls.Config{RootCAs: roots}, resolver: fixedResolver{"127.0.0.2", "127.0.0.3", "127.0.0.4", "127.0.0.1"}, dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, _, _ := net.SplitHostPort(addr)
		if host != "127.0.0.1" {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}}
	r := p.Run(context.Background(), c)
	if a := Assess(c, r); a.Status != "healthy" || calls.Load() != 3 {
		t.Fatalf("reachable fourth address not reached: calls=%d assessment=%+v", calls.Load(), a)
	}
}

func TestInvalidHTTPStatusDoesNotBreakReportContract(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(600) }))
	defer srv.Close()
	roots := x509.NewCertPool()
	roots.AddCert(srv.Certificate())
	r := (&Prober{TLSConfig: &tls.Config{RootCAs: roots}}).Run(context.Background(), testConfig(srv.URL))
	for _, s := range r {
		if s.HTTPStatus != 0 || s.HTTP.Status != "fail" || s.HTTP.Code != "HTTP_INVALID_STATUS" {
			t.Fatalf("invalid status escaped: %+v", s)
		}
	}
}

func TestCancellationDuringActiveRequestIsIncomplete(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { cancel(); <-r.Context().Done() }))
	defer srv.Close()
	roots := x509.NewCertPool()
	roots.AddCert(srv.Certificate())
	c := testConfig(srv.URL)
	r := (&Prober{TLSConfig: &tls.Config{RootCAs: roots}}).Run(ctx, c)
	if len(r) != 1 || r[0].Complete || r[0].HTTP.Code != "CANCELLED" {
		t.Fatalf("cancelled evidence=%+v", r)
	}
	if a := Assess(c, r); a.Score != nil || a.Status != "insufficient_evidence" {
		t.Fatalf("cancelled assessment=%+v", a)
	}
}
