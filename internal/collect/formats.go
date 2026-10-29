package collect

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"golang.org/x/sys/unix"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// Format drivers only extract. Raw files are never re-serialized or repaired.
type Registry struct{ drivers []FormatDriver }

func NewRegistry() *Registry {
	return &Registry{drivers: []FormatDriver{wireGuardFormat{}, textFormat{}}}
}
func (r *Registry) Register(driver FormatDriver) {
	r.drivers = append([]FormatDriver{driver}, r.drivers...)
}
func (r *Registry) Extract(ctx context.Context, path string, emit Emitter) error {
	return r.extract(ctx, path, emit, true)
}
func (r *Registry) extract(ctx context.Context, path string, emit Emitter, allowEnvelope bool) error {
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	prefix := make([]byte, 4096)
	n, _ := f.Read(prefix)
	f.Close()
	prefix = bytes.TrimSpace(bytes.TrimPrefix(prefix[:n], []byte{0xef, 0xbb, 0xbf}))
	if len(prefix) == 0 {
		return nil
	}
	count := 0
	wrapped := func(ctx context.Context, r io.Reader, kind string) error { count++; return emit(ctx, r, kind) }
	for _, driver := range r.drivers {
		if driver.Match(prefix) {
			e = driver.Extract(ctx, path, wrapped)
			break
		}
	}
	if e != nil || count > 0 || !allowEnvelope {
		return e
	}
	if !base64Prefix(prefix) {
		return nil
	}
	source, e := os.Open(path)
	if e != nil {
		return e
	}
	defer source.Close()
	out, e := os.CreateTemp("", "config-envelope-")
	if e != nil {
		return e
	}
	defer os.Remove(out.Name())
	defer out.Close()
	encoding := base64.RawStdEncoding
	if bytes.ContainsAny(prefix, "-_") {
		encoding = base64.RawURLEncoding
	}
	_, e = io.Copy(out, base64.NewDecoder(encoding, &base64Filter{r: bufio.NewReader(&contextReader{ctx, source})}))
	if e != nil {
		return nil
	}
	if e = out.Close(); e != nil {
		return e
	}
	return r.extract(ctx, out.Name(), emit, false)
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if e := r.ctx.Err(); e != nil {
		return 0, e
	}
	return r.r.Read(p)
}

type base64Filter struct{ r *bufio.Reader }

func (r *base64Filter) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		b, e := r.r.ReadByte()
		if e != nil {
			if n > 0 {
				return n, nil
			}
			return 0, e
		}
		if b == ' ' || b == '\r' || b == '\n' || b == '\t' || b == '=' {
			continue
		}
		p[n] = b
		n++
		if r.r.Buffered() == 0 {
			break
		}
	}
	return n, nil
}
func base64Prefix(b []byte) bool {
	for _, c := range b {
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.ContainsRune("+/=_-\r\n\t ", rune(c)) {
			continue
		}
		return false
	}
	return len(b) > 8
}
func kindOf(raw []byte) string {
	if len(raw) > 4096 {
		raw = raw[:4096]
	}
	v := strings.TrimSpace(string(raw))
	scheme, _, found := strings.Cut(v, "://")
	if !found {
		return ""
	}
	scheme = strings.ToLower(scheme)
	switch scheme {
	case "vless", "vmess", "trojan", "ss", "ssr", "hysteria", "hysteria2", "hy2", "hy", "wireguard", "wg", "json", "custom", "socks", "socks5", "httpcustom", "napsternet":
		return scheme
	}
	return ""
}

type wireGuardFormat struct{}

func (wireGuardFormat) Name() string { return "wireguard-ini.v1" }
func (wireGuardFormat) Match(b []byte) bool {
	return bytes.HasPrefix(bytes.ToLower(b), []byte("[interface]"))
}
func (wireGuardFormat) Extract(ctx context.Context, path string, emit Emitter) error {
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	return emit(ctx, &contextReader{ctx, f}, "wireguard")
}

type textFormat struct{}

func (textFormat) Name() string      { return "mixed-uri-json.v1" }
func (textFormat) Match([]byte) bool { return true }

