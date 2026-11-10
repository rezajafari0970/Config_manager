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
	if len(manifest.Cores) != 3 {
		t.Fatal("expected three explicit official pins")
	}
	for _, c := range manifest.Cores {
		if CoreID(c.SHA) != c.ID {
			t.Fatal("core registry drift", c.ID)
		}
	}
	if CoreID("arbitrary") != "" {
		t.Fatal("unknown core admitted")
	}
}
