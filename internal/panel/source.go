package panel

import (
	"errors"
	"math"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var ErrInvalid = errors.New("invalid source")
var ErrConflict = errors.New("source changed or duplicate")
var ErrNotFound = errors.New("not found")

type SourceInput struct {
	Name           string  `json:"name"`
	URL            string  `json:"url"`
	Min            float64 `json:"min"`
	Max            float64 `json:"max"`
	Unit           string  `json:"unit"`
	Enabled        bool    `json:"enabled"`
	TimeoutSeconds int64   `json:"timeout_seconds"`
}
type Source struct {
	SourceInput
	ID             string    `json:"id"`
	DisplayURL     string    `json:"display_url"`
	Revision       int64     `json:"revision"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	NextRunAt      time.Time `json:"next_at"`
	State          string    `json:"state"`
	ErrorCode      string    `json:"error_code"`
	LastFetch      int64     `json:"last_fetch"`
	LastSuccess    int64     `json:"last_success"`
	HTTPStatus     int       `json:"http_status"`
	DurationMS     int64     `json:"duration_ms"`
	FetchCount     int64     `json:"fetch_count"`
	ConfigCount    int       `json:"config_count"`
	DuplicateCount int       `json:"duplicates"`
	HasResponse    bool      `json:"has_response"`
	ContentState   string    `json:"content_state"`
	ValidCount     int       `json:"valid_count"`
	InvalidCount   int       `json:"invalid_count"`
	UnknownCount   int       `json:"unknown_count"`
	ReceivedCount  int       `json:"received_count"`
	ResponseBytes  int64     `json:"response_bytes"`
	ErrorStreak    int       `json:"error_streak"`
}

func (s SourceInput) durations() (time.Duration, time.Duration, error) {
	unit, ok := map[string]float64{"seconds": float64(time.Second), "minutes": float64(time.Minute), "hours": float64(time.Hour)}[s.Unit]
	if !ok || s.Min <= 0 || s.Max < s.Min || math.IsNaN(s.Min) || math.IsNaN(s.Max) || math.IsInf(s.Max, 0) || s.Max*unit >= float64(math.MaxInt64) {
		return 0, 0, ErrInvalid
	}
	min, max := time.Duration(s.Min*unit), time.Duration(s.Max*unit)
	if min <= 0 || time.Now().Add(max).UnixNano() < 0 {
		return 0, 0, ErrInvalid
	}
	return min, max, nil
}
func validateSource(s SourceInput) (string, error) {
	if s.URL == "" || s.TimeoutSeconds < 0 || s.TimeoutSeconds > int64((1<<63-1)/int64(time.Second))-30 {
		return "", ErrInvalid
	}
	for _, value := range []string{s.Name, s.URL} {
		for _, r := range value {
			if unicode.IsControl(r) {
				return "", ErrInvalid
			}
		}
	}
	if _, _, e := s.durations(); e != nil {
		return "", e
	}
	u, e := url.Parse(s.URL)
	if e != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.Opaque != "" {
		return "", ErrInvalid
	}
	if port := u.Port(); port != "" {
		n, e := strconv.Atoi(port)
		if e != nil || n < 1 || n > 65535 {
			return "", ErrInvalid
		}
	}
	return u.Scheme + "://" + u.Host + "/…", nil
}
func AutoName(raw string) string {
	u, e := url.Parse(raw)
	if e != nil {
		return "Source"
	}
	p := strings.TrimSuffix(u.Path, "/")
	name := path.Base(path.Dir(p))
	if strings.HasSuffix(u.Path, "/") {
		name = path.Base(p)
	}
	if name == "." || name == "/" || name == "" {
		name = u.Hostname()
	}
	if decoded, e := url.PathUnescape(name); e == nil {
		name = decoded
	}
	return strings.TrimSpace(name)
}
func normalizeDigits(v string) string {
	return strings.Map(func(r rune) rune {
		if r >= '۰' && r <= '۹' {
			return r - '۰' + '0'
		}
		if r >= '٠' && r <= '٩' {
			return r - '٠' + '0'
		}
		if r == '٫' {
			return '.'
		}
		return r
	}, v)
}
func parseInterval(v string) (float64, float64, error) {
	v = normalizeDigits(strings.TrimSpace(v))
	for _, sep := range []string{"تا", "–", "—", "to"} {
		v = strings.ReplaceAll(v, sep, "-")
	}
	parts := strings.Split(v, "-")
	if len(parts) > 2 || len(parts) == 0 {
		return 0, 0, ErrInvalid
	}
	a, e := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if e != nil {
		return 0, 0, ErrInvalid
	}
	b := a
	if len(parts) == 2 {
		b, e = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	}
	if e != nil {
		return 0, 0, ErrInvalid
	}
	return a, b, nil
}
