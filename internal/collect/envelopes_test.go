package collect

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMixedBase64JSONExactPayloads(t *testing.T) {
	uri := "vless://synthetic@vpn.example:443?security=tls#exact\r\n"
	other := "trojan://DifferentSecret@vpn.example:8443#other\n"
	// One full custom config, not one record per child outbound.
	custom := "{ \"outbounds\" : [{\"tag\":\"a\",\"protocol\":\"vless\"},{\"tag\":\"b\",\"protocol\":\"hysteria\"}], \"remarks\":\"keep\" }\r\n"
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		t.Run(encoding.EncodeToString([]byte("??>")), func(t *testing.T) {
			a := encoding.EncodeToString([]byte(uri))
			b := encoding.EncodeToString([]byte(other))
			// Independent encoded lines (padded or not), JSON on both sides,
			// and a direct copy of the same URI must preserve exact bytes.
			body := custom + a + "\n" + b + "\n" + custom + uri
			items, kinds, err := extractTest(t, []byte(body))
			want := []string{custom, uri, other, custom, uri}
			if err != nil || len(items) != len(want) {
				t.Fatalf("items=%d err=%v", len(items), err)
			}
			for i, raw := range want {
				if string(items[i]) != raw {
					t.Fatalf("payload %d changed", i)
				}
			}
			if kinds[0] != "json" || kinds[1] != "vless" || kinds[2] != "trojan" {
				t.Fatal(kinds)
			}
			if fp(string(items[1]), kinds[1]) != fp(string(items[4]), kinds[4]) {
				t.Fatal("encoded and plain copies bypass identity dedup")
			}
		})
	}
}

func TestWrappedAndNestedBase64PreservesDecodedBytes(t *testing.T) {
	raw := "{\n \"outbounds\": [{\"protocol\":\"vless\",\"settings\":{}}],\n \"remarks\":\"large " + strings.Repeat("x", 8192) + "\"\n}\r\n"
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		encoded := encoding.EncodeToString([]byte(raw))
		// Non-quartet width plus a short last line: line-by-line decoding is wrong.
		var wrapped strings.Builder
		for len(encoded) > 61 {
			wrapped.WriteString(encoded[:61] + "\r\n")
			encoded = encoded[61:]
		}
		wrapped.WriteString(encoded)
		for _, body := range []string{
			"\ufeff" + wrapped.String(),
			"vless://before@vpn.example:443\n" + wrapped.String() + "\n" + "trojan://after@vpn.example:443\n",
			encoding.EncodeToString([]byte(wrapped.String())),
		} {
			items, _, err := extractTest(t, []byte(body))
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, item := range items {
				found = found || string(item) == raw
			}
			if !found {
				t.Fatal("wrapped/nested JSON lost or split")
			}
			if strings.HasPrefix(body, "vless:") && len(items) != 3 {
				t.Fatal("mixed siblings lost")
			}
		}
	}
	// Base64 URL alphabet is actually exercised, not only its encoder name.
	uri := []byte("vless://synthetic@vpn.example:443#\u083e\u083f\r\n")
	encoded := base64.URLEncoding.EncodeToString(uri)
	if !strings.ContainsAny(encoded, "-_") {
		t.Fatal("fixture must contain URL alphabet")
	}
	items, _, err := extractTest(t, []byte(encoded))
	if err != nil || len(items) != 1 || !bytes.Equal(items[0], uri) {
		t.Fatal("URL alphabet decode", err)
	}
}

func TestShortWrappedEnvelopeAndDeepNesting(t *testing.T) {
	raw := "vless://synthetic@vpn.example:443\r\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))
	var wrapped strings.Builder
	for i := 0; i < len(encoded); i += 4 {
		wrapped.WriteString(encoded[i:min(i+4, len(encoded))] + "\n\n")
	}
	items, _, err := extractTest(t, []byte(wrapped.String()))
	if err != nil || len(items) != 1 || string(items[0]) != raw {
		t.Fatal("short wrapping lost", err)
	}
	for i := 0; i < maxEnvelopeDepth+2; i++ {
		encoded = base64.StdEncoding.EncodeToString([]byte(encoded))
	}
	items, _, err = extractTest(t, []byte(encoded))
	if len(items) != 0 || (err != nil && !IsContentError(err)) {
		t.Fatal("nested envelope guard", err)
	}
}

