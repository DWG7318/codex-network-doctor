package speedtest

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"time"
)

const defaultMaximumResponseBytes int64 = 16 << 20

type Request struct {
	Path                 Path
	Attempt              int
	URL                  string
	Transport            *http.Transport
	MaximumResponseBytes int64
}

func Measure(ctx context.Context, request Request) Sample {
	sample := Sample{Path: request.Path, Attempt: request.Attempt}
	if request.URL == "" {
		sample.Error = "speed target URL is empty"
		return sample
	}
	transport := request.Transport
	if transport == nil {
		transport = http.DefaultTransport.(*http.Transport).Clone()
	} else {
		transport = transport.Clone()
	}
	transport.DisableKeepAlives = true
	defer transport.CloseIdleConnections()

	started := time.Now()
	var dnsStarted, connectStarted, tlsStarted time.Time
	trace := &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) { dnsStarted = time.Now() },
		DNSDone: func(httptrace.DNSDoneInfo) {
			if !dnsStarted.IsZero() {
				sample.DNS = time.Since(dnsStarted)
			}
		},
		ConnectStart: func(_, _ string) { connectStarted = time.Now() },
		ConnectDone: func(_, _ string, _ error) {
			if !connectStarted.IsZero() {
				sample.Connect = time.Since(connectStarted)
			}
		},
		TLSHandshakeStart: func() { tlsStarted = time.Now() },
		TLSHandshakeDone: func(tls.ConnectionState, error) {
			if !tlsStarted.IsZero() {
				sample.TLS = time.Since(tlsStarted)
			}
		},
		GotFirstResponseByte: func() { sample.TTFB = time.Since(started) },
	}
	httpRequest, err := http.NewRequestWithContext(
		httptrace.WithClientTrace(ctx, trace),
		http.MethodGet,
		request.URL,
		nil,
	)
	if err != nil {
		sample.Error = err.Error()
		return sample
	}
	httpRequest.Header.Set("User-Agent", "CND-Speed-Diagnostics/1")
	response, err := (&http.Client{Transport: transport}).Do(httpRequest)
	if err != nil {
		sample.Total = time.Since(started)
		sample.Error = err.Error()
		return sample
	}
	defer response.Body.Close()
	sample.StatusCode = response.StatusCode
	limit := request.MaximumResponseBytes
	if limit <= 0 {
		limit = defaultMaximumResponseBytes
	}
	sample.Bytes, err = io.Copy(io.Discard, io.LimitReader(response.Body, limit+1))
	sample.Total = time.Since(started)
	if err != nil {
		sample.Error = err.Error()
		return sample
	}
	if sample.Bytes > limit {
		sample.Error = fmt.Sprintf("response exceeded %d byte safety limit", limit)
		return sample
	}
	if sample.Total > 0 {
		sample.BytesPerSecond = float64(sample.Bytes) / sample.Total.Seconds()
	}
	if response.StatusCode < 200 || response.StatusCode >= 400 {
		sample.Error = fmt.Sprintf("HTTP status %d", response.StatusCode)
	}
	return sample
}

func SOCKS5DialContext(endpoint string, timeout time.Duration) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if network != "tcp" && network != "tcp4" && network != "tcp6" {
			return nil, fmt.Errorf("SOCKS5 only supports TCP, got %q", network)
		}
		host, portText, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("parse SOCKS5 destination: %w", err)
		}
		port, err := strconv.Atoi(portText)
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("invalid SOCKS5 destination port %q", portText)
		}
		connection, err := (&net.Dialer{Timeout: timeout}).DialContext(ctx, "tcp", endpoint)
		if err != nil {
			return nil, fmt.Errorf("connect SOCKS5 endpoint: %w", err)
		}
		failed := true
		defer func() {
			if failed {
				_ = connection.Close()
			}
		}()
		if deadline, ok := ctx.Deadline(); ok {
			_ = connection.SetDeadline(deadline)
		} else if timeout > 0 {
			_ = connection.SetDeadline(time.Now().Add(timeout))
		}
		if _, err := connection.Write([]byte{5, 1, 0}); err != nil {
			return nil, fmt.Errorf("write SOCKS5 greeting: %w", err)
		}
		greeting := make([]byte, 2)
		if _, err := io.ReadFull(connection, greeting); err != nil {
			return nil, fmt.Errorf("read SOCKS5 greeting: %w", err)
		}
		if greeting[0] != 5 || greeting[1] != 0 {
			return nil, fmt.Errorf("SOCKS5 endpoint rejected no-auth method")
		}
		requestBytes, err := socksConnectRequest(host, port)
		if err != nil {
			return nil, err
		}
		if _, err := connection.Write(requestBytes); err != nil {
			return nil, fmt.Errorf("write SOCKS5 connect request: %w", err)
		}
		reply := make([]byte, 4)
		if _, err := io.ReadFull(connection, reply); err != nil {
			return nil, fmt.Errorf("read SOCKS5 connect reply: %w", err)
		}
		if reply[0] != 5 || reply[1] != 0 {
			return nil, fmt.Errorf("SOCKS5 connect failed with code %d", reply[1])
		}
		if err := discardSOCKSAddress(connection, reply[3]); err != nil {
			return nil, err
		}
		_ = connection.SetDeadline(time.Time{})
		failed = false
		return connection, nil
	}
}

func socksConnectRequest(host string, port int) ([]byte, error) {
	request := []byte{5, 1, 0}
	if ip := net.ParseIP(host); ip != nil {
		if ipv4 := ip.To4(); ipv4 != nil {
			request = append(request, 1)
			request = append(request, ipv4...)
		} else {
			request = append(request, 4)
			request = append(request, ip.To16()...)
		}
	} else {
		if len(host) == 0 || len(host) > 255 {
			return nil, fmt.Errorf("invalid SOCKS5 destination hostname length")
		}
		request = append(request, 3, byte(len(host)))
		request = append(request, host...)
	}
	return append(request, byte(port>>8), byte(port)), nil
}

func discardSOCKSAddress(reader io.Reader, addressType byte) error {
	length := 0
	switch addressType {
	case 1:
		length = 4
	case 4:
		length = 16
	case 3:
		value := []byte{0}
		if _, err := io.ReadFull(reader, value); err != nil {
			return fmt.Errorf("read SOCKS5 reply hostname length: %w", err)
		}
		length = int(value[0])
	default:
		return fmt.Errorf("unknown SOCKS5 reply address type %d", addressType)
	}
	buffer := make([]byte, length+2)
	if _, err := io.ReadFull(reader, buffer); err != nil {
		return fmt.Errorf("read SOCKS5 reply address: %w", err)
	}
	return nil
}
