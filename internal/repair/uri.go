package repair

import (
	"bytes"
	"github.com/rezajafari0970/Config_manager/internal/driver"
	"net/url"
	"strings"
)

// URIUserinfoDriver only encodes literal ASCII spaces in unambiguous credentials.
// It preserves their decoded bytes and every byte outside the userinfo.
type URIUserinfoDriver struct{ Protocol string }

func (d URIUserinfoDriver) Kind() string { return d.Protocol }
func (d URIUserinfoDriver) Propose(raw []byte) Diagnostic {
	result := baseDiagnostic(raw)
	s := string(raw)
	if _, err := url.Parse(strings.TrimSpace(s)); err == nil {
		return result
	}
	start := strings.Index(s, "://")
	if start < 0 {
		return result
	}
	start += 3
	tail := s[start:]
	end := strings.IndexAny(tail, "/?#")
	if end < 0 {
		end = len(tail)
	}
	authority := tail[:end]
	if strings.Count(authority, "@") != 1 {
		return result
	}
	at := strings.IndexByte(authority, '@')
	user := authority[:at]
	if !strings.Contains(user, " ") || strings.ContainsAny(user, "\r\n\t") {
		return result
	}
	candidate := []byte(s[:start] + strings.ReplaceAll(user, " ", "%20") + s[start+at:])
	u, err := url.Parse(strings.TrimSpace(string(candidate)))
	if err != nil || u.User == nil || u.Hostname() == "" || u.Port() == "" {
		return result
	}
	beforeUser, err := url.PathUnescape(user)
	if err != nil {
		return result
	}
	afterUser := u.User.Username()
	if password, ok := u.User.Password(); ok {
		afterUser += ":" + password
	}
	if beforeUser != afterUser || bytes.Equal(raw, candidate) {
		return result
	}
	result.Status = "repaired"
	result.Candidate = candidate
	result.CandidateSHA256 = Digest(candidate)
	result.Patches = []driver.Patch{{RuleID: RulesVersion + ".uri-userinfo-space", Path: "/userinfo", BeforeSHA256: Digest(raw), AfterSHA256: Digest(candidate)}}
	return result
}
