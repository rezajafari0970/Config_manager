package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rezajafari0970/Config_manager/internal/residential"
)

type receiptTransport func(*http.Request) (*http.Response, error)

func (f receiptTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUnsupportedInputCompletesWithIndependentCompatibilityIdentity(t *testing.T) {
	raw := []byte("unsupported-protocol://synthetic.example.invalid")
	j := residential.Job{Raw: raw, Hash: residential.Digest(raw), Policy: residential.DefaultPolicy()}
	var got residential.Completion
	calls := 0
	c := client{http: &http.Client{Transport: receiptTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path != "/residential/complete" {
			t.Fatalf("unexpected request %s", r.URL.Path)
		}
		if e := json.NewDecoder(r.Body).Decode(&got); e != nil {
			t.Fatal(e)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("{}"))}, nil
	})}}
	if e := execute(context.Background(), c, "/must-not-run", "/must-not-run", "test-worker", "test-epoch", 1, j, make([]byte, 32)); e != nil {
		t.Fatal(e)
	}
	if calls != 1 || residential.CoreID(got.Report.CoreSHA256) == "" || got.Report.CoreID != residential.CoreID(got.Report.CoreSHA256) {
		t.Fatal("unsupported receipt would be rejected as an unknown core identity", calls, got.Report.CoreID)
	}
	if len(got.Candidate) != 0 || len(got.Report.Variants) != 2 {
		t.Fatal("unsupported input synthesized a candidate")
	}
	for _, v := range got.Report.Variants {
		if v.Core != "unsupported" || v.Health != "UNKNOWN" {
			t.Fatal("compatibility failure became connection evidence", v)
		}
	}
}
