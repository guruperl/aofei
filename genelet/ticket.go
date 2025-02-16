package genelet

import (
	"strconv"
	"strings"
	"net/url"
	"fmt"
	"github.com/delongw/go-int-cipher"
//	"github.com/golang/glog"
)

type Ticket struct {
	Access
	Uri			string
	Provider	string
	Out_hash	map[string]interface{}
// each item in condition uri is: variable name, how, variable value, base uri, other variable names wholse values would be passed in the redirect.
// how: t--out hash, u--go uri, p--provider; A--equal, B--not equal, C--match; 0--no sign, 1--sign;  0--no authen cookie, 1--yes authen cookie
// for u and p in 'how', 'variable name' will be go_uri that matches
	Condition_uri [][]string
}

func New_Ticket(base Base, uri string, provider string) *Ticket {
	a := new(Ticket)
	a.CGI = a
	a.Base = base
	a.Uri = uri
	a.Provider = provider
	return a
}

func (self *Ticket) Handler() error {
	ARGS := self.R.Form
	found, err := self.R.Cookie(self.C.Go_probe_name)
	if err != nil {
		self.Set_cookie_session(self.C.Go_probe_name, self.R.RequestURI)
		return Err(1036)
	}
	if self.Uri == "" { self.Uri = found.Value }

    if ARGS.Get(self.C.Go_err_name) != "" {
        code, _ := strconv.Atoi(ARGS.Get(self.C.Go_err_name))
		return Err(code)
    }

	return self.Handler_login()
}

func (self *Ticket)Handler_login() error {
	ARGS := self.R.Form
	role := self.C.Roles[self.Role_value]
	issuer := role.Issuers[self.Provider]

	if passin := ARGS.Get(role.Surface); passin != "" {
		if role.Surface==issuer.Credential[3] {
			if err := self.Verify_cookie(passin); err != nil {
				return err
			} else {
				self.Set_cookie(role.Surface, passin, role.Max_age)
				self.Set_cookie_session(role.Surface+"_", passin)
				return Err(303, self.Uri)
			}
		}
	}

	login    := ARGS.Get(issuer.Credential[0])
	password := ARGS.Get(issuer.Credential[1])
	if err := self.CGI.Authenticate(login, password); err != nil {
		return err
	}

	return self.Handler_fields()
}

func (self *Ticket)Get_attributes() []string {
	role := self.C.Roles[self.Role_value]
	fields := make([]string, len(role.Attributes))
	for i, v := range role.Attributes {
		if self.Out_hash[v] == nil { continue }
		out := Interface2String(self.Out_hash[v])
		if v==role.Id_name && role.Id_cipher {
			id64, _ := strconv.ParseInt(out, 10, 64)
			fields[i] = strconv.FormatInt(int64(int_cipher.Encrypt(uint(id64), self.C.Secret)), 10)
		} else {
			fields[i] = out
		}
	}
	return fields
}

func (self *Ticket)Handler_fields() error {
	c := self.C
	role := c.Roles[self.Role_value]
	fields := self.Get_attributes()
	if fields[0]=="" { return Err(1032) }

	final_uri := self.Uri

	// how, variable name, value, base uri, others wholse values in redirect
	// how: (t, u, p)+(A, B, C)+(0,1)+(0,1), t:out_hash, u:go_uri, p:provider
	// for u and p, go_uri has to match values[1] (i.e. 2nd match)
	// A:equal, B:not equal, C:match, 
	// 0: no stamp nor md5, 1: stamp and md5; 0:no coookie, 1:cookie
	c_uri := (role.Issuers[self.Provider]).Condition_uri
	if c_uri != nil {
		outs := self.Out_hash
		for _, values := range c_uri {
			how := values[0]
			obs := ""
			if how[0:1]=="t" {
				obs = outs[values[1]].(string)
			} else {
				if how[0:1]=="p" {
					obs = self.Provider
				} else {
					obs = final_uri
				}
				u, err := url.Parse(final_uri)
				if err != nil { continue }
				if len(u.Path) < len(values[1]) { continue }
				if (u.Path)[0:len(values[1])] != values[1] { continue }
			}
			if (how[1:2]=="A" && obs==values[2]) ||
			   (how[1:2]=="B" && obs!=values[2]) ||
			   (how[1:2]=="C" && strings.Contains(obs, values[2])) {
				target, err := url.Parse(values[3])
				if err != nil { break }
				q := target.Query()
				for i:=4; i<len(values); i++ {
					if v, ok := outs[values[i]]; ok {
						q.Add(values[i], Interface2String(v))
					}
				}
				if how[2:3]=="1" {
					q.Add(c.Go_stamp_name,fmt.Sprintf("%d", Unix_timestamp()))
					q.Add(c.Go_md5_name, SortMapMd5(c.Secret, self.C.Go_md5_name, q))
				}
				target.RawQuery = q.Encode()
				final_uri = target.RequestURI()
				if how[3:4]=="0" {
					self.Uri = final_uri
					return Err(303, final_uri)
				}
			}
		}
	}

	signed := self.Signature(fields...)
	self.Set_cookie(role.Surface, signed, role.Max_age)
	self.Set_cookie_session(role.Surface+"_", signed)

	chartag, ok := c.Chartags[self.Chartag_value]
	if ok && chartag.Case > 0 {
		self.Send_nocache(chartag.Call_logged())
		return nil
	}

	self.Uri = final_uri
	return Err(303, final_uri)
}

func (self *Ticket)Authenticate(login, password string) error {
	if login=="" || password=="" { return Err(1037) }
	role := self.C.Roles[self.Role_value]
	issuer := role.Issuers[self.Provider]
	if login != issuer.Provider_pars["Def_login"] || password != issuer.Provider_pars["Def_password"] {
		return Err(1031)
	}

	role.Attributes = []string{"login","provider"}
	self.Out_hash = map[string]interface{}{"login":issuer.Provider_pars["Def_login"],"provider":self.Provider}

	return nil
}
