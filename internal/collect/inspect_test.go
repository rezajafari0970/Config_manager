package collect

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestContentDiagnosisDoesNotRepair(t *testing.T) {
	for _, v := range []struct{ kind, raw, want string }{{"vless", "vless://user@vpn.example:443#original", "valid"}, {"vless", "vless://missing-host", "invalid"}, {"json", `{"outbounds":[{"protocol":"vless","settings":{}}],"stats":[]}`, "invalid"}, {"json", `{"outbounds":[{"protocol":"vless","settings":{}}],"stats":{}}`, "valid"}, {"custom", "custom://opaque", "unknown"}, {"json", `{"outbounds":[]}`, "invalid"}} {
		if got := InspectConfig([]byte(v.raw), true, v.kind); got != v.want {
			t.Fatal(v.kind, got, v.want)
		}
	}
	for _, v := range []struct{ raw, want string }{{" \r\n", "empty"}, {"[]", "empty"}, {`{"error":"no data"}`, "error_response"}, {"<html>error</html>", "error_response"}, {"unrelated text", "unrecognized"}} {
		path := filepath.Join(t.TempDir(), "raw")
		os.WriteFile(path, []byte(v.raw), 0600)
		if got := ContentState(path, 0, 0, 0); got != v.want {
			t.Fatal(got, v.want)
		}
	}
}

func TestMalformedSiblingDoesNotDiscardGoodExactSpans(t *testing.T) {
	first := "vless://one@vpn.example:443#exact\r\n"
	last := "trojan://Secret@vpn.example:443#two\r\n"
	raw := first + "{\"outbounds\": broken}\r\n" + last
	path := filepath.Join(t.TempDir(), "raw")
	os.WriteFile(path, []byte(raw), 0600)
	got := []string{}
	e := NewRegistry().Extract(context.Background(), path, func(ctx context.Context, r io.Reader, kind string) error {
		b, e := io.ReadAll(r)
		got = append(got, string(b))
		return e
	})
	if !IsContentError(e) || len(got) != 2 || got[0] != first || got[1] != last {
		t.Fatal("sibling isolation or raw bytes", e, got)
	}
}

func TestJSONTextEnvelopePreservesPayload(t *testing.T) {
	payload := "{ \"outbounds\" : [{\"protocol\":\"vless\",\"settings\":{}}], \"number\":9007199254740993 }\r\n"
	encoded, _ := json.Marshal(payload)
	for _, raw := range []string{string(encoded), "[" + string(encoded) + "]"} {
		path := filepath.Join(t.TempDir(), "raw")
		os.WriteFile(path, []byte(raw), 0600)
		got := ""
		e := NewRegistry().Extract(context.Background(), path, func(ctx context.Context, r io.Reader, kind string) error {
			if kind != "json" {
				t.Fatal(kind)
			}
			b, e := io.ReadAll(r)
			got = string(b)
			return e
		})
		if e != nil || got != payload {
			t.Fatal("envelope changed config", e)
		}
	}
}

func TestHTTPUsesConfiguredDeadlineBeyondOldHeaderLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("15-second timeout regression")
	}
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(15200 * time.Millisecond):
			w.Write([]byte("raw"))
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	address := strings.TrimPrefix(server.URL, "http://")
	driver := HTTPDriver{Dial: func(ctx context.Context, network, ignored string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, address)
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	response, e := driver.Fetch(ctx, Job{URL: "http://1.1.1.1/late"})
	if e != nil {
		t.Fatal("hidden transport timeout overrides panel", e)
	}
	body, e := io.ReadAll(response.Body)
	response.Body.Close()
	if e != nil || string(body) != "raw" {
		t.Fatal(e)
	}
	short, cancelShort := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancelShort()
	start := time.Now()
	if response, e = driver.Fetch(short, Job{URL: "http://1.1.1.1/short"}); e == nil {
		response.Body.Close()
		t.Fatal("timeout ignored")
	}
	if time.Since(start) > time.Second {
		t.Fatal("request cancellation too slow")
	}
}

type panicInspector struct{}

func (panicInspector) Kind() string                { return "custom" }
func (panicInspector) Inspect([]byte, bool) string { panic("synthetic inspector panic") }
func TestInspectorFailureIsLocal(t *testing.T) {
	r := NewInspectionRegistry()
	r.Register(panicInspector{})
	if r.Inspect([]byte("custom://raw"), true, "custom") != "unknown" {
		t.Fatal("failed inspector accepted input")
	}
	if r.Inspect([]byte("vless://user@vpn.example:443"), true, "vless") != "valid" {
		t.Fatal("one inspector broke another")
	}
}
