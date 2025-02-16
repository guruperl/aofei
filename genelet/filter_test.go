package genelet

import (
	"bytes"
	"net/http"
	"net/url"
	"testing"
)

func TestFilter(t *testing.T) {
	configure := NewConfig(filename)

	args := make(url.Values)
	args.Set("aaa", "111")
	args.Set("bbb", "222")
	args.Set("Errorstr", Err(1001).Error())
	args.Set("Script", configure.Script)
	args.Set(configure.Role_name, "m")
	args.Set(configure.Go_uri_name, "aaa")
	args.Set("Login_name", "email")
	args.Set("Password_name", "passwd")

	topics := map[string][]string{"aliases": {"list", "subj"}, "groups": {"m", "p"}, "validate": {"aaa", "bbb"}}
	ads := map[string][]string{"groups": {"m"}, "validate": {"ccc"}}
	dashboard := map[string][]string{"groups": {"p"}, "validate": {"ccc"}}
	actions := map[string]map[string][]string{"dashboard": dashboard, "topics": topics, "ads": ads}
	r, err := http.NewRequest("GET", "http://xxx.yyy", bytes.NewBuffer([]byte("sss")))
	if err != nil {
		panic(err)
	}
	f := new(Filter)
	f.C = configure
	f.R = r
	f.Role_value = "m"
	f.Chartag_value = "json"
	f.R.Form = args
	f.Actions = actions
	f.Action = configure.Default_actions["GET"]

	a := f.Action
	// a, as := f.Get_action()
	as, _ := f.Get_all()
	if a != "dashboard" {
		t.Errorf("%v got", a)
	}
	if as["groups"][0] != "p" {
		t.Errorf("%v got", as)
	}

	args.Set("_gaction", "topics")
	validate, ok := as["validate"]
	if ok != true {
		t.Errorf("%s got", "validate not")
	}
	args.Set("_gaction", "ads")
	validate, ok = as["validate"]
	if validate[0] != "ccc" {
		t.Errorf("%s got", validate[0])
	}
}
