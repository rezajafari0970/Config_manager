package collect

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func extractTest(t *testing.T, data []byte) ([][]byte, []string, error) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "response")
	os.WriteFile(p, data, 0600)
	var values [][]byte
	var kinds []string
	e := NewRegistry().Extract(context.Background(), p, func(ctx context.Context, r io.Reader, kind string) error {
		b, e := io.ReadAll(r)
		values = append(values, b)
		kinds = append(kinds, kind)
		return e
	})
	return values, kinds, e
}
func TestFormatsPreserveExactConfigBytes(t *testing.T) {
	uri := []byte("vless://synthetic@vpn.example:443?security=tls#Keep%20This\r\n")
	values, kinds, e := extractTest(t, append(append([]byte{}, uri...), uri...))
	if e != nil || len(values) != 2 || !bytes.Equal(values[0], uri) || kinds[0] != "vless" {
		t.Fatal("URI bytes changed", e)
	}
	raw := []byte(" \r\n{ \"outbounds\" : [{\"protocol\":\"vless\",\"settings\":{\"n\":9007199254740993}}], \"unknown\" : [ 1, 2 ] }\n")
	values, _, e = extractTest(t, raw)
	if e != nil || len(values) != 1 || !bytes.Equal(values[0], raw) {
		t.Fatal("JSON bytes changed", e)
	}
	values, _, e = extractTest(t, []byte("["+string(bytes.TrimSpace(raw))+", "+string(bytes.TrimSpace(raw))+"]"))
	if e != nil || len(values) != 2 || !bytes.Equal(values[0], bytes.TrimSpace(raw)) {
		t.Fatal("JSON array bytes changed", e)
	}
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		values, _, e = extractTest(t, []byte(encoding.EncodeToString(uri)))
		if e != nil || len(values) != 1 || !bytes.Equal(values[0], uri) {
			t.Fatal("subscription envelope changed payload", e)
		}
	}
	wg := []byte("[Interface]\r\nPrivateKey = SYNTHETIC\r\n[Peer]\r\nEndpoint = [2001:db8::1]:1234\r\n")
	values, kinds, e = extractTest(t, wg)
	if e != nil || len(values) != 1 || kinds[0] != "wireguard" || !bytes.Equal(values[0], wg) {
		t.Fatal("WireGuard altered", e)
	}
	values, _, e = extractTest(t, []byte(`{"error":"upstream unavailable"}`))
	if e != nil || len(values) != 0 {
		t.Fatal("error JSON classified as config")
	}
	_, _, e = extractTest(t, []byte(`{"outbounds":[],"outbounds":[1]}`))
	if e == nil {
		t.Fatal("ambiguous JSON accepted")
	}
}
func fp(raw, kind string) string {
	sum := sha256.Sum256([]byte(raw))
	return Fingerprint([]byte(raw), true, kind, hex.EncodeToString(sum[:]))
}
func TestMixedURIsAndCustomJSON(t *testing.T) {
	uri := []byte("vless://synthetic@vpn.example:443#exact\r\n")
	raw := []byte(`{ "outbounds": [] }` + "\r\n")
	for _, body := range [][]byte{append(append([]byte{}, uri...), raw...), append(append([]byte{}, raw...), uri...)} {
		items, _, e := extractTest(t, body)
		if e != nil || len(items) != 2 {
			t.Fatal("mixed extraction failed", e, len(items))
		}
		if !bytes.Equal(items[0], body[:len(items[0])]) {
			t.Fatal("mixed bytes changed")
		}
	}
}
func TestIdentityNeverMergesCredentials(t *testing.T) {
	a := "vless://SecretA@VPN.example:443?type=tcp&security=tls#one"
	b := "vless://SecretA@vpn.example:443?security=tls&type=tcp#two"
	if fp(a, "vless") != fp(b, "vless") {
		t.Fatal("same identity not deduplicated")
	}
	if fp(a, "vless") == fp(strings.ReplaceAll(a, "SecretA", "secreta"), "vless") {
		t.Fatal("credential case lost")
	}
	if fp("ss://AbCdEf#one", "ss") == fp("ss://abcdef#one", "ss") {
		t.Fatal("SS encoded credential case lost")
	}
	if fp(`{"outbounds":[],"remarks":"one"}`, "json") != fp("{ \"remarks\":\"two\", \"outbounds\" : [] }", "json") {
		t.Fatal("JSON identity")
	}
	if fp(`{"outbounds":[],"x":1,"x":2}`, "json") == fp(`{"outbounds":[],"x":2}`, "json") {
		t.Fatal("duplicate key identity collapsed")
	}
}
func TestHTTPSSRFAndPerJobTransport(t *testing.T) {
	for _, ip := range []string{"127.0.0.1", "::1", "::ffff:127.0.0.1", "10.1.2.3", "169.254.169.254", "100.64.1.2", "192.0.2.2", "fc00::1", "64:ff9b:1::a00:1", "64:ff9b::7f00:1"} {
		if PublicIP(netip.MustParseAddr(ip)) {
			t.Fatal("private address accepted", ip)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "http://127.0.0.1/private", 302)
		case "/etag":
			if r.Header.Get("If-None-Match") == `"fixture"` {
				w.WriteHeader(304)
				return
			}
			w.Header().Set("ETag", `"fixture"`)
			w.Write([]byte("config"))
		default:
			if r.Header.Get("Cookie") != "" {
				t.Error("cookie leaked")
			}
			w.Header().Set("Set-Cookie", "private=fixture")
			w.Write([]byte("body\r\n"))
		}
	}))
	defer server.Close()
	address := strings.TrimPrefix(server.URL, "http://")
	calls := 0
	driver := HTTPDriver{Dial: func(ctx context.Context, network, target string) (net.Conn, error) {
		calls++
		if target != "1.1.1.1:80" {
			t.Error("unexpected dial", target)
		}
		return (&net.Dialer{}).DialContext(ctx, network, address)
	}}
	for i := 0; i < 2; i++ {
		response, e := driver.Fetch(context.Background(), Job{URL: "http://1.1.1.1/"})
		if e != nil {
			t.Fatal(e)
		}
		body, e := io.ReadAll(response.Body)
		response.Body.Close()
		if e != nil || string(body) != "body\r\n" {
			t.Fatal("response changed", e)
		}
	}
	if _, e := driver.Fetch(context.Background(), Job{URL: "http://127.0.0.1/"}); e == nil {
		t.Fatal("private direct request")
	}
	before := calls
	if _, e := driver.Fetch(context.Background(), Job{URL: "http://1.1.1.1/redirect"}); e == nil {
		t.Fatal("private redirect followed")
	}
	if calls != before+1 {
		t.Fatal("redirect dial escaped policy")
	}
	response, e := driver.Fetch(context.Background(), Job{URL: "http://1.1.1.1/etag", ETag: `"fixture"`})
	if e != nil {
		t.Fatal(e)
	}
	response.Body.Close()
	if response.Status != 304 {
		t.Fatal("conditional request")
	}
}
