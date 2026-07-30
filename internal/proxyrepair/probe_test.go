package proxyrepair

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProbeHTTPProxyAcceptsConnectResponse(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer func() { _ = connection.Close() }()
		reader := bufio.NewReader(connection)
		for {
			line, readErr := reader.ReadString('\n')
			if readErr != nil || line == "\r\n" {
				break
			}
		}
		_, _ = fmt.Fprint(connection, "HTTP/1.1 200 Connection established\r\n\r\n")
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := probeHTTPProxy(ctx, Candidate{Host: "127.0.0.1", Port: port}); err != nil {
		t.Fatalf("probeHTTPProxy returned error: %v", err)
	}
}

func TestVerifyThroughProxyCompletesTLSRequest(t *testing.T) {
	t.Parallel()

	target := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()

	proxy := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodConnect {
			http.Error(writer, "CONNECT required", http.StatusMethodNotAllowed)
			return
		}
		upstream, err := net.Dial("tcp", request.Host)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadGateway)
			return
		}
		hijacker := writer.(http.Hijacker)
		client, _, err := hijacker.Hijack()
		if err != nil {
			_ = upstream.Close()
			return
		}
		_, _ = fmt.Fprint(client, "HTTP/1.1 200 Connection established\r\n\r\n")
		go func() {
			defer func() { _ = upstream.Close() }()
			defer func() { _ = client.Close() }()
			_, _ = io.Copy(upstream, client)
		}()
		go func() {
			defer func() { _ = upstream.Close() }()
			defer func() { _ = client.Close() }()
			_, _ = io.Copy(client, upstream)
		}()
	}))
	defer proxy.Close()

	proxyPort := proxy.Listener.Addr().(*net.TCPAddr).Port
	transport := target.Client().Transport.(*http.Transport).Clone()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	status, err := verifyThroughProxy(
		ctx,
		Candidate{Host: "127.0.0.1", Port: proxyPort},
		target.URL,
		transport,
	)
	if err != nil {
		t.Fatalf("verifyThroughProxy returned error: %v", err)
	}
	if status != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", status, http.StatusNoContent)
	}
}
