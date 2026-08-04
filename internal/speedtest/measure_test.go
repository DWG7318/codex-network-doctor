package speedtest

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMeasureDirectReadsEntireResponse(t *testing.T) {
	t.Parallel()

	payload := make([]byte, 64*1024)
	target := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write(payload)
	}))
	defer target.Close()

	transport := target.Client().Transport.(*http.Transport).Clone()
	transport.Proxy = nil
	sample := Measure(context.Background(), Request{
		Path:      PathSystem,
		Attempt:   1,
		URL:       target.URL,
		Transport: transport,
	})

	if sample.Error != "" {
		t.Fatalf("Measure returned error %q", sample.Error)
	}
	if sample.Bytes != int64(len(payload)) {
		t.Fatalf("Bytes = %d, want %d", sample.Bytes, len(payload))
	}
	if sample.BytesPerSecond <= 0 || sample.Total <= 0 {
		t.Fatalf("invalid timing sample: %#v", sample)
	}
}

func TestSOCKS5DialContextUsesRemoteDomainAndRelaysTraffic(t *testing.T) {
	t.Parallel()

	target := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, "through socks")
	}))
	defer target.Close()
	targetAddress := target.Listener.Addr().String()

	proxyListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer proxyListener.Close()
	domainSeen := make(chan string, 1)
	go serveSingleSOCKSConnection(proxyListener, targetAddress, domainSeen)

	dialContext := SOCKS5DialContext(proxyListener.Addr().String(), 2*time.Second)
	connection, err := dialContext(context.Background(), "tcp", "speed.example:80")
	if err != nil {
		t.Fatalf("SOCKS5DialContext: %v", err)
	}
	defer connection.Close()
	_, _ = fmt.Fprint(connection, "GET / HTTP/1.1\r\nHost: speed.example\r\nConnection: close\r\n\r\n")
	response, err := http.ReadResponse(bufio.NewReader(connection), &http.Request{Method: http.MethodGet})
	if err != nil {
		t.Fatalf("read relayed response: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != "through socks" {
		t.Fatalf("body = %q", body)
	}
	if got := <-domainSeen; got != "speed.example" {
		t.Fatalf("SOCKS domain = %q, want speed.example", got)
	}
}

func serveSingleSOCKSConnection(listener net.Listener, targetAddress string, domainSeen chan<- string) {
	client, err := listener.Accept()
	if err != nil {
		return
	}
	defer client.Close()
	reader := bufio.NewReader(client)
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return
	}
	methods := make([]byte, int(header[1]))
	if _, err := io.ReadFull(reader, methods); err != nil {
		return
	}
	_, _ = client.Write([]byte{5, 0})
	request := make([]byte, 4)
	if _, err := io.ReadFull(reader, request); err != nil || request[3] != 3 {
		return
	}
	length, err := reader.ReadByte()
	if err != nil {
		return
	}
	domain := make([]byte, int(length))
	if _, err := io.ReadFull(reader, domain); err != nil {
		return
	}
	domainSeen <- string(domain)
	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(reader, portBytes); err != nil {
		return
	}
	_ = binary.BigEndian.Uint16(portBytes)
	upstream, err := net.Dial("tcp", targetAddress)
	if err != nil {
		_, _ = client.Write([]byte{5, 1, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}
	defer upstream.Close()
	_, _ = client.Write([]byte{5, 0, 0, 1, 127, 0, 0, 1, 0, 0})
	go func() { _, _ = io.Copy(upstream, reader) }()
	_, _ = io.Copy(client, upstream)
}
