package genelet

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/golang/glog"
)

func Oauth1_sign(method string, uri string, hash map[string]string, items []string, combined []string, form url.Values) string {
	tmps := make([]string, len(items))
	for i, v := range items {
		tmps[i] = v
	}
	if form != nil {
		for k := range form {
			tmps = append(tmps, k)
		}
	}
	sort.Strings(tmps)
	x_items := make([]string, len(tmps))
	for i, v := range tmps {
		x := hash[v]
		if x == "" && form != nil {
			x = form.Get(v)
		}
		x_items[i] = v + "%3D" + url.QueryEscape(x)
	}
	str := method + "&" + url.QueryEscape(uri) + "&" + strings.Join(x_items, "%26")
	key := combined[0] + "&"
	if len(combined) > 1 {
		key += combined[1]
	}

	return url.QueryEscape(Digest64(key, str))
}

func Oauth1_request(method string, uri string, hash map[string]string, items []string, combined []string, x_li_format string, form url.Values) ([]byte, error) {
	oauth_signature := Oauth1_sign(method, uri, hash, items, combined, form)

	x_items := make([]string, len(items))
	for i, key := range items {
		x_items[i] = key + "=\"" + hash[key] + "\""
	}
	h := make(map[string]string)
	h["Authorization"] = "OAuth oauth_signature=\"" + oauth_signature + "\", " + strings.Join(x_items, ", ")
	if x_li_format != "" {
		h["x-li-format"] = x_li_format
	}

	glog.Infof("%s\n", uri)
	glog.Infof("%v\n", h)
	return Do(method, uri, form, h)
}

func get_body(method string, uri string, hash map[string]string, items []string, combined []string) (map[string]interface{}, error) {
	for _, v := range []string{"oauth_consumer_key", "oauth_nonce", "oauth_signature_method", "oauth_timestamp", "oauth_version"} {
		items = append(items, v)
	}
	body, err := Oauth1_request(method, uri, hash, items, combined, "", nil)
	if err != nil {
		return nil, err
	}

	glog.Infof("%s\n", string(body))
	back := make(map[string]interface{})
	a := strings.Split(string(body), "&")
	for _, v := range a {
		b := strings.Split(v, "=")
		back[b[0]] = b[1]
	}

	return back, nil
}

type Oauth1 struct {
	Procedure
	Default_pars map[string]string
	Combined     []string
	X_li_format  string
}

func New_Oauth1(base Base, db *sql.DB, uri string, provider string) *Oauth1 {
	a := new(Oauth1)
	a.CGI = a
	a.Base = base
	a.Db = db
	a.Uri = uri
	a.Provider = provider
	a.Default_pars = make(map[string]string)
	a.Combined = make([]string, 0)
	a.Default_pars["oauth_signature_method"] = "HMAC-SHA1"
	a.Default_pars["oauth_version"] = "1.0"
	switch provider {
	case "twitter":
		a.Default_pars["oauth_request_token"] = "https://api.twitter.com/oauth/request_token"
		a.Default_pars["oauth_authorize_uri"] = "https://api.twitter.com/oauth/authorize"
		a.Default_pars["oauth_access_token"] = "https://api.twitter.com/oauth/access_token"
	//a.Default_pars["oauth_endpoint"]			= "https://api.twitter.com/1.1/account/settings.json"
	case "linkedin":
		a.Default_pars["oauth_request_token"] = "https://api.linkedin.com/uas/oauth/requestToken"
		a.Default_pars["oauth_authorize_uri"] = "https://api.linkedin.com/uas/oauth/authenticate"
		a.Default_pars["oauth_access_token"] = "https://api.linkedin.com/uas/oauth/accessToken"
		a.Default_pars["oauth_endpoint"] = "https://api.linkedin.com/v1/people/~:(id,first-name,last-name,publicProfileUrl,pictureUrl)"
		a.Default_pars["fields"] = "id,email,name,first_name,last_name,age_range,gender"
	}

	role := base.C.Roles[base.Role_value]
	issuer := role.Issuers[provider]
	for k, v := range issuer.Provider_pars {
		a.Default_pars[k] = v
	}

	return a
}

