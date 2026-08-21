package demo

import (
	"reflect"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestGetLangsHandlesMissingDevice(t *testing.T) {
	request := &openrtb2.BidRequest{}
	if got := getLangs(request); got != nil {
		t.Fatalf("languages = %#v, want nil", got)
	}
	if got := getLangs(nil); got != nil {
		t.Fatalf("nil request languages = %#v, want nil", got)
	}
}

func TestGetLangsUsesDeviceLanguages(t *testing.T) {
	request := &openrtb2.BidRequest{Device: &openrtb2.Device{Language: "zh", LangB: "cmn"}}
	if got, want := getLangs(request), []string{"zh", "cmn"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("languages = %#v, want %#v", got, want)
	}
}