func TestBadEnvelopeDoesNotPoisonSiblings(t *testing.T) {
	uri := "vless://synthetic@vpn.example:443\r\n"
	bad := base64.StdEncoding.EncodeToString([]byte(uri)) + "==" // invalid padding
	good := "{\"outbounds\":[{\"protocol\":\"trojan\"}]}\n"
	items, _, err := extractTest(t, []byte(good+bad+"\n"+uri))
	if !IsContentError(err) || len(items) != 2 || string(items[0]) != good || string(items[1]) != uri {
		t.Fatalf("bad envelope isolation: count=%d err=%v", len(items), err)
	}
	for _, body := range []string{"UnrelatedNotice\n", base64.StdEncoding.EncodeToString([]byte("<html>error</html>")), base64.StdEncoding.EncodeToString([]byte("{\"error\":\"unavailable\"}"))} {
		items, _, err = extractTest(t, []byte(body))
		if err != nil || len(items) != 0 {
			t.Fatal("ordinary text became config", err)
		}
	}
}

func TestEnvelopesCancellationAndEmitterFailure(t *testing.T) {
	uri := "vless://synthetic@vpn.example:443\n"
	path := filepath.Join(t.TempDir(), "feed.txt")
	os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString([]byte(uri))), 0600)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := NewRegistry().Extract(ctx, path, func(context.Context, io.Reader, string) error {
		t.Fatal("emitted after cancellation")
		return nil
	}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	failure := errors.New("synthetic storage failure")
	if err := NewRegistry().Extract(context.Background(), path, func(context.Context, io.Reader, string) error { return failure }); !errors.Is(err, failure) {
		t.Fatal("emitter failure swallowed", err)
	}
}

func TestTXTPathIsCaseSensitiveAndMIMEIndependent(t *testing.T) {
	raw := "vless://synthetic@vpn.example:443\r\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Api/Example/exo.txt" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=exo.txt")
		w.Write([]byte(raw))
	}))
	defer server.Close()
	driver := HTTPDriver{Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, strings.TrimPrefix(server.URL, "http://"))
	}}
	for _, path := range []string{"Exo.txt", "exo.txt"} {
		response, err := driver.Fetch(context.Background(), Job{URL: "http://1.1.1.1/Api/Example/" + path})
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if path == "Exo.txt" {
			if response.Status != 404 {
				t.Fatal("case was guessed/rewritten")
			}
			continue
		}
		items, _, err := extractTest(t, body)
		if response.Status != 200 || err != nil || len(items) != 1 || string(items[0]) != raw {
			t.Fatal("TXT/attachment extraction failed", response.Status, err)
		}
	}
}

func TestEncodedSubscriptionMayStartWithComment(t *testing.T) {
	raw := "# subscription heading\n\nvless://synthetic@vpn.example:443\r\n"
	for _, prefix := range []string{"", "{\"outbounds\":[]}\n"} {
		items, _, err := extractTest(t, []byte(prefix+base64.StdEncoding.EncodeToString([]byte(raw))))
		if err != nil || len(items) == 0 || string(items[len(items)-1]) != "vless://synthetic@vpn.example:443\r\n" {
			t.Fatal("encoded comment lost payload", err)
		}
	}
}

func TestWrappedJSONNeverSplitsAtChildObjectBoundary(t *testing.T) {
	prefix := "{\"outbounds\":["
	prefix += strings.Repeat(" ", (3-len(prefix)%3)%3)
	child := "{\"protocol\":\"vless\",\"settings\":{}}"
	child += strings.Repeat(" ", (3-len(child)%3)%3)
	raw := prefix + child + "]}"
	next := "trojan://synthetic@vpn.example:443\n"
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding} {
		body := enc.EncodeToString([]byte(prefix)) + "\n" + enc.EncodeToString([]byte(child)) + "\n" + enc.EncodeToString([]byte("]}")) + "\n" + enc.EncodeToString([]byte(next))
		items, _, err := extractTest(t, []byte(body))
		if err != nil || len(items) != 2 || string(items[0]) != raw || string(items[1]) != next {
			t.Fatalf("child promoted or parent lost: count=%d err=%v", len(items), err)
		}
	}
}