func (self *Oauth1) Authenticate(login, password string) error {
	role := self.C.Roles[self.Role_value]
	hash := self.Default_pars
	hash["oauth_callback"] = url.QueryEscape(self.Callback_address())

	now := int32(time.Now().Unix())
	hash["oauth_timestamp"] = fmt.Sprintf("%d", now)
	hash["oauth_nonce"] = fmt.Sprintf("%x%8x", os.Getpid(), now)

	if login == "" {
		self.Combined = []string{hash["oauth_consumer_secret"]}
		back, err := get_body("GET", hash["oauth_request_token"], hash, []string{"oauth_callback"}, self.Combined)
		glog.Infof("%v\n", back)
		if err != nil {
			return err
		}
		if back["oauth_callback_confirmed"].(string) != "true" {
			return Err(404)
		}
		self.Set_cookie(self.Provider, Encode_scoder(back["oauth_token_secret"].(string), role.Coding))
		return Err(303, hash["oauth_authorize_uri"]+"?"+"oauth_token="+back["oauth_token"].(string)+"&oauth_callback="+hash["oauth_callback"])
	}

	hash["oauth_token"] = login
	hash["oauth_verifier"] = password
	oauth_token_secret, err := self.R.Cookie(self.Provider)
	if err != nil {
		return err
	}
	if oauth_token_secret == nil {
		return Err(404)
	}

	hash["oauth_token_secret"] = oauth_token_secret.Value
	hash["oauth_token_secret"] = Decode_scoder(hash["oauth_token_secret"], role.Coding)
	self.Combined = []string{hash["oauth_consumer_secret"], hash["oauth_token_secret"]}
	back, err := get_body("GET", hash["oauth_access_token"], hash, []string{"oauth_token", "oauth_verifier"}, self.Combined)
	if err != nil {
		return err
	}
	for k, v := range back {
		hash[k] = v.(string)
	}
	// we get back oauth_token oauth_token_secret user_id screen_name x_auth_expires
	// need to re-assigned to Combined
	self.Combined = []string{hash["oauth_consumer_secret"], hash["oauth_token_secret"]}

	if hash["oauth_endpoint"] != "" {
		back1, err := self.Oauth1_api("GET", hash["oauth_endpoint"], nil)
		if err != nil {
			return err
		}
		for k, v := range back1 {
			back[k] = v
		}
	}

	for k, v := range hash {
		back[k] = v
	}
	glog.Infof("%#v\n", back)

	// oauth_token oauth_token_secret user_id screen_name x_auth_expires
	// oauth_consumer_key oauth_consumer_secre already
	return self.Fill_provider(back)
}

func (self *Oauth1) oauth1_request(method string, uri string, form url.Values) ([]byte, error) {
	hash := self.Default_pars
	now := int32(time.Now().Unix())
	hash["oauth_timestamp"] = fmt.Sprintf("%d", now)
	hash["oauth_nonce"] = fmt.Sprintf("%x%8x", os.Getpid(), now)
	items := []string{"oauth_consumer_key", "oauth_nonce", "oauth_signature_method", "oauth_token", "oauth_timestamp", "oauth_version"}
	return Oauth1_request(method, uri, hash, items, self.Combined, "json", form)
}

func (self *Oauth1) Oauth1_api(method string, uri string, form url.Values) (map[string]interface{}, error) {
	body, err := self.oauth1_request(method, uri, form)
	if err != nil {
		return nil, err
	}

	return To_hash(body)
}

func (self *Oauth1) Oauth1_apis(method string, uri string, form url.Values) ([]map[string]interface{}, error) {
	body, err := self.oauth1_request(method, uri, form)
	if err != nil {
		return nil, err
	}

	return To_slice(body)
}
