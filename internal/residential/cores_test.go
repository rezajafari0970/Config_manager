package residential

import (
	"encoding/json"
	"os"
	"testing"
)

func TestOfficialCorePinsMatchSupervisor(t *testing.T) {
	b, err := os.ReadFile("../../scripts/residential-cores.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Cores []struct {
			ID  string `json:"id"`
			SHA string `json:"sha256"`
		} `json:"cores"`
	}
	if err = json.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Cores) != 4 {
		t.Fatal("expected three official Xray pins and one native sing-box pin")
	}
	seen := map[string]bool{}
	for _, c := range manifest.Cores {
		if seen[c.ID] {
			t.Fatal("duplicate pin", c.ID)
		}
		seen[c.ID] = true
		if CoreID(c.SHA) != c.ID {
			t.Fatal("core registry drift", c.ID)
		}
	}
	for _, id := range []string{"xray-26.11.9", "xray-26.7.28", "xray-26.5.9", SingBoxID} {
		if !seen[id] {
			t.Fatal("missing driver", id)
		}
	}
	if CoreID("arbitrary") != "" {
		t.Fatal("unknown core admitted")
	}
}
