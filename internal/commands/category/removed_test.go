package category

import (
	"fmt"
	"testing"

	"github.com/suprsend/cli/mgmnt"
)

func TestRemovedCategories(t *testing.T) {
	tree := func(slugs ...string) []mgmnt.RootCategory {
		var cats []mgmnt.Category
		for _, s := range slugs {
			cats = append(cats, mgmnt.Category{Category: s})
		}
		return []mgmnt.RootCategory{{RootCategory: "transactional", Sections: []mgmnt.Section{{Categories: cats}}}}
	}
	got := RemovedCategories(tree("a", "b", "c"), tree("b", "d"))
	if fmt.Sprint(got) != "[a c]" {
		t.Errorf("removed = %v, want [a c]", got)
	}
	if got := RemovedCategories(tree("a"), tree("a", "b")); len(got) != 0 {
		t.Errorf("nothing was removed, got %v", got)
	}
}

// The service trims strings, so padding in the file is not a change.
func TestTreeFingerprintTrims(t *testing.T) {
	stored := []mgmnt.RootCategory{{RootCategory: "system", Sections: []mgmnt.Section{{Name: "Billing", Categories: []mgmnt.Category{{Category: "invoices", Name: "Invoices", Description: "paid"}}}}}}
	padded := []mgmnt.RootCategory{{RootCategory: "system", Sections: []mgmnt.Section{{Name: " Billing ", Categories: []mgmnt.Category{{Category: "invoices", Name: "Invoices", Description: " paid"}}}}}}
	if treeFingerprint(stored) != treeFingerprint(padded) {
		t.Error("padding around a name or description should not count as a change")
	}
	padded[0].Sections[0].Categories[0].Description = "unpaid"
	if treeFingerprint(stored) == treeFingerprint(padded) {
		t.Error("a changed description should differ")
	}
}
