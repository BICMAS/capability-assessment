package roleprofile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type evalExample struct {
	Name              string          `json:"name"`
	Taxonomy          []Skill         `json:"taxonomy"`
	Output            json.RawMessage `json:"output"`
	ExpectError       bool            `json:"expectError"`
	ExpectAccepted    int             `json:"expectAccepted"`
	ExpectUncertainty int             `json:"expectUncertainty"`
	ExpectUnsupported int             `json:"expectUnsupported"`
}

func TestRoleProfileEvaluationHarness(t *testing.T) {
	matches, err := filepath.Glob("testdata/eval/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatal("no evaluation examples")
	}
	for _, path := range matches {
		t.Run(filepath.Base(path), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var example evalExample
			if err := json.Unmarshal(raw, &example); err != nil {
				t.Fatal(err)
			}
			accepted, unsupported, uncertainty, err := validateGenerated(example.Output, example.Taxonomy)
			if example.ExpectError {
				if err == nil {
					t.Fatal("expected validation error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(accepted) != example.ExpectAccepted || len(unsupported) != example.ExpectUnsupported || len(uncertainty) != example.ExpectUncertainty {
				t.Fatalf("accepted %d unsupported %d uncertainty %v", len(accepted), len(unsupported), uncertainty)
			}
		})
	}
}
