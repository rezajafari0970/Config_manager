package collect

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

var ErrBlocked = errors.New("blocked_destination")

type HTTPDriver struct {
	Resolver *net.Resolver
	Dial     func(context.Context, string, string) (net.Conn, error)
}

func (HTTPDriver) Name() string { return "http.v1" }
func PublicIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.Zone() != "" {
		return false
	}
	for _, cidr := range []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001:db8::/32", "64:ff9b::/96", "64:ff9b:1::/48", "2001::/32", "2002::/16"} {
		if netip.MustParsePrefix(cidr).Contains(ip) {
			return false
		}
	}
	return true
}
func ValidateURL(raw string) error {
	u, e := url.Parse(raw)
	if e != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.Opaque != "" {
		return ErrBlocked
	}
	h := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if h == "localhost" || strings.HasSuffix(h, ".localhost") || strings.HasSuffix(h, ".local") {
		return ErrBlocked
	}
	if ip, e := netip.ParseAddr(h); e == nil && !PublicIP(ip) {
		return ErrBlocked
	}
	return nil
}
func (d HTTPDriver) Fetch(ctx context.Context, job Job) (Response, error) {
	if err := ValidateURL(job.URL); err != nil {
		return Response{}, err
	}
	resolver := d.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	dial := d.Dial
	if dial == nil {
		dial = (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	}
	transport := &http.Transport{Proxy: nil, ForceAttemptHTTP2: true, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 15 * time.Second, MaxIdleConns: 1, MaxIdleConnsPerHost: 1, MaxConnsPerHost: 1, IdleConnTimeout: time.Second, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := resolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, errors.New("dns_empty")
		}
		for _, ip := range ips {
			if !PublicIP(ip) {
				return nil, ErrBlocked
			}
		}
		var last error
		for _, ip := range ips {
			c, e := dial(ctx, network, net.JoinHostPort(ip.Unmap().String(), port))
			if e == nil {
				return c, nil
			}
			last = e
			if ctx.Err() != nil {
				break
			}
		}
		return nil, last
	}}
	client := &http.Client{Transport: transport, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 20 {
			return errors.New("redirect_loop")
		}
		if err := ValidateURL(req.URL.String()); err != nil {
			return err
		}
		if len(via) > 0 {
			previous := via[len(via)-1]
			if previous.URL.Scheme == "https" && req.URL.Scheme != "https" {
				return ErrBlocked
			}
			if req.URL.Host != previous.URL.Host {
				req.Header.Del("Authorization")
				req.Header.Del("If-None-Match")
				req.Header.Del("If-Modified-Since")
			}
		}
		return nil
	}}
	request, err := http.NewRequestWithContext(ctx, "GET", job.URL, nil)
	if err != nil {
		return Response{}, err
	}
	request.Header.Set("User-Agent", "ConfigManager/3")
	request.Header.Set("Accept", "*/*")
	if job.ETag != "" {
		request.Header.Set("If-None-Match", job.ETag)
	}
	if job.Modified != "" {
		request.Header.Set("If-Modified-Since", job.Modified)
	}
	response, err := client.Do(request)
	if err != nil {
		transport.CloseIdleConnections()
		return Response{}, err
	}
	return Response{Body: &transportBody{ioReadCloser: response.Body, closeIdle: transport.CloseIdleConnections}, Status: response.StatusCode, ETag: response.Header.Get("ETag"), Modified: response.Header.Get("Last-Modified")}, nil
}

type transportBody struct {
	ioReadCloser
	closeIdle func()
}
type ioReadCloser interface {
	Read([]byte) (int, error)
	Close() error
}

func (b *transportBody) Close() error { e := b.ioReadCloser.Close(); b.closeIdle(); return e }
