package genelet

import (
	"net/url"
	"strings"
)

type Gate struct {
	Access
}

func New_Gate(base Base) *Gate {
	a := new(Gate)
	a.CGI = a
	a.Base = base
	return a
}

func (self *Gate) Forbid() error {
	c := self.C
	role, ok := c.Roles[self.Role_value]
	if !ok {
		return nil
	}

	coo, err := self.R.Cookie(role.Surface)
	if err == nil {
		err = self.Verify_cookie(coo.Value)
		if err == nil {
			return nil
		}
	}

	chartag, ok := c.Chartags[self.Chartag_value]
	if ok && chartag.Case > 0 {
		return Err(200, chartag.Call_challenge())
	}

	escaped := url.QueryEscape(self.R.RequestURI)
	self.Set_cookie_session(c.Go_probe_name, escaped)
	self.Set_cookie_expire(role.Surface)

	var k string
	for k = range role.Issuers {
		if !Grep(c.Oauth2s, k) && !Grep(c.Oauth1s, k) {
			redirect := c.Script + "/" + self.Role_value + "/" + self.Chartag_value + "/" + c.Login_name + "?" + c.Go_uri_name + "=" + escaped + "&" + c.Go_err_name + "=1025&" + c.Role_name + "=" + self.Role_value + "&" + c.Tag_name + "=" + self.Chartag_value + "&" + c.Provider_name + "=" + k
			return Err(303, redirect)
		}
	}
	redirect := c.Script + "/" + self.Role_value + "/" + self.Chartag_value + "/" + k + "?" + c.Go_uri_name + "=" + escaped + "&" + c.Go_err_name + "=1025&" + c.Tag_name + "=" + self.Chartag_value
	return Err(303, redirect)
}

func (self *Gate) Handler_logout() error {
	role, ok := self.C.Roles[self.Role_value]
	if !ok {
		return Err(1029)
	}

	self.Set_cookie_expire(role.Surface)
	self.Set_cookie_expire(role.Surface + "_")
	self.Set_cookie_expire(self.C.Go_probe_name)

	chartag, ok := self.C.Chartags[self.Chartag_value]
	if ok && chartag.Case > 0 {
		return Err(200, chartag.Call_logout())
	} else {
		return Err(303, role.Logout)
	}
}

/*
func (self *Gate)Get_attribute(ref string) (string, error) {
	role, ok := self.C.Roles[self.Role_value]
    if (!ok) { return "", Err(1029) }

	_, login, group, _, _, err := self.get_cookie()
    if err != nil { return "", err }

	if (ref==role.Attributes[0]) {
		return login, nil
	}

	groups := strings.Split(group, "|")
	for i, a := range role.Attributes {
		if (len(groups) >= i && ref==a) {
			return groups[i-1], nil
		}
	}

	return "", Err(1039)
}
*/

func (self *Gate) Get_attribute(key string) (string, error) {
	ref := make(map[string]string)
	err := self.Get_attributes(ref)
	if err != nil {
		return "", err
	}
	val, ok := ref[key]
	if !ok {
		return "", Err(1039)
	}

	return val, nil
}

func (self *Gate) Get_attributes(ref map[string]string) error {
	role, ok := self.C.Roles[self.Role_value]
	if !ok {
		return Err(1029)
	}

	_, login, group, _, _, err := self.get_cookie()
	if err != nil {
		return err
	}

	groups := strings.Split(group, "|")
	for i, a := range role.Attributes {
		if i == 0 {
			ref[a] = login
		} else if len(groups) >= i {
			ref[a] = groups[i-1]
		}
	}
	return nil
}

func (self *Gate) Set_attribute(key string, value string) error {
	ref := make(map[string]string)
	ref[key] = value
	return self.Set_attributes(ref)
}

func (self *Gate) Set_attributes(ref map[string]string) error {
	role, ok := self.C.Roles[self.Role_value]
	if !ok {
		return Err(1029)
	}

	ip, login, group, when, hash, err := self.get_cookie()
	if err != nil {
		return err
	}

	n_login, ok := ref[role.Attributes[0]]
	if ok {
		login = n_login
	}

	groups := strings.Split(group, "|")
	new_groups := make([]string, len(role.Attributes)-1)
	for i := 1; i < len(role.Attributes); i++ {
		n_value, ok := ref[role.Attributes[i]]
		if ok {
			new_groups[i-1] = n_value
		} else if len(groups) >= i {
			new_groups[i-1] = groups[i-1]
		} else {
			new_groups[i-1] = ""
		}
	}

	new_group := strings.Join(new_groups, "|")
	hash = Digest(role.Secret, ip, login, new_group, when)
	signed := Encode_scoder(strings.Join([]string{ip, login, new_group, when, hash}, "/"), role.Coding)

	self.Set_cookie(role.Surface, signed, role.Max_age)
	self.Set_cookie_session(role.Surface+"_", signed)

	return nil
}
