package match

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
)

// readSampleZBidReqest read sample json and parse it int openrtb2 bid request object
func readSampleBidRequest(filename string) (*openrtb2.BidRequest, error) {
	// read sample bid request
	fh, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	b := new(openrtb2.BidRequest)
	if err := json.NewDecoder(fh).Decode(b); err != nil {
		return nil, err
	}
	return b, nil
}

// TestConvertNative tests the parseNative function
func TestConvertNative(t *testing.T) {
	// read sample bid request
	bidRequest, err := readSampleBidRequest("sample_bid.json")
	if err != nil {
		t.Fatalf("failed to read sample bid request: %v", err)
	}
	native, err := parseNative(bidRequest.Imp[0].Native)
	if err != nil {
		t.Fatalf("failed to convert native: %v", err)
	}

	for i, item := range native.Native.Assets {
		if i == 1 && (item.Img == nil || item.Img.WMin != 64) {
			t.Errorf("image: %#v", item.Img)
		}
	}
}

// TestConvertAdImp tests the NewAdImp function
func TestConvertAdImp(t *testing.T) {
	// read sample bid request
	bidRequest, err := readSampleBidRequest("sample_bid.json")
	if err != nil {
		t.Fatalf("failed to read sample bid request: %v", err)
	}

	adImp, err := NewAdImp(bidRequest)
	if err != nil {
		t.Fatalf("failed to convert ad imp: %v", err)
	}
	if adImp[0].Banner != nil || adImp[0].Video != nil ||
		adImp[0].RPub.PubID != 1 || adImp[0].RPub.SiteID != 1 || adImp[0].RPub.SlotID != 1 {
		t.Errorf("adImp: %#v", adImp[0])
	}
	for i, item := range adImp[0].Native.Native.Assets {
		if i == 1 && (item.Img == nil || item.Img.WMin != 64) {
			t.Errorf("image: %#v", item.Img)
		}
	}
}
