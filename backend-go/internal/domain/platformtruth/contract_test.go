package platformtruth

import (
	"os"
	"sort"
	"testing"
)

func TestCurrentContractIsCompleteAndUnambiguous(t *testing.T) {
	c := CurrentContract()
	if c.Version == "" || c.Direction == "" || len(c.BoundaryRules) == 0 || len(c.Unknowns) == 0 || len(c.ClaimLevels) == 0 || len(c.SystemBoundaries) == 0 || len(c.ObjectIdentityRules) == 0 || len(c.SourceRules) == 0 {
		t.Fatal("contract metadata and evidence limits are required")
	}
	wantTruth := []string{"actual", "quoted", "estimated", "unknown", "mock", "inferred"}
	if len(c.TruthLevels) != len(wantTruth) {
		t.Fatalf("truth level count = %d, want %d", len(c.TruthLevels), len(wantTruth))
	}
	for i, want := range wantTruth {
		if c.TruthLevels[i].Code != want || c.TruthLevels[i].Meaning == "" {
			t.Fatalf("truth level %d = %#v, want %q with meaning", i, c.TruthLevels[i], want)
		}
	}
	seen := map[string]bool{}
	for _, domain := range c.DomainDispositions {
		if domain.ID == "" || domain.Name == "" || domain.System == "" || domain.Disposition == "" || domain.Reason == "" || domain.Evidence == "" || domain.XiaoQ == "" || domain.OwnerScope != "single_owner" || domain.Risk == "" {
			t.Fatalf("incomplete domain disposition: %#v", domain)
		}
		if seen[domain.ID] {
			t.Fatalf("duplicate domain disposition: %s", domain.ID)
		}
		seen[domain.ID] = true
	}
	for _, required := range []string{"demandcase", "experiment", "sourcing1688", "order", "settlement", "profit", "decision", "xiaoq"} {
		if !seen[required] {
			t.Fatalf("missing core domain disposition: %s", required)
		}
	}
}

func TestDomainRegistryCoversEveryDomainPackage(t *testing.T) {
	entries, err := os.ReadDir("..")
	if err != nil {
		t.Fatal(err)
	}
	var directories []string
	for _, entry := range entries {
		if entry.IsDir() {
			directories = append(directories, entry.Name())
		}
	}
	registered := make([]string, 0, len(CurrentContract().DomainDispositions))
	for _, item := range CurrentContract().DomainDispositions {
		registered = append(registered, item.ID)
	}
	sort.Strings(directories)
	sort.Strings(registered)
	if len(directories) != len(registered) {
		t.Fatalf("domain registry has %d entries, filesystem has %d\nregistry=%v\nfilesystem=%v", len(registered), len(directories), registered, directories)
	}
	for i := range directories {
		if directories[i] != registered[i] {
			t.Fatalf("domain registry mismatch at %d: got %q want %q", i, registered[i], directories[i])
		}
	}
}

func TestDomainRegistryUsesClosedVocabularies(t *testing.T) {
	validSystems := map[string]bool{"fact": true, "decision": true, "collaboration": true, "kernel": true, "support": true, "frozen": true}
	validActions := map[string]bool{"reuse": true, "adapt": true, "migrate": true, "rebuild": true, "freeze": true, "delete": true}
	validEvidence := map[string]bool{"implemented": true, "planned": true, "mock": true, "superseded": true}
	validXiaoQ := map[string]bool{"active": true, "deferred": true, "not_applicable": true}
	validRisk := map[string]bool{"low": true, "medium": true, "high": true}
	for _, item := range CurrentContract().DomainDispositions {
		if !validSystems[item.System] || !validActions[item.Disposition] || !validEvidence[item.Evidence] || !validXiaoQ[item.XiaoQ] || !validRisk[item.Risk] {
			t.Fatalf("domain %s uses value outside closed contract: %#v", item.ID, item)
		}
		if item.System == "frozen" && item.Disposition != "freeze" && item.Disposition != "delete" && item.Disposition != "migrate" {
			t.Fatalf("frozen domain %s cannot use disposition %s", item.ID, item.Disposition)
		}
	}
}
