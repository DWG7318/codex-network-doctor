package proxyrepair

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

func probeHTTPProxy(ctx context.Context, candidate Candidate) error {
	if !validCandidate(candidate) {
		return fmt.Errorf("invalid local proxy endpoint %s:%d", candidate.Host, candidate.Port)
	}
	endpoint := net.JoinHostPort(candidate.Host, fmt.Sprintf("%d", candidate.Port))
	connection, err := (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return fmt.Errorf("connect to local listener: %w", err)
	}
	defer func() { _ = connection.Close() }()
	if deadline, ok := ctx.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	} else {
		_ = connection.SetDeadline(time.Now().Add(3 * time.Second))
	}
	request := "CONNECT chatgpt.com:443 HTTP/1.1\r\nHost: chatgpt.com:443\r\nUser-Agent: cnd-network-repair\r\nProxy-Connection: keep-alive\r\n\r\n"
	if _, err := connection.Write([]byte(request)); err != nil {
		return fmt.Errorf("write HTTP CONNECT probe: %w", err)
	}
	response, err := http.ReadResponse(bufio.NewReader(connection), &http.Request{Method: http.MethodConnect})
	if err != nil {
		return fmt.Errorf("read HTTP CONNECT response: %w", err)
	}
	if response.Body != nil {
		_ = response.Body.Close()
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("HTTP CONNECT returned %s", response.Status)
	}
	return nil
}

func verifyThroughProxy(
	ctx context.Context,
	candidate Candidate,
	target string,
	baseTransport *http.Transport,
) (int, error) {
	if !validCandidate(candidate) {
		return 0, fmt.Errorf("invalid local proxy endpoint %s:%d", candidate.Host, candidate.Port)
	}
	if baseTransport == nil {
		baseTransport = http.DefaultTransport.(*http.Transport)
	}
	transport := baseTransport.Clone()
	proxyURL := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(candidate.Host, fmt.Sprintf("%d", candidate.Port)),
	}
	transport.Proxy = http.ProxyURL(proxyURL)
	transport.DialContext = (&net.Dialer{
		Timeout:   4 * time.Second,
		KeepAlive: 15 * time.Second,
	}).DialContext
	defer transport.CloseIdleConnections()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return 0, fmt.Errorf("create verification request: %w", err)
	}
	request.Header.Set("User-Agent", "cnd-network-repair")
	response, err := (&http.Client{Transport: transport}).Do(request)
	if err != nil {
		return 0, fmt.Errorf("request through local proxy: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	return response.StatusCode, nil
}
