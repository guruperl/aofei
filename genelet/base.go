package genelet

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Base struct {
	C            *Config
	W            http.ResponseWriter
	R            *http.Request
	RoleValue    string
	ChartagValue string
}

func (self *Base) Fulfill() error {
	ARGS := self.R.Form
	go_uri := ARGS.Get(self.C.GoURIName)
	self.RoleValue = ARGS.Get(self.C.RoleName)
	self.ChartagValue = ARGS.Get(self.C.TagName)

	if self.RoleValue != "" && self.ChartagValue != "" {
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
		self.RoleValue = path_info[0]
		self.ChartagValue = path_info[1]
	}

	if self.RoleValue == "" {
		return Err(404, "Redirected role name not found")
	}
	_, ok := self.C.Roles[self.RoleValue]
	if !ok {
		return Err(404, "Redirected role not found")
	}
	return nil
}

func (self *Base) GetRole() Role {
	return self.C.Roles[self.RoleValue]
}

func (self *Base) GetProvider() string {
	if self.RoleValue == "" {
		return ""
	}
	role, ok := self.C.Roles[self.RoleValue]
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

func (self *Base) SendStatusPage(status int, output ...string) {
	chartag, ok := self.C.Chartags[self.ChartagValue]
	ct := "text/html; charset=UTF-8"
	if ok {
		ct = chartag.ContentType
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
}

func (self *Base) SendPage(output string) {
	self.SendStatusPage(200, output)
}

func (self *Base) SendNocache(output string) {
	self.W.Header().Set("Pragma", "no-cache")
	self.W.Header().Set("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate")
	self.SendStatusPage(200, output)
}

func (self *Base) GetIP() string {
	host, _, _ := net.SplitHostPort(self.R.RemoteAddr)
	return host
}

//func (self *Base) GetIP_int() uint32 {
//	x := net.ParseIP(self.GetIP())
//	return binary.BigEndian.Uint32(x.To4())
//}

func (self *Base) SetCookie(name string, value string, max_age ...int) {
	domain := self.R.Host
	path := "/"
	role, ok := self.C.Roles[self.RoleValue]
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

func (self *Base) SetCookieSession(name string, value string) {
	self.SetCookie(name, value)
}

func (self *Base) SetCookieExpire(name string) {
	self.SetCookie(name, "0", -365*24*3600)
}
