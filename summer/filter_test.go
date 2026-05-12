package summer

import (
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/genelet/winter/genelet"
)

func TestFilter(t *testing.T) {
	filter := new(Filter)
	comp := genelet.NewComponent("./address/component.json")
	filter.Initialize(comp)
	filter.Action = "insert"
	filter.Component = "address"

	filter.Base.C = testSummerConfig(t)
	jar := GetJar()
	filter.Base.R = GetNewRequest("http://www.u2link.com", jar)
	filter.Base.W = httptest.NewRecorder()
	if err := filter.Base.R.ParseForm(); err != nil {
		t.Errorf("%v\n", err)
	}
	filter.R.Form.Set("_gtime", strconv.FormatInt(time.Now().Unix(), 10))
	if err := filter.Preset(); err != nil {
		t.Errorf("%v\n", err)
	}
	if filter.R.Form.Get("ip") != "210.51.200.123" {
		t.Errorf("%v\n", filter.R.Form)
	}
}
