package collect

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"strings"
	"sync"
)

const httpBodyRecoveryVersion = "http10-incomplete-chunk.v1"

// A few providers close HTTP/1.1 chunked responses without the final chunk.
// Negotiate a fresh close-delimited HTTP/1.0 representation, never accept the
// truncated first body. TLS, destination validation and the deadline stay intact.
func (d HTTPDriver) RecoverBody(ctx context.Context, job Job, previous Response, cause error) (Response, error) {
	if ctx.Err() != nil {
		return Response{}, ctx.Err()
	}
	if !errors.Is(cause, io.ErrUnexpectedEOF) || !previous.Chunked || previous.Status < 200 || previous.Status >= 300 {
		return Response{}, cause
	}
	job.ETag, job.Modified = "", ""
	response, err := d.fetch(ctx, job, true)
	if err == nil {
		response.RecoveryRule = httpBodyRecoveryVersion
	}
	return response, err
}

type http10Transport struct {
	dial func(context.Context, string, string) (net.Conn, error)
}

func (t http10Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != "GET" || req.Body != nil {
		return nil, errors.New("http10_method_unsupported")
	}
	if err := ValidateURL(req.URL.String()); err != nil {
		return nil, err
	}
	host := req.Host
	if host == "" {
		host = req.URL.Host
	}
	uri := req.URL.RequestURI()
	if strings.ContainsAny(host, "\r\n") || strings.ContainsAny(uri, "\r\n") {
		return nil, ErrBlocked
	}
	port := req.URL.Port()
	if port == "" {
		if req.URL.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	conn, err := t.dial(req.Context(), "tcp", net.JoinHostPort(req.URL.Hostname(), port))
	if err != nil {
		return nil, err
	}
	rawConn := conn
	stop := context.AfterFunc(req.Context(), func() { _ = rawConn.Close() })
	success := false
	defer func() {
		if !success {
			stop()
			_ = rawConn.Close()
		}
	}()
	if deadline, ok := req.Context().Deadline(); ok {
		if err = conn.SetDeadline(deadline); err != nil {
			return nil, err
		}
	}
	if req.URL.Scheme == "https" {
		secured := tls.Client(conn, &tls.Config{ServerName: req.URL.Hostname(), MinVersion: tls.VersionTLS12})
		if err = secured.HandshakeContext(req.Context()); err != nil {
			return nil, err
		}
		conn = secured
	}
	headers := req.Header.Clone()
	headers.Set("Connection", "close")
	headers.Set("Accept-Encoding", "identity")
	headers.Del("Content-Length")
	headers.Del("Transfer-Encoding")
	headers.Del("Host")
	var request bytes.Buffer
	fmt.Fprintf(&request, "GET %s HTTP/1.0\r\nHost: %s\r\n", uri, host)
	if err = headers.Write(&request); err != nil {
		return nil, err
	}
	request.WriteString("\r\n")
	if _, err = io.Copy(conn, &request); err != nil {
		return nil, err
	}
	reader := bufio.NewReader(conn)
	var header bytes.Buffer
	fragmented := false
	for {
		line, readErr := reader.ReadSlice('\n')
		if header.Len()+len(line) > 64*1024 {
			return nil, errors.New("http10_headers_too_large")
		}
		header.Write(line)
		if readErr != nil {
			if errors.Is(readErr, bufio.ErrBufferFull) {
				fragmented = true
				continue
			}
			return nil, readErr
		}
		if !fragmented && bytes.Equal(line, []byte("\r\n")) {
			break
		}
		fragmented = false
	}
	// net/http intentionally ignores Transfer-Encoding on HTTP/1.0. Reject
	// that invalid framing before it could be mistaken for a complete raw body.
	preview := textproto.NewReader(bufio.NewReader(bytes.NewReader(header.Bytes())))
	statusLine, err := preview.ReadLine()
	if err != nil {
		return nil, err
	}
	fields, err := preview.ReadMIMEHeader()
	if err != nil {
		return nil, err
	}
	if _, present := fields["Transfer-Encoding"]; present && strings.HasPrefix(statusLine, "HTTP/1.0 ") {
		return nil, fmt.Errorf("%w: invalid HTTP/1.0 transfer encoding", io.ErrUnexpectedEOF)
	}
	response, err := http.ReadResponse(bufio.NewReader(io.MultiReader(bytes.NewReader(header.Bytes()), reader)), req)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 {
		response.Body.Close()
		return nil, errors.New("http10_interim_response_unsupported")
	}
	response.Body = &http10Body{ReadCloser: response.Body, conn: conn, stop: stop}
	success = true
	return response, nil
}

type http10Body struct {
	io.ReadCloser
	conn net.Conn
	stop func() bool
	once sync.Once
	err  error
}

func (b *http10Body) Close() error {
	b.once.Do(func() { b.stop(); _ = b.conn.Close(); b.err = b.ReadCloser.Close() })
	return b.err
}
