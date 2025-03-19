package match

import (
	"testing"
)

// TestRPubPack tests the Pack and Unpack functions of RPub.
func TestRPubPack(t *testing.T) {
	rpub := RPub{PubID: 55555, SiteID: 666666, SlotID: 7777777, SizeID: 88888888}
	packed, err := rpub.PackString()
	if err != nil {
		t.Error(err)
	}
	rpub2, err := UnpackRPubString(packed)
	if err != nil {
		t.Error(err)
	}
	if rpub.PubID != rpub2.PubID {
		t.Errorf("PubID %d != %d", rpub.PubID, rpub2.PubID)
	}
	if rpub.SiteID != rpub2.SiteID {
		t.Errorf("SiteID %d != %d", rpub.SiteID, rpub2.SiteID)
	}
	if rpub.SlotID != rpub2.SlotID {
		t.Errorf("SlotID %d != %d", rpub.SlotID, rpub2.SlotID)
	}
	if rpub.SizeID != rpub2.SizeID {
		t.Errorf("SizeID %d != %d", rpub.SizeID, rpub2.SizeID)
	}
}
