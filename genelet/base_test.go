package genelet

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func new_Base(c *Config, rv string, cv string, w http.ResponseWriter, r *http.Request) *Base {
	return &Base{C: c, W: w, R: r, Role_value: rv, Chartag_value: cv}
}

func TestBase(t *testing.T) {
	configure, err := NewConfig(filename)
	if err != nil {
		t.Fatal(err)
	}
	//base := new_Base(configure, "", "", nil, nil)

	d := Digest64("1234567", fmt.Sprint("root", "script", "tmpl"))
	if d != "v7AeDE9+Z6iXuxheHt2fEfEpejI=" {
		t.Errorf("digest64 wrong %s", d)
	}

	req, err := http.NewRequest("GET", "http://example.com/foo", nil)
	if err != nil {
		log.Fatal(err)
	}
	w := httptest.NewRecorder()
	b := new_Base(configure, "", "", w, req)
	b.Set_cookie("c", "v", 100)
	h := w.Header()
	matched, err := regexp.MatchString("^c=v; Path=/; Domain=example.com; Expires=", h.Get("Set-Cookie"))
	if err != nil || !matched {
		t.Errorf("%s gotten", h.Get("Set-Cookie"))
	}

	req, err = http.NewRequest("GET", "http://example.com/foo", nil)
	if err != nil {
		log.Fatal(err)
	}
	w = httptest.NewRecorder()
	b = new_Base(configure, "", "", w, req)
	b.Set_cookie_expire("c")
	b.Send_page("okokok")
	h = w.Header()
	matched, err = regexp.MatchString("^c=0; Path=/; Domain=example.com; Expires=", h.Get("Set-Cookie"))
	if err != nil || !matched {
		t.Errorf("%s gotten", h.Get("Set-Cookie"))
	}
	if w.Body.String() != "okokok" {
		t.Errorf("%s wanted", "okokok")
	}

	err = req.ParseForm()
	if err != nil {
		panic(err)
	}
	req.Form.Add("go_uri", "bbAAA/BBB/CCC?action=x")
	err = b.Fulfill()
	if err == nil || err.(Gerror).Errstr != "Redirected role not found" {
		t.Errorf("%d code for %s\n", err.(Gerror).Code, b.Role_value)
	}
	b.C.Script = "/bb"
	req.Form.Set("go_uri", "/bb/m/BBB/CCC?action=x")
	err = b.Fulfill()
	if err != nil {
		t.Errorf("%d code for %s\n", err.(Gerror).Code, b.Role_value)
	}

	if b.Role_value != "m" {
		t.Errorf("role is %s", b.Role_value)
	}
	if b.Chartag_value != "BBB" {
		t.Errorf("chartag is %s", b.Chartag_value)
	}
	if b.Get_provider() != "db" {
		t.Errorf("provider is %s", b.Get_provider())
	}
}
