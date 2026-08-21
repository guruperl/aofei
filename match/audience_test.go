package match

import (
	"testing"

	"github.com/guruperl/aofei/acl"
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
