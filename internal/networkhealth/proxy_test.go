package networkhealth

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestExistingHTTPAndSOCKSProxy(t *testing.T) {
	for _, kind := range []string{"http", "socks5h"} {
		t.Run(kind, func(t *testing.T) {
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
			defer srv.Close()
			roots := x509.NewCertPool()
			roots.AddCert(srv.Certificate())
			proxy, stop := testProxy(t, kind)
			defer stop()
			c := testConfig(srv.URL)
			c.Routes = append(c.Routes, Route{ID: "lcnd-egress-cn", ProxyURL: proxy})
			r := (&Prober{TLSConfig: &tls.Config{RootCAs: roots}}).Run(context.Background(), c)
			a := Assess(c, r)
			if a.Score == nil || *a.Score != 100 || len(r) != 6 {
				t.Fatalf("assessment=%+v results=%+v", a, r)
			}
			for _, s := range r {
				if s.RouteID != "direct" && (s.DNS.Status != "not_observed" || s.TCPScope != "proxy_endpoint") {
					t.Fatalf("invented target DNS/TCP evidence: %+v", s)
				}
			}
		})
	}
}

func TestProxyFailureDoesNotFallBackToDirect(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(204) }))
	defer srv.Close()
	roots := x509.NewCertPool()
	roots.AddCert(srv.Certificate())
	// This listener deterministically rejects HTTP CONNECT without changing any host service.
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(403) }))
	defer proxy.Close()
	c := testConfig(srv.URL)
	c.Routes = append(c.Routes, Route{ID: "lcnd-egress-cn", ProxyURL: proxy.URL})
	r := (&Prober{TLSConfig: &tls.Config{RootCAs: roots}}).Run(context.Background(), c)
	if calls.Load() != 3 || Assess(c, r).Status == "healthy" {
		t.Fatalf("direct calls=%d results=%+v", calls.Load(), r)
	}
	report := NewReport("test", time.Now(), Host{}, c, r)
	if len(report.Paths) != 2 || report.Paths[0].Assessment.Status != "healthy" || report.Paths[1].Assessment.Status == "healthy" || report.Paths[1].HTTPPassed != 0 {
		t.Fatalf("paths=%+v", report.Paths)
	}
}

func TestStalledProxyHandshakeIsBounded(t *testing.T) {
	for _, kind := range []string{"http", "socks5h"} {
		t.Run(kind, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			var connections []net.Conn
			done := make(chan struct{})
			go func() {
				defer close(done)
				for {
					conn, err := listener.Accept()
					if err != nil {
						return
					}
					mu.Lock()
					connections = append(connections, conn)
					mu.Unlock()
				}
			}()
			defer func() {
				_ = listener.Close()
				<-done
				mu.Lock()
				defer mu.Unlock()
				for _, conn := range connections {
					_ = conn.Close()
				}
			}()
			c := testConfig("https://example.invalid/")
			c.TimeoutMS = 100
			c.Routes = append(c.Routes, Route{ID: "lcnd-egress-global", ProxyURL: kind + "://" + listener.Addr().String()})
			started := time.Now()
			results := (&Prober{resolver: failingResolver{}}).Run(context.Background(), c)
			if time.Since(started) > 2*time.Second {
				t.Fatal("proxy handshake exceeded request budget")
			}
			for _, s := range results {
				if s.RouteID != "direct" && (s.TLS.Code != "TIMEOUT" || s.HTTP.Status != "blocked") {
					t.Fatalf("unexpected proxy timeout evidence: %+v", s)
				}
			}
		})
	}
}

func testProxy(t *testing.T, kind string) (string, func()) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { _ = conn.Close() }()
				_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
				var address string
				if kind == "http" {
					req, err := http.ReadRequest(bufio.NewReader(conn))
					if err != nil {
						return
					}
					if req.Method != "CONNECT" {
						return
					}
					address = req.Host
				} else {
					greeting := make([]byte, 3)
					if _, err := io.ReadFull(conn, greeting); err != nil {
						return
					}
					if greeting[0] != 5 {
						return
					}
					if _, err := conn.Write([]byte{5, 0}); err != nil {
						return
					}
					head := make([]byte, 4)
					if _, err := io.ReadFull(conn, head); err != nil {
						return
					}
					var host string
					switch head[3] {
					case 1:
						b := make([]byte, 4)
						if _, err := io.ReadFull(conn, b); err != nil {
							return
						}
						host = net.IP(b).String()
					case 4:
						b := make([]byte, 16)
						if _, err := io.ReadFull(conn, b); err != nil {
							return
						}
						host = net.IP(b).String()
					case 3:
						var n [1]byte
						if _, err := io.ReadFull(conn, n[:]); err != nil {
							return
						}
						b := make([]byte, int(n[0]))
						if _, err := io.ReadFull(conn, b); err != nil {
							return
						}
						host = string(b)
					default:
						return
					}
					var port [2]byte
					if _, err := io.ReadFull(conn, port[:]); err != nil {
						return
					}
					address = net.JoinHostPort(host, strconv.Itoa(int(port[0])*256+int(port[1])))
				}
				// Test fixture must never forward outside its own loopback server.
				host, _, err := net.SplitHostPort(address)
				if err != nil || !net.ParseIP(host).IsLoopback() {
					return
				}
				upstream, err := net.DialTimeout("tcp", address, time.Second)
				if err != nil {
					return
				}
				defer func() { _ = upstream.Close() }()
				_ = upstream.SetDeadline(time.Now().Add(3 * time.Second))
				if kind == "http" {
					_, err = fmt.Fprint(conn, "HTTP/1.1 200 Connection Established\r\n\r\n")
				} else {
					_, err = conn.Write([]byte{5, 0, 0, 1, 127, 0, 0, 1, 0, 0})
				}
				if err != nil {
					return
				}
				done := make(chan struct{})
				go func() { _, _ = io.Copy(upstream, conn); _ = upstream.Close(); close(done) }()
				_, _ = io.Copy(conn, upstream)
				_ = conn.Close()
				<-done
			}()
		}
	}()
	return (&url.URL{Scheme: kind, Host: l.Addr().String()}).String(), func() { _ = l.Close(); wg.Wait() }
}