// mmap keeps large responses/file spans outside the Go heap. Readers stream exact
// spans into the encrypted vault; no response-size or per-source item quota exists.
func (textFormat) Extract(ctx context.Context, path string, emit Emitter) error {
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil {
		return e
	}
	if info.Size() == 0 {
		return nil
	}
	if int64(int(info.Size())) != info.Size() {
		return errors.New("platform_address_space")
	}
	data, e := unix.Mmap(int(f.Fd()), 0, int(info.Size()), unix.PROT_READ, unix.MAP_PRIVATE)
	if e != nil {
		return e
	}
	defer unix.Munmap(data)
	_ = unix.Madvise(data, unix.MADV_SEQUENTIAL)
	trim := bytes.TrimSpace(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf}))
	if len(trim) > 0 && (trim[0] == '{' || trim[0] == '[') && json.Valid(trim) {
		if trim[0] == '{' {
			keys, e := documentKeys(trim)
			if e != nil {
				return e
			}
			if configKeys(keys) {
				return emit(ctx, &contextReader{ctx, bytes.NewReader(data)}, "json")
			}
			return nil
		}
		return emitJSON(ctx, trim, emit)
	}
	for pos := 0; pos < len(data); {
		if e = ctx.Err(); e != nil {
			return e
		}
		end := bytes.IndexByte(data[pos:], '\n')
		if end < 0 {
			end = len(data)
		} else {
			end += pos + 1
		}
		line := data[pos:end]
		trim = bytes.TrimSpace(bytes.TrimPrefix(line, []byte{0xef, 0xbb, 0xbf}))
		if kind := kindOf(trim); kind != "" {
			if e = emit(ctx, &contextReader{ctx, bytes.NewReader(line)}, kind); e != nil {
				return e
			}
		} else if len(trim) > 0 && (trim[0] == '{' || trim[0] == '[') {
			start := pos + bytes.Index(line, trim)
			stop, e := valueEnd(ctx, data, start)
			if e != nil {
				return e
			}
			end = stop
			for end < len(data) && (data[end] == ' ' || data[end] == '\t' || data[end] == '\r') {
				end++
			}
			if end < len(data) && data[end] == '\n' {
				end++
			}
			span := data[pos:end]
			if e = emitJSON(ctx, span, emit); e != nil {
				return e
			}
		}
		pos = end
	}
	return nil
}
func valueEnd(ctx context.Context, b []byte, start int) (int, error) {
	depth := 0
	quoted, escaped := false, false
	for i := start; i < len(b); i++ {
		if i%65536 == 0 && ctx.Err() != nil {
			return 0, ctx.Err()
		}
		c := b[i]
		if quoted {
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				quoted = false
			}
			continue
		}
		switch c {
		case '"':
			quoted = true
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				return i + 1, nil
			}
		}
	}
	return 0, errors.New("invalid_json")
}
func emitJSON(ctx context.Context, raw []byte, emit Emitter) error {
	b := bytes.TrimSpace(raw)
	if !json.Valid(b) {
		return errors.New("invalid_json")
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if b[0] == '{' {
		keys, e := documentKeys(b)
		if e != nil {
			return e
		}
		if configKeys(keys) {
			return emit(ctx, &contextReader{ctx, bytes.NewReader(raw)}, "json")
		}
		return nil
	}
	if b[0] != '[' {
		return nil
	}
	for pos := 1; pos < len(b)-1; {
		for pos < len(b)-1 && (b[pos] == ' ' || b[pos] == '\t' || b[pos] == '\r' || b[pos] == '\n' || b[pos] == ',') {
			pos++
		}
		if pos >= len(b)-1 {
			break
		}
		start := pos
		if b[pos] == '{' || b[pos] == '[' {
			end, e := valueEnd(ctx, b, pos)
			if e != nil {
				return e
			}
			if e = emitJSON(ctx, b[pos:end], emit); e != nil {
				return e
			}
			pos = end
			continue
		}
		if b[pos] == '"' {
			pos++
			escaped := false
			for pos < len(b) {
				c := b[pos]
				pos++
				if escaped {
					escaped = false
					continue
				}
				if c == '\\' {
					escaped = true
					continue
				}
				if c == '"' {
					break
				}
			}
			reader := &jsonStringReader{data: b[start+1 : pos-1]}
			prefix := make([]byte, 4096)
			n, e := reader.Read(prefix)
			if e != nil && e != io.EOF {
				return e
			}
			if kind := kindOf(prefix[:n]); kind != "" {
				if e = emit(ctx, &contextReader{ctx, io.MultiReader(bytes.NewReader(prefix[:n]), reader)}, kind); e != nil {
					return e
				}
			}
			continue
		}
		for pos < len(b)-1 && b[pos] != ',' {
			pos++
		}
	}
	return nil
}

// JSON string envelopes are decoded as a stream; the full source body is also kept.
type jsonStringReader struct {
	data    []byte
	pos     int
	pending []byte
}

func (r *jsonStringReader) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		if len(r.pending) > 0 {
			k := copy(p[n:], r.pending)
			r.pending = r.pending[k:]
			n += k
			continue
		}
		if r.pos >= len(r.data) {
			if n > 0 {
				return n, nil
			}
			return 0, io.EOF
		}
		c := r.data[r.pos]
		r.pos++
		if c != '\\' {
			p[n] = c
			n++
			continue
		}
		if r.pos >= len(r.data) {
			return n, errors.New("invalid_json_string")
		}
		c = r.data[r.pos]
		r.pos++
		switch c {
		case '"', '\\', '/':
			r.pending = []byte{c}
		case 'b':
			r.pending = []byte{'\b'}
		case 'f':
			r.pending = []byte{'\f'}
		case 'n':
			r.pending = []byte{'\n'}
		case 'r':
			r.pending = []byte{'\r'}
		case 't':
			r.pending = []byte{'\t'}
		case 'u':
			if r.pos+4 > len(r.data) {
				return n, errors.New("invalid_json_string")
			}
			value, e := strconv.ParseUint(string(r.data[r.pos:r.pos+4]), 16, 16)
			if e != nil {
				return n, e
			}
			r.pos += 4
			code := rune(value)
			if utf16.IsSurrogate(code) {
				if r.pos+6 <= len(r.data) && string(r.data[r.pos:r.pos+2]) == "\\u" {
					other, e := strconv.ParseUint(string(r.data[r.pos+2:r.pos+6]), 16, 16)
					if e == nil {
						decoded := utf16.DecodeRune(code, rune(other))
						if decoded != utf8.RuneError {
							code = decoded
							r.pos += 6
						} else {
							code = utf8.RuneError
						}
					}
				} else {
					code = utf8.RuneError
				}
			}
			r.pending = utf8.AppendRune(nil, code)
		default:
			return n, errors.New("invalid_json_string")
		}
	}
	return n, nil
}
func configKeys(keys map[string]bool) bool {
	return keys["outbounds"] || keys["outbound"] || keys["proxies"] || keys["protocol"]
}
func documentKeys(b []byte) (map[string]bool, error) {
	if len(b) <= 1024*1024 {
		return strictDocument(bytes.NewReader(b))
	}
	keys := map[string]bool{}
	depth := 0
	quoted, escaped, wantKey := false, false, false
	start := -1
	for i, c := range b {
		if quoted {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == '"' {
				quoted = false
				if start >= 0 && i-start < 128 {
					var key string
					if json.Unmarshal(b[start:i+1], &key) == nil {
						keys[key] = true
					}
				}
				start = -1
			}
			continue
		}
		switch c {
		case '"':
			quoted = true
			if depth == 1 && wantKey {
				start = i
			}
		case '{', '[':
			depth++
			if depth == 1 {
				wantKey = true
			}
		case '}', ']':
			depth--
		case ':':
			if depth == 1 {
				wantKey = false
			}
		case ',':
			if depth == 1 {
				wantKey = true
			}
		}
	}
	return keys, nil
}
func strictDocument(reader io.Reader) (map[string]bool, error) {
	d := json.NewDecoder(reader)
	d.UseNumber()
	root := map[string]bool{}
	var walk func(int) error
	walk = func(depth int) error {
		token, e := d.Token()
		if e != nil {
			return e
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				keyToken, e := d.Token()
				if e != nil {
					return e
				}
				key, ok := keyToken.(string)
				if !ok || seen[key] {
					return errors.New("ambiguous_json")
				}
				seen[key] = true
				if depth == 0 {
					root[key] = true
				}
				if e = walk(depth + 1); e != nil {
					return e
				}
			}
		case '[':
			for d.More() {
				if e = walk(depth + 1); e != nil {
					return e
				}
			}
		default:
			return errors.New("invalid_json")
		}
		_, e = d.Token()
		return e
	}
	if e := walk(0); e != nil {
		return nil, e
	}
	if _, e := d.Token(); e != io.EOF {
		return nil, errors.New("extra_json")
	}
	return root, nil
}

