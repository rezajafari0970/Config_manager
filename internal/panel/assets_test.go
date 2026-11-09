package panel

import (
	"io/fs"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestBrowserModuleImportsAreServed(t *testing.T) {
	app, e := NewServer(testStore(t), "https://panel.example")
	if e != nil {
		t.Fatal(e)
	}
	h := app.Handler()
	imports := regexp.MustCompile(`(?:from\s+|import\s*\(\s*)['"]\./([^'"]+)['"]`)
	files, e := fs.Glob(assets, "web/*.js")
	if e != nil {
		t.Fatal(e)
	}
	for _, file := range files {
		b, e := assets.ReadFile(file)
		if e != nil {
			t.Fatal(e)
		}
		for _, match := range imports.FindAllStringSubmatch(string(b), -1) {
			r := httptest.NewRequest("GET", app.origin+"/assets/"+match[1], nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 200 || !strings.Contains(w.Header().Get("Content-Type"), "javascript") {
				t.Fatalf("%s imports unavailable module %s: %d", file, match[1], w.Code)
			}
		}
	}
}
