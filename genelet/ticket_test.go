package genelet;

import (
	"testing"
	"log"
	"regexp"
	"net/http"
	"net/http/httptest"
)

type TTicket struct {
    Ticket
}
func New_TTicket(base Base, uri string, provider string) *TTicket {
    a := new(TTicket)
    a.CGI = a
    a.Base = base
    a.Uri = uri
    a.Provider = provider
    return a
}
func (self *TTicket)Set_ip() string {
    return "123.123.123.123"
}
func (self *TTicket)Set_when() int {
    return 1
}
func (self *TTicket)Authenticate(login, password string) error {
    if (login=="" || password=="") { return Err(1037) }
    role := self.C.Roles[self.Role_value]
    issuer := role.Issuers[self.Provider]
    if (login != issuer.Provider_pars["Def_login"] || password != issuer.Provider_pars["Def_password"]) {
        return Err(1031)
    }

    self.Out_hash = map[string]interface{}{"email":issuer.Provider_pars["Def_login"], "user":"x2"}

    return nil
}

func TestTicket(t *testing.T) {
    configure := NewConfig(filename)
	req, err := http.NewRequest("GET", "http://example.com/foo?email=hello&passwd=world", nil)
	if err != nil {
		log.Fatal(err)
	}
	w := httptest.NewRecorder()

	//b := new_Base(configure, "m", "json", w, req)
// test login page with json
	//ticket := New_Ticket(*b, "http://xxx.yyy.zzz/foo/bar", "db")
	//str := ticket.Login_page(1001)
	//if str != "{\"data\":\"failed\"}" {
	//	t.Errorf("%s wanted", str)
	//}

// test login page with mime foo and error code 1001
	//b = new_Base(configure, "m", "foo", w, req)
	//ticket = New_Ticket(*b, "http://xxx.yyy.zzz/foo/bar", "db")
	//str = ticket.Login_page(1001)
	//matched, err := regexp.MatchString("Google authorization required", str)
	//if !matched {
	//	t.Errorf("%s wanted", str)
	//}
	//matched, err = regexp.MatchString("email.*passwd", str)
	//if !matched {
//		t.Errorf("%s wanted", str)
//	}
// the location of on-disk template, that does not exist
	//if (ticket.C.Template+"/"+ticket.Role_value+"/"+ticket.C.Roles["m"].Login+"."+ticket.Chartag_value != "ee/m/login.foo") {
//		t.Errorf("%s\t%s\t%s\t%s\n", ticket.C.Template, ticket.Role_value, ticket.C.Roles["m"].Login, ticket.Chartag_value)
	//}

// test authentication
	b := new_Base(configure, "m", "json", w, req)
	tticket := New_TTicket(*b, "http://xxx.yyy.zzz/foo/bar", "db")
	tticket.Uri="asw"
	ret := tticket.Authenticate("","w")
	if ret.(Gerror).Code != 1037 {
		t.Errorf("%s returned", ret.Error())
	}
	ret = tticket.Authenticate("h","w")
	if ret.(Gerror).Code != 1031 {
		t.Errorf("%s returned", ret.Error())
	}
	ret = tticket.Authenticate("hello","world")
	if ret != nil {
		t.Errorf("%s returned", ret.Error())
	}
	if tticket.C.Roles["m"].Attributes[0] != "email" {
		t.Errorf("%s wanted", tticket.C.Roles["m"].Attributes[0])
	}
	if tticket.C.Roles["m"].Attributes[1] != "m_id" {
		t.Errorf("%s wanted", tticket.C.Roles["m"].Attributes[1])
	}
	if tticket.Out_hash["email"] != "hello" {
		t.Errorf("%s wanted", tticket.Out_hash["login"])
	}
	if tticket.Out_hash["user"] != "x2" {
		t.Errorf("%s wanted", tticket.Out_hash["x2"])
	}
	ret = tticket.Handler_fields()
	if (ret != nil) {
		t.Errorf("%s returned", ret.Error())
	}

// test Handler_login which also test authentication and login page
	w = httptest.NewRecorder()
	b = new_Base(configure, "m", "e", w, req)
	tticket = New_TTicket(*b, "http://xxx.yyy.zzz/foo/bar", "db")
	_ = req.ParseForm()
	ret = tticket.Handler_login()
	if ret.(Gerror).Code != 303 {
		t.Errorf("%s returned", ret.Error())
	}
	h := w.Header()
	matched, err := regexp.MatchString("^mc=Ec9rwEEzh1\\/0UTuoE7dvi\\/k4lCC5RHKsRBaR\\/yMLbq3QvmN8Dd37pnJaBfiillFOGaQJZ\\/\\+RyhRgVb7hjHU=", h.Get("Set-Cookie"))
	if !matched {
		// t.Errorf("%s wanted", h.Get("Set-Cookie"))
	}
	ret = tticket.Verify_cookie("Ec9rwEEzh1/0UTuoE7dvi/k4lCC5RHKsRBaR/yMLbq3QvmN8Dd37pnJaBfiillFOGaQJZ/+RyhRgVb7hjHU=")
	if ret != nil {
		t.Errorf("%s wanted", ret.Error())
	}
	h = tticket.R.Header
	if (h["X-Forwarded-User"][0] != "hello") {
		t.Errorf("%s wanted", h["X-Forwarded-User"])
	}
	if (h["X-Forwarded-Group"][0] != `||||`) {
		t.Errorf("%s wanted", h["X-Forwarded-Group"])
	}

// test Handler with direct login
	req, _ = http.NewRequest("GET", "http://example.com/foo?email=hello&passwd=world&direct=1", nil)
	w = httptest.NewRecorder()
	b = new_Base(configure, "m", "e", w, req)
	tticket = New_TTicket(*b, "http://xxx.yyy.zzz/foo/bar", "db")
	_ = req.ParseForm()
	ret = tticket.Handler_login()
	if ret.(Gerror).Code != 303 {
		t.Errorf("%s wanted", ret.Error())
	}
	h = w.Header()
	matched, err = regexp.MatchString("^mc=Ec9rwEEzh1\\/0UTuoE7dvi\\/k4lCC5RHKsRBaR\\/yMLbq3QvmN8Dd37pnJaBfiillFOGaQJZ\\/\\+RyhRgVb7hjHU=", h.Get("Set-Cookie"))
	if !matched {
		// t.Errorf("%s wanted", h.Get("Set-Cookie"))
	}

// test Handler with error code but no cookie
	req, _ = http.NewRequest("GET", "http://example.com/foo?email=hello&passwd=world&go_err=1020", nil)
	w = httptest.NewRecorder()
	b = new_Base(configure, "m", "e", w, req)
	tticket = New_TTicket(*b, "http://xxx.yyy.zzz/foo/bar", "db")
	_ = req.ParseForm()
	ret = tticket.Handler()
	if ret.(Gerror).Code != 1036 {
		t.Errorf("%#v found", ret)
	}

// test Handler with error code and cookie, the most common case for redirect
	req, _ = http.NewRequest("GET", "http://example.com/foo?go_uri=xxxx&go_err=1020", nil)
	w = httptest.NewRecorder()
	b = new_Base(configure, "m", "e", w, req)
	tticket = New_TTicket(*b, "http://xxx.yyy.zzz/foo/bar", "db")
	_ = req.ParseForm()
	cookie := &http.Cookie{Name:"go_probe", Value:"http://xxx.yyy.zzz/foo/bar", Path:"/", Domain:"genelet.com"}
	req.AddCookie(cookie)
	ret = tticket.Handler()
	if ret.(Gerror).Code != 1020 {
		t.Errorf("%#v found", ret)
	}

// test Handler with login and password, as well as cookie
	req, _ = http.NewRequest("GET", "http://example.com/foo?email=hello&passwd=world", nil)
	w = httptest.NewRecorder()
	b = new_Base(configure, "m", "e", w, req)
	tticket = New_TTicket(*b, "http://xxx.yyy.zzz/foo/bar", "db")
	_ = req.ParseForm()
	cookie = &http.Cookie{Name:"go_probe", Value:"http://xxx.yyy.zzz/foo/bar", Path:"/", Domain:"genelet.com"}
	req.AddCookie(cookie)
	ret = tticket.Handler()
	if ret.(Gerror).Code != 303 {
		t.Errorf("%s wanted", ret.Error())
	}
	h = w.Header()
	matched, err = regexp.MatchString("^mc=Ec9rwEEzh1\\/0UTuoE7dvi\\/k4lCC5RHKsRBaR\\/yMLbq3QvmN8Dd37pnJaBfiillFOGaQJZ\\/\\+RyhRgVb7hjHU=", h.Get("Set-Cookie"))
	if !matched {
		// t.Errorf("%s wanted", h.Get("Set-Cookie"))
	}
	if ret.(Gerror).Errstr != "http://xxx.yyy.zzz/foo/bar" {
		t.Errorf("%s wanted", ret.(Gerror).Errstr)
	}
}
