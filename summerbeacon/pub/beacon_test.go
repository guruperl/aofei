package pub

import (
    "testing"
    "net/url"
	"strings"
	"flag"
	"github.com/golang/glog"
)

func init() {
	glog.Info("Logging configured")
	flag.Set("logtostderr", "true")
    flag.Set("stderrthreshold", "INFO")
    flag.Set("v", "2")
}

func TestBeacon(t *testing.T) {
	web, err := NewBeacon("web")
	if err != nil { t.Fatal(err) }

	err = web.GET("action=startnew")
	if web.Code != 200 {
		t.Errorf("%v", web.Code)
		t.Errorf("%v", web.Content)
	}

	pub, err := NewBeacon("pub")
	if err != nil { t.Fatal(err) }
	args := url.Values{}
	args.Set("email","publisher@example.test")
	args.Set("passwd","123x")
	err = pub.LOGIN(args)
	if err != nil { t.Fatal(err) }
	if pub.Code != 200 || pub.Content != `{"data":"challenge"}` {
		t.Errorf("%v",  pub.Code)
		t.Errorf("%v",  pub.Redirect)
		t.Errorf("%v",  pub.Content)
		t.Errorf("%v",  pub.Cookies())
	}

	pub, err = NewBeacon("pub")
	if err != nil { t.Fatal(err) }
	args = url.Values{}
	args.Set("email","publisher@example.test")
	args.Set("passwd","123")
	err = pub.LOGIN(args)
	if err != nil { t.Fatal(err) }
	x := pub.Cookies()
	if pub.Code != 200 || pub.Content != `{"data":"logged"}` || len(x)<1 || !strings.Contains(x[0].String(), "gpub=") {
		t.Errorf("%v",  pub.Code)
		t.Errorf("%v",  pub.Redirect)
		t.Errorf("%v",  pub.Content)
		t.Errorf("%v",  pub.Cookies())
	}
}
