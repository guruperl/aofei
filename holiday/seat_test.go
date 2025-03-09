package holiday

import (
	"testing"
	"time"
	"encoding/json"
)

func TestSeatBase(t *testing.T) {
	incoming, err := NewIncoming([]byte(ADs))
	if err != nil {
		t.Fatal(err)
	}
	adunits := incoming.Adunits
	c, err := NewConfig("sample.json")
    if err != nil { t.Fatal(err) }

    user := new(User)
	user.UserId = 111

	sb := &SeatBase{user, c, time.Date(2009, 1, 1, 12, 0, 0, 0, time.UTC), adunits[0]}
	psa := sb.PSABid()

	result := `{"seat":"PSA19661050","bid":[{"id":"PSA_BID19661050","purl":"click_psa","burl":"click_psa","lurl":"click_psa","exp":10,"mid":"PSA_AD19661050","media":{"ad":{"id":"PSA_AD19661050","adomain":["pzcom.com"],"iurl":"display_psa","init":1230811200000000000,"display":{"mime":"image/png","ctype":3,"w":300,"h":250,"priv":"http://www.pzcom.com/psa.html","adm":"\u003ca href='click_psa'\u003e\u003cimg src='display_psa\u003e\u003c/a\u003e","banner":{"img":"display_psa","link":{"url":"click_psa","urlfb":"click_psa","trkr":["click_psa"]}}}}}}]}`
	json_str, _ := json.Marshal(psa)

	if string(json_str) != result {
		t.Errorf("%s", result)
		t.Errorf("%s", string(json_str))
	}
}
