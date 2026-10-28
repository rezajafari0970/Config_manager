package panel

import (
	"errors"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var ErrInvalid = errors.New("invalid source")
var ErrConflict = errors.New("source changed or duplicate")
var ErrNotFound = errors.New("not found")

type SourceInput struct {
	Name     string
	URL      string
	Min, Max int64
	Unit     string
	Enabled  bool
}

type Source struct {
	SourceInput
	ID                              string
	DisplayURL                      string
	Revision                        int64
	CreatedAt, UpdatedAt, NextRunAt time.Time
}

func (s SourceInput) durations() (time.Duration, time.Duration, error) {
	units := map[string]int64{"seconds": 1, "minutes": 60, "hours": 3600}
	unit, ok := units[s.Unit]
	if !ok || s.Min < 1 || s.Max < s.Min || s.Max > 30*24*3600/unit {
		return 0, 0, ErrInvalid
	}
	return time.Duration(s.Min*unit) * time.Second, time.Duration(s.Max*unit) * time.Second, nil
}

func validateSource(s SourceInput) (string, error) {
	if strings.TrimSpace(s.Name) == "" || utf8.RuneCountInString(s.Name) > 120 || len(s.URL) > 8192 || s.URL == "" {
		return "", ErrInvalid
	}
	for _, value := range []string{s.Name, s.URL} {
		for _, r := range value {
			if unicode.IsControl(r) {
				return "", ErrInvalid
			}
		}
	}
	if _, _, err := s.durations(); err != nil {
		return "", err
	}
	u, err := url.Parse(s.URL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.Fragment != "" || u.Opaque != "" {
		return "", ErrInvalid
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || !strings.Contains(host, ".") && !strings.Contains(host, ":") {
		return "", ErrInvalid
	}
	if strings.ContainsAny(host, " %/\\") {
		return "", ErrInvalid
	}
	if port := u.Port(); port != "" {
		n, e := strconv.Atoi(port)
		if e != nil || n < 1 || n > 65535 {
			return "", ErrInvalid
		}
	}
	if ip, e := netip.ParseAddr(host); e == nil {
		ip = ip.Unmap()
		if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.Zone() != "" {
			return "", ErrInvalid
		}
		for _, p := range []string{"100.64.0.0/10", "192.0.0.0/24", "198.18.0.0/15"} {
			if netip.MustParsePrefix(p).Contains(ip) {
				return "", ErrInvalid
			}
		}
	} else if strings.Contains(host, ":") || net.ParseIP(host) != nil {
		return "", ErrInvalid
	}
	// DNS is deliberately not resolved here. P3 must re-check every resolved address,
	// redirect and actual dial target to prevent DNS rebinding and private egress.
	return u.Scheme + "://" + u.Host + "/…", nil
}
