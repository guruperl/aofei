package genelet

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

type TGate struct {
	Gate
}

func (self *TGate) Set_ip() string {
	return "123.123.123.123"
}
func (self *TGate) Set_when() int {
	return 1
}
func New_TGate(base Base) *TGate {
	a := new(TGate)
	a.CGI = a
	a.Base = base
	return a
}

func TestGate(t *testing.T) {
	configure, err := NewConfig(filename)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("GET", "http://example.com/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	b := new_Base(configure, "m", "json", w, req)
	g := New_Gate(*b)
	err = g.Handler_logout()

	h := w.Header()["Set-Cookie"]
	matched, err := regexp.MatchString("^mc=0; Path=/; Domain=genelet", h[0])
	if !matched {
		t.Errorf("%s wanted", h[0])
	}
	matched, err = regexp.MatchString("^mc_=0; Path=/; Domain=genelet", h[1])
	if !matched {
		t.Errorf("%s wanted", h[1])
	}
	matched, err = regexp.MatchString("^go_probe=0; Path=/; Domain=genelet", h[2])
	if !matched {
		t.Errorf("%s wanted", h[2])
	}

	cookie := &http.Cookie{Name: "mc", Value: "Ec9rwEEzh1/0UTuoE7dvi/k4lCC5RHm/SgbasG0Jca7XoTUFbKrrnkpWOcmZ8UQUEPAMPeLsi0pteOPNl2s1TO2I", Path: "/", Domain: "genelet.com"}
	b.R.AddCookie(cookie)
	access := New_TGate(*b)
	err = access.Forbid()
	if err != nil {
		t.Errorf("%s got\n", err.Error())
	}

	access.Set_attributes(map[string]string{"last_name": "aaa", "address": "bbb", "company": "ccc"})
	x := w.Header()
	c := x["Set-Cookie"][3]
	matched, err = regexp.MatchString("Ec9rwEEzh1\\/0UTuoE7dvi\\/k4lCC5RHm\\/SgbasG0JIvyA8HlXP\\+fAv38AbqPM8jhMZe8wWvTCyWgdOfaQklgxZvC24NY9RuvIwjMplkfD", c)
	if !matched {
		t.Errorf("%s wanted", c)
	}

	b.R.Header.Del("Cookie")
	cookie = &http.Cookie{Name: "mc", Value: "Ec9rwEEzh1/0UTuoE7dvi/k4lCC5RHm/SgbasG0JIvyA8HlXP+fAv38AbqPM8jhMZe8wWvTCyWgdOfaQklgxZvC24NY9RuvIwjMplkfD", Path: "/", Domain: "genelet.com"}
	b.R.AddCookie(cookie)
	access = New_TGate(*b)
	err = access.Forbid()
	if err != nil {
		t.Errorf("%s got\n", err.Error())
	}

	b.R.Header.Del("Cookie")
	cookie = &http.Cookie{Name: "mc", Value: "xxEc9rwEEzh1/0UTuoE7dvi/k4lCC5RHm/SgbasG0Jca7XoTUFbKrrnkpWOcmZ8UQUEPAMPeLsi0pteOPNl2s1TO2I", Path: "/", Domain: "genelet.com"}
	b.R.AddCookie(cookie)
	err = access.Forbid()
	if err.Error() != `{"data":"challenge"}` {
		t.Errorf("%s got\n", err.Error())
	}
	access.Chartag_value = "e"
	err = access.Forbid()
	if err.Error() != `bb/m/e/login?go_uri=&go_err=1025&role=m&tag=e&provider=db` {
		t.Errorf("%s got\n", err.Error())
	}
}
