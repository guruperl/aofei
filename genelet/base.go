package genelet

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Base struct {
	C             *Config
	W             http.ResponseWriter
	R             *http.Request
	Role_value    string
	Chartag_value string
}

func (self *Base) Fulfill() error {
	ARGS := self.R.Form
	go_uri := ARGS.Get(self.C.Go_uri_name)
	self.Role_value = ARGS.Get(self.C.Role_name)
	self.Chartag_value = ARGS.Get(self.C.Tag_name)

	if self.Role_value != "" && self.Chartag_value != "" {
		return nil
	}

	new_url, err := url.Parse(go_uri)
	if err != nil {
		return Err(404, "Redirected URL not found")
	}

	length := len(self.C.Script)
	u1 := new_url.Path[:length]
	u2 := new_url.Path[length+1:]
	if u1 == self.C.Script && len(u2) > 0 {
		path_info := strings.Split(u2, "/")
		self.Role_value = path_info[0]
		self.Chartag_value = path_info[1]
	}

	if self.Role_value == "" {
		return Err(404, "Redirected role name not found")
	}
	_, ok := self.C.Roles[self.Role_value]
	if !ok {
		return Err(404, "Redirected role not found")
	}
	return nil
}

func (self *Base) GetRole() Role {
	return self.C.Roles[self.Role_value]
}

func (self *Base) Get_provider() string {
	if self.Role_value == "" {
		return ""
	}
	role, ok := self.C.Roles[self.Role_value]
	if !ok {
		return ""
	}
	one := ""
	for key, val := range role.Issuers {
		if val.Default {
			return key
		}
		one = key
	}
	return one
}

func (self *Base) Send_status_page(status int, output ...string) {
	chartag, ok := self.C.Chartags[self.Chartag_value]
	ct := "text/html; charset=UTF-8"
	if ok {
		ct = chartag.Content_type
	}
	self.W.Header().Set("Content-Type", ct)

	if status == 303 || status == 302 || status == 301 {
		if output != nil {
			self.W.Header().Set("Location", output[0])
		}
		self.W.WriteHeader(status)
		return
	}

	self.W.WriteHeader(status)
	if output != nil {
		self.W.Write([]byte(output[0]))
	}
	return
}

func (self *Base) Send_page(output string) {
	self.Send_status_page(200, output)
	return
}

func (self *Base) Send_nocache(output string) {
	self.W.Header().Set("Pragma", "no-cache")
	self.W.Header().Set("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate")
	self.Send_status_page(200, output)
}

func (self *Base) Get_ip() string {
	host, _, _ := net.SplitHostPort(self.R.RemoteAddr)
	return host
}

//func (self *Base) Get_ip_int() uint32 {
//	x := net.ParseIP(self.Get_ip())
//	return binary.BigEndian.Uint32(x.To4())
//}

func (self *Base) Set_cookie(name string, value string, max_age ...int) {
	domain := self.R.Host
	path := "/"
	role, ok := self.C.Roles[self.Role_value]
	if ok && role.Domain != "" {
		domain = role.Domain
	}
	if ok && role.Path != "" {
		path = role.Path
	}

	var cookie http.Cookie
	if max_age != nil {
		expiration := time.Now().Add(time.Duration(max_age[0]) * time.Second)
		cookie = http.Cookie{Name: name, Value: value, Domain: domain, Path: path, MaxAge: max_age[0], Expires: expiration}
	} else {
		cookie = http.Cookie{Name: name, Value: value, Domain: domain, Path: path}
	}
	http.SetCookie(self.W, &cookie)
}

func (self *Base) Set_cookie_session(name string, value string) {
	self.Set_cookie(name, value)
}

func (self *Base) Set_cookie_expire(name string) {
	self.Set_cookie(name, "0", -365*24*3600)
}
