package googlertb

import (
    "testing"
	"io/ioutil"
	"encoding/json"
	"github.com/golang/protobuf/proto"
	openrtb2 "github.com/prebid/openrtb/v20/openrtb2"
)

func TestConvert(t *testing.T) {
	content, err := ioutil.ReadFile("./request.json")
	if err != nil { panic(err) }
	parsed := &openrtb2.BidRequest{}
	err = json.Unmarshal(content, parsed)
	if err != nil { panic(err) }

	pb_request := ConvertBidRequest(parsed)
	data, err := proto.Marshal(pb_request)
	if err != nil { panic(err) }
	request := &BidRequest{}
	err = proto.Unmarshal(data, request)
	if err != nil {
		t.Errorf("%v", pb_request)
		t.Errorf("%v", request)
	}
}
