package match

import (
	"testing"

	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/demo"
)

func TestAudienceHasNilInputsFailSafely(t *testing.T) {
	var nilAudience *Audience
	if nilAudience.Has(&Attribute{}) {
		t.Fatal("nil audience matched")
	}
	if (&Audience{}).Has(nil) {
		t.Fatal("nil attribute matched")
	}
	if !(&Audience{}).Has(&Attribute{}) {
		t.Fatal("empty audience should remain a wildcard for a valid attribute")
	}
	if (&Audience{ACLAudience: &acl.ACLAudience{}}).Has(&Attribute{}) {
		t.Fatal("ACL audience matched an attribute without ACL data")
	}
}

func TestAudienceHasMissingDemographics(t *testing.T) {
	attr := &Attribute{}
	if !(&Audience{DemoAudience: &demo.DemoAudience{}}).Has(attr) {
		t.Fatal("empty demographic audience should match missing demographics")
	}

	tests := []struct {
		name     string
		audience *demo.DemoAudience
	}{
		{name: "gender", audience: &demo.DemoAudience{Genders: 1 << demo.GENDERM}},
		{name: "age", audience: &demo.DemoAudience{Yobs: 1 << demo.YOB1990}},
		{name: "language", audience: &demo.DemoAudience{Languages: 1 << demo.LanguageEN}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if (&Audience{DemoAudience: test.audience}).Has(attr) {
				t.Fatal("configured demographic targeting matched missing request demographics")
			}
		})
	}
}
