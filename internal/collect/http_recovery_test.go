package collect

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func recoveryFixture(fn func(*http.Request) string) (HTTPDriver, *[]string, *sync.Mutex) {
	versions := []string{}
	mu := &sync.Mutex{}
	d := HTTPDriver{Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		c, s := net.Pipe()
		go func() {
			defer s.Close()
			r, e := http.ReadRequest(bufio.NewReader(s))
			if e != nil {
				return
			}
			mu.Lock()
			versions = append(versions, r.Proto)
			mu.Unlock()
			_, _ = io.WriteString(s, fn(r))
		}()
		return c, nil
	}}
	return d, &versions, mu
}
func TestHTTPBodyRecoveryDiscardsPartialAndPreservesFullRaw(t *testing.T) {
	good := "vless://00000000-0000-4000-8000-000000000001@8.8.8.8:443?security=tls#fixture\n"
	d, versions, mu := recoveryFixture(func(r *http.Request) string {
		if r.Proto == "HTTP/1.1" {
			return "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n4\r\nbad!\r\n"
		}
		if r.Header.Get("If-None-Match") != "" || r.Header.Get("If-Modified-Since") != "" || r.Header.Get("Accept-Encoding") != "identity" {
			return "HTTP/1.0 400 Bad Request\r\n\r\n"
		}
		return "HTTP/1.0 200 OK\r\nETag: fresh\r\n\r\n" + good
	})
	e, repo := pipelineFixture(t, 1)
	e.fetch = d
	e.runJob(context.Background(), Job{URL: "http://8.8.8.8/sub", ETag: "old", Modified: "yesterday"})
	if repo.result.State != "updated" || repo.items != 1 || repo.result.ETag != "fresh" {
		t.Fatalf("recovery failed: %+v", repo.result)
	}
	raw, err := e.vault.Open(repo.result.Raw.Hash)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	b, err := io.ReadAll(raw)
	if err != nil || string(b) != good {
		t.Fatal("partial bytes or changed raw", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(*versions, ",") != "HTTP/1.1,HTTP/1.0" {
		t.Fatal(*versions)
	}
}
func TestHTTPBodyRecoveryNeverAcceptsTruncationOrLoops(t *testing.T) {
	for _, second := range []string{"HTTP/1.0 200 OK\r\nContent-Length: 900\r\n\r\nshort", "HTTP/1.0 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n4\r\nbad!\r\n"} {
		d, versions, mu := recoveryFixture(func(r *http.Request) string {
			if r.Proto == "HTTP/1.0" {
				return second
			}
			return "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n4\r\nbad!\r\n"
		})
		e, repo := pipelineFixture(t, 1)
		e.fetch = d
		e.runJob(context.Background(), Job{URL: "http://8.8.8.8/sub"})
		if repo.result.ErrorCode != "incomplete_body" || repo.batches != 0 || repo.result.Raw.Hash != "" {
			t.Fatal(repo.result)
		}
		mu.Lock()
		n := len(*versions)
		mu.Unlock()
		if n != 2 {
			t.Fatal("unbounded retry", n)
		}
	}
}
func TestHTTPBodyRecoveryScopeAndRedirectGuard(t *testing.T) {
	for _, initial := range []string{"HTTP/1.1 200 OK\r\nContent-Length: 900\r\n\r\nshort", "HTTP/1.1 500 Failure\r\nTransfer-Encoding: chunked\r\n\r\n4\r\nbad!\r\n"} {
		d, versions, mu := recoveryFixture(func(*http.Request) string { return initial })
		e, repo := pipelineFixture(t, 1)
		e.fetch = d
		e.runJob(context.Background(), Job{URL: "http://8.8.8.8/sub"})
		mu.Lock()
		n := len(*versions)
		mu.Unlock()
		if n != 1 || repo.batches != 0 {
			t.Fatal("invalid recovery scope", n, repo.result)
		}
	}
	d, versions, mu := recoveryFixture(func(*http.Request) string {
		return "HTTP/1.0 302 Found\r\nLocation: http://127.0.0.1/private\r\nContent-Length: 0\r\n\r\n"
	})
	_, err := d.RecoverBody(context.Background(), Job{URL: "http://8.8.8.8/sub"}, Response{Status: 200, Chunked: true}, io.ErrUnexpectedEOF)
	if !errors.Is(err, ErrBlocked) {
		t.Fatal(err)
	}
	mu.Lock()
	n := len(*versions)
	mu.Unlock()
	if n != 1 {
		t.Fatal("private redirect dialled")
	}
}
func TestHTTPBodyRecoveryCancellationAndHeaderBound(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d, versions, mu := recoveryFixture(func(*http.Request) string { return "" })
	_, err := d.RecoverBody(ctx, Job{URL: "http://8.8.8.8/sub"}, Response{Status: 200, Chunked: true}, io.ErrUnexpectedEOF)
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	mu.Lock()
	n := len(*versions)
	mu.Unlock()
	if n != 0 {
		t.Fatal("cancelled request dialled")
	}
	d, _, _ = recoveryFixture(func(*http.Request) string {
		return "HTTP/1.0 200 OK\r\nX-Large: " + strings.Repeat("x", 70*1024) + "\r\n\r\n"
	})
	_, err = d.RecoverBody(context.Background(), Job{URL: "http://8.8.8.8/sub"}, Response{Status: 200, Chunked: true}, io.ErrUnexpectedEOF)
	if err == nil || !strings.Contains(err.Error(), "headers_too_large") {
		t.Fatal(err)
	}
	d = HTTPDriver{Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		c, s := net.Pipe()
		go func() { defer s.Close(); _, _ = http.ReadRequest(bufio.NewReader(s)); <-ctx.Done() }()
		return c, nil
	}}
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = d.RecoverBody(ctx, Job{URL: "http://8.8.8.8/sub"}, Response{Status: 200, Chunked: true}, io.ErrUnexpectedEOF)
	if err == nil || time.Since(start) > time.Second {
		t.Fatal("deadline lost", err)
	}
}
func TestHTTPBodyRecoveryRetainsTLSVerification(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "must not be trusted") }))
	defer server.Close()
	d := HTTPDriver{Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := d.RecoverBody(ctx, Job{URL: "https://8.8.8.8/sub"}, Response{Status: 200, Chunked: true}, io.ErrUnexpectedEOF)
	if err == nil || !strings.Contains(err.Error(), "certificate") {
		t.Fatal("untrusted TLS accepted", err)
	}
}

func TestHTTPBodyRecoveryFragmentedHeaderBoundary(t *testing.T) {
	for _, large := range []bool{false, true} {
		d, _, _ := recoveryFixture(func(*http.Request) string {
			text := "HTTP/1.0 200 OK\r\nX-Pad: " + strings.Repeat("x", 4096-len("X-Pad: ")) + "\r\n"
			if large {
				text += "X-Next: " + strings.Repeat("y", 70*1024) + "\r\n"
			}
			return text + "\r\ncomplete"
		})
		response, err := d.RecoverBody(context.Background(), Job{URL: "http://8.8.8.8/sub"}, Response{Status: 200, Chunked: true}, io.ErrUnexpectedEOF)
		if large {
			if err == nil || !strings.Contains(err.Error(), "headers_too_large") {
				t.Fatal("fragment bypass", err)
			}
			continue
		}
		if err != nil {
			t.Fatal("valid fragmented header rejected", err)
		}
		b, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil || string(b) != "complete" {
			t.Fatal(string(b), err)
		}
	}
}