// Fingerprint normalizes identity only. The stored bytes never pass through this result.
func Fingerprint(sample []byte, complete bool, kind, rawHash string) string {
	if !complete {
		return rawHash
	}
	value := bytes.TrimSpace(bytes.TrimPrefix(sample, []byte{0xef, 0xbb, 0xbf}))
	canonical := value
	identityKind := kind
	if kind == "hy2" {
		identityKind = "hysteria2"
	}
	if kind == "wg" {
		identityKind = "wireguard"
	}
	if kind == "vmess" {
		payload := strings.TrimPrefix(string(value), "vmess://")
		payload, _, _ = strings.Cut(payload, "#")
		if b, e := decodeBase64(payload); e == nil {
			if b, e = canonicalJSON(b, "ps"); e == nil {
				canonical = append([]byte("vmess:"), b...)
			}
		}
	} else if kind == "json" {
		if b, e := canonicalJSON(value, "remarks"); e == nil {
			canonical = b
		}
	} else if kind == "ss" {
		if b, e := ssIdentity(string(value)); e == nil {
			canonical = b
		}
	} else if strings.Contains(" vless trojan hysteria hysteria2 hy2 hy wireguard wg ", " "+kind+" ") {
		if u, e := url.Parse(string(value)); e == nil && u.Scheme != "" && u.Host != "" {
			u.Fragment = ""
			u.RawFragment = ""
			u.Scheme = identityKind
			u.Host = strings.ToLower(u.Host)
			if q, e := url.ParseQuery(u.RawQuery); e == nil {
				u.RawQuery = q.Encode()
			}
			canonical = []byte(u.String())
		}
	}
	sum := sha256.Sum256(append([]byte(identityKind+"\x00"), canonical...))
	return hex.EncodeToString(sum[:])
}
func ssIdentity(raw string) ([]byte, error) {
	u, e := url.Parse(raw)
	if e != nil {
		return nil, e
	}
	if u.User == nil {
		payload, _, _ := strings.Cut(strings.TrimPrefix(raw, "ss://"), "#")
		b, e := decodeBase64(payload)
		if e != nil {
			return nil, e
		}
		u, e = url.Parse("ss://" + string(b))
		if e != nil {
			return nil, e
		}
	}
	if u.User == nil {
		return nil, errors.New("ambiguous_ss")
	}
	method := u.User.Username()
	password, plain := u.User.Password()
	if !plain {
		b, e := decodeBase64(method)
		if e != nil {
			return nil, e
		}
		method, password, plain = strings.Cut(string(b), ":")
		if !plain {
			return nil, errors.New("ambiguous_ss")
		}
	}
	q, e := url.ParseQuery(u.RawQuery)
	if e != nil {
		return nil, e
	}
	return json.Marshal([]string{method, password, strings.ToLower(u.Host), q.Encode()})
}
func decodeBase64(s string) ([]byte, error) {
	for _, encoding := range []*base64.Encoding{base64.RawStdEncoding, base64.StdEncoding, base64.RawURLEncoding, base64.URLEncoding} {
		if b, e := encoding.DecodeString(s); e == nil {
			return b, nil
		}
	}
	return nil, errors.New("invalid_base64")
}
func canonicalJSON(b []byte, label string) ([]byte, error) {
	if _, e := strictDocument(bytes.NewReader(b)); e != nil {
		return nil, e
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.UseNumber()
	var value any
	if e := decoder.Decode(&value); e != nil {
		return nil, e
	}
	if object, ok := value.(map[string]any); ok {
		delete(object, label)
	}
	return json.Marshal(value)
}
