package genelet

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	int_cipher "github.com/delongw/go-int-cipher"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
)

/*
type Mmodel interface {
    Initialize(*Component)
    Set_db(*sql.DB)
    Set_defaults(url.Values, *[]map[string]interface{}, *map[string]interface{})
}

type Ffilter interface {
    Initialize(*Component)
    Set_all(Base, string, string, *map[string]interface{})
    Get_all() (map[string][]string, []string)
    Preset() error
    Before(*sql.DB, url.Values, url.Values) error
    After(*sql.DB, *[]map[string]interface{}, *map[string]interface{}) error
}
*/

type Controller struct {
	C       *Config
	Db      *sql.DB
	Models  map[string]interface{}
	Filters map[string]interface{}
	Storage map[string]interface{}
}

func (self *Controller) static_page(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.URL.Path, `..`) {
		http.NotFound(w, r)
	}

	glog.Infof("(%d) %s\n", os.Getpid(), "not genelet pattern")
	c := self.C
	url := r.URL
	d_root := c.DocumentRoot
	if c.DocumentRoots != nil {
		glog.Infof("(%d) %s\n", os.Getpid(), "virtual host: "+r.Host)
		if x, ok := c.DocumentRoots[r.Host]; ok {
			d_root = x
		}
	}

	/*
	   	form := r.Form
	   	for _, pattern := range c.Patterns {
	   		values := pattern.Regs.FindStringSubmatch(url.Path)
	   		if (values == nil) { continue }

	   		// match found, we must serve with this pattern
	   		need_cache := false
	   		if pattern.Case==CACHE {
	   			finfo, err := os.Stat(c.DocumentRoot+url.Path)
	   			if err != nil || (time.Now().Sub(finfo.ModTime()) / time.Second ) > pattern.Expire {
	   				need_cache = true
	   				pattern.Case=REROUTE
	   			}
	   		}

	   		form := make(*url.Values)
	   		if pattern.Initials != "" {
	               q, _ := url.ParseQuery(pattern.Initials)
	               form = q
	           }

	           for i, key := range pattern.Keys {
	               v := form.Get(key)
	               if v != "" { continue; }
	               form.Set(key, values[i])
	           }

	   		role := form.Get(c.Role_name)
	   		tag := form.Get(c.Tag_name)
	   		component := form.Get(c.Component_name)

	   		role := form.Get(c.Role_name)
	   		tag := form.Get(c.Tag_name)
	   		component := form.Get(c.Component_name)
	   		if pattern.Case==STATIC || pattern.Case==CACHE {
	   			if (role=="" || role==c.Pubrole) {
	   				break
	   			}
	   			if (tag=="") {
	   glog.Infof("(%d) %s\n", os.Getpid(), "static file matched not good")
	   				http.NotFound(w,r)
	   				return
	   			}
	   			base := &Base{C:c, R:r, W:w, Role_value:role, Chartag_value:tag}
	   			gate := New_Gate(*base)
	   			err := gate.Forbid()
	   			if (err != nil) {
	   glog.Infof("(%d) %s\n", os.Getpid(), "static file asks for login")
	   				self.login_page(base)
	   				return
	   			}
	   			break
	   		} else {
	   			if role=="" || tag=="" || component=="" {
	   				http.NotFound(w,r)
	   			} else {
	   				url.Path = c.Script+"/"+role+"/"+tag+"/"+component
	   				form.Del(c.Role_name)
	   				form.Del(c.Tag_name)
	   				form.Del(c.Component_name)
	   				url.RawQuery = form.Encode()
	   			}
	   			return
	   		}
	   	}

	*/
	http.ServeFile(w, r, d_root+url.Path)
	return
}

func (self *Controller) login_page(base *Base) {
	c := self.C
	uri := base.R.Form.Get(c.Go_uri_name)

	provider := base.R.Form.Get(c.Provider_name)
	glog.Infof("(%d) %s %s\n", os.Getpid(), "provider? ", provider)
	if provider == "" {
		provider = base.Get_provider()
		if provider == "" {
			http.NotFound(base.W, base.R)
			return
		}
	}

	db := self.Db

	var err error
	if Grep(c.Oauth2s, provider) {
		ticket := New_Oauth2(*base, db, uri, provider)
		glog.Infof("(%d) %s %s\n", os.Getpid(), "oauth2 uses: ", provider)
		err = ticket.Handler_login()
		uri = ticket.Uri // use the same vriable for the targeting uri
	} else if Grep(c.Oauth1s, provider) {
		ticket := New_Oauth1(*base, db, uri, provider)
		glog.Infof("(%d) %s %s\n", os.Getpid(), "oauth1 uses: ", provider)
		err = ticket.Handler_login()
		uri = ticket.Uri // use the same vriable for the targeting uri
	} else {
		glog.Infof("(%d) %s %s\n", os.Getpid(), "login uses: ", provider)
		ticket := New_Procedure(*base, db, uri, provider)
		err = ticket.Handler()
		uri = ticket.Uri // use the same vriable for the targeting uri
	}
	if err == nil {
		return
	}
	if base.Chartag_value == "json" {
		base.Send_page(c.Chartags[base.Chartag_value].Call_challenge())
		return
	}

	glog.Infof("(%d) %s %#v\n", os.Getpid(), "ticket returns error: ", err)
	gerr := err.(Gerror)
	if gerr.Code < 1000 {
		base.Send_status_page(gerr.Code, gerr.Errstr)
		return
	}
	issuer := (c.Roles[base.Role_value]).Issuers[provider]
	T, err := template.ParseFiles(c.Template + "/" + base.Role_value + "/" + c.Login_name + "." + base.Chartag_value)
	if err == nil {
		errstr := c.Errors[strconv.Itoa(gerr.Code)]
		if errstr == "" {
			errstr = gerr.Error()
		}
		var buffer bytes.Buffer
		err = T.Execute(&buffer, map[string]interface{}{
			"Login_name": c.Login_name, "Go_uri_name": c.Go_uri_name,
			"Errorstr": errstr, "Go_uri": uri,
			"Login": issuer.Credential[0], "Password": issuer.Credential[1]})
		base.Send_nocache(buffer.String())
	}
	if err != nil {
		http.Error(base.W, err.Error(), 500)
	}
}

func check_form(r *http.Request, dir string) error {
	reader, err := r.MultipartReader()

	if reader == nil {
		glog.Infof("(%d) %s\n", os.Getpid(), "No multipart")

		err = r.ParseForm()
		if err != nil {
			return err
		}
		form := r.Form

		header := r.Header
		// not json, return
		if header.Get("Content-Type") == "" || !strings.Contains(header.Get("Content-Type"), "application/json") || r.Body == nil {
			return nil
		}

		data, err := io.ReadAll(r.Body)
		t := make(map[string]interface{})
		if err == nil && data != nil {
			err = json.Unmarshal(data, &t)
		}
		if err != nil {
			return err
		}

		for key, value := range t {
			if value == nil {
				continue
			}
			switch value.(type) {
			case []string:
				for _, v := range value.([]string) {
					form.Add(key, v)
				}
			case []uint8:
				form.Add(key, string(value.([]uint8)))
			case []interface{}:
				for _, v := range value.([]interface{}) {
					form.Add(key, Interface2String(v))
				}
			default:
				form.Add(key, Interface2String(value))
			}
		}
	} else {
		glog.Infof("(%d) %s\n", os.Getpid(), "multipart/uploading found...")
		r.Form = make(url.Values)
		form := r.Form
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}

			field_name := part.FormName()
			file_name := part.FileName()
			if file_name == "" {
				scanner := bufio.NewScanner(part)
				scanner.Scan()
				form.Add(field_name, scanner.Text())
			} else {
				fullname := dir + "/" + file_name
				dst, err := os.Create(fullname)
				defer dst.Close()
				if err != nil {
					return err
				}
				if _, err := io.Copy(dst, part); err != nil {
					return err
				}
				form.Add(field_name, file_name)
			}
			part.Close()
		}
	}

	return nil
}

func (self *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := self.C
	length := len(c.Script)
	l_url := len(r.URL.Path)

	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Max-Age", "1728000")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	if acrm := r.Header.Get("Access-Control-Request-Method"); acrm != "" {
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	}
	if acrh := r.Header.Get("Access-Control-Request-Headers"); acrh != "" {
		w.Header().Set("Access-Control-Allow-Headers", acrh)
	}
	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	glog.Infof("(%d) start url: %s\n", os.Getpid(), r.URL.Path)
	if l_url <= length || r.URL.Path[:length+1] != c.Script+"/" {
		self.static_page(w, r)
		return
	}

	method_found := false
	for k := range c.Default_actions {
		if k == r.Method {
			method_found = true
			break
		}
	}
	if method_found == false {
		http.Error(w, "The http method is not supported", 405)
		return
	}

	path_info := strings.Split(r.URL.Path[length+1:], "/")
	if len(path_info) == 4 {
		r.Header.Add("X-Forwarded-ID", path_info[3])
	} else if len(path_info) != 3 {
		glog.Infof("(%d) %s\n", os.Getpid(), "not genelet url")
		http.Error(w, "Bad Request", 400)
		return
	}

	chartag, ok := c.Chartags[path_info[1]]
	if !ok {
		glog.Infof("(%d) %s\n", os.Getpid(), "check chartag")
		http.Error(w, "Bad Request", 400)
		return
	}

	base := &Base{C: c, W: w, R: r, Role_value: path_info[0], Chartag_value: path_info[1]}
	gate := New_Gate(*base)
	obj := path_info[2]

	glog.Infof("(%d) %s\n", os.Getpid(), "parse form")
	err := check_form(r, c.Upload_dir)
	if err != nil {
		http.Error(w, "Bad Request", 400)
		return
	}

	glog.Infof("(%d) %s\n", os.Getpid(), "if role "+base.Role_value+", is defined or public")
	_, ok = c.Roles[base.Role_value]
	if !ok && (gate.Role_value != c.Pubrole) {
		http.NotFound(w, r)
		return
	}

	glog.Infof("(%d) %s %s\n", os.Getpid(), "object is", obj)
	if obj == c.Login_name || Grep(c.Oauth2s, obj) || Grep(c.Oauth1s, obj) {
		glog.Infof("(%d) %s %s\n", os.Getpid(), "start login for", obj)
		if obj != c.Login_name {
			r.Form.Set(c.Provider_name, obj)
		}
		self.login_page(base)
		glog.Infof("(%d) %s\n\n", os.Getpid(), "end login ...")
		return
	} else if obj == c.Logout_name {
		glog.Infof("(%d) %s\n", os.Getpid(), "start logout")
		err = gate.Handler_logout()
		if err != nil {
			gate.Send_status_page(err.(Gerror).Code, err.(Gerror).Errstr)
			glog.Infof("(%d) %s\n\n", os.Getpid(), "end logout ...")
		}
		return
	}

	if gate.Role_value != c.Pubrole {
		err = gate.Forbid()
		if err != nil {
			glog.Infof("(%d) forbidden ... %d : %s\n\n", os.Getpid(), err.(Gerror).Code, err.(Gerror).Errstr)
			gate.Send_status_page(err.(Gerror).Code, err.(Gerror).Errstr)
			return
		}
	}

	glog.Infof("(%d) %s\n", os.Getpid(), "starting genelet handler ...")
	err = self.Handle(obj, *base, r.Method)
	if err != nil {
		switch err.(type) {
		case Gerror:
			g := err.(Gerror)
			if g.Code == 303 {
				base.Send_status_page(303, err.Error())
				return
			} else if g.Code < 1000 {
				err = Gerror{g.Code, http.StatusText(g.Code)}
			} else {
				err = Gerror{g.Code, err.Error()}
			}
		default:
			err = Gerror{1000, err.Error()}
		}
		glog.Infof("(%d) error found: %#v\n", os.Getpid(), err)

		tmplfile := c.Template + "/" + base.Role_value + "/error." + base.Chartag_value
		T0, er := template.ParseFiles(tmplfile)
		if er != nil {
			base.Send_page(add_json(chartag.Case, er.Error()))
			return
		}
		var buffer bytes.Buffer
		er = T0.Execute(&buffer, err)
		if er != nil {
			base.Send_page(add_json(chartag.Case, er.Error()))
		} else if err.(Gerror).Code < 1000 {
			base.Send_status_page(err.(Gerror).Code, buffer.String())
		} else {
			base.Send_page(buffer.String())
		}
	}
	glog.Infof("(%d) %s\n\n", os.Getpid(), "genelet handler ended.")
	return
}

func add_json(c int8, msg string) string {
	if c > 0 {
		return `{"data": "` + msg + `"}`
	}
	return msg
}

func (self *Controller) Handle(obj string, base Base, method string) error {
	model, ok := self.Models[obj]
	if !ok {
		return Err(404)
	}
	filter, ok := self.Filters[obj]
	if !ok {
		return Err(404)
	}
	who := base.Role_value
	tag := base.Chartag_value

	c := self.C
	r := base.R
	ARGS := r.Form

	ARGS.Set("_gobj", obj)
	ARGS.Set("_gtime", strconv.FormatInt(time.Now().Unix(), 10))

	lists := make([]map[string]interface{}, 0)
	other := make(map[string]interface{})
	glog.Infof("(%d) %s\n", os.Getpid(), "set defaults")
	Invoke0(model, "Set_defaults", ARGS, &lists, &other, self.Storage)

	action := ARGS.Get(c.Action_name)
	if action == "" {
		action = c.Default_actions[method]
		if method == "GET" && r.Header.Get("X-Forwarded-ID") != "" {
			action = c.Default_actions["GET_item"]
		}
	}
	if r.Header.Get("X-Forwarded-ID") != "" {
		ARGS.Set("_gid_url", r.Header.Get("X-Forwarded-ID"))
	}
	glog.Infof("(%d) %s\n", os.Getpid(), "get action: "+action)
	Invoke0(filter, "Set_all", base, action, obj, &other)
	ret := Invoke(filter, "Get_all")
	if ret[0].Interface() == nil {
		return Err(404)
	}
	actionHash := ret[0].Interface().(map[string][]string)
	fk := make([]string, 0)
	if ret[1].Interface() != nil {
		fk = ret[1].Interface().([]string)
	}
	parts := strings.Split(r.RequestURI, "/")
	parts[3] = "json"
	ARGS.Set("_guri_json", strings.Join(parts, "/"))
	ARGS.Set("_guri", r.RequestURI)
	ARGS.Set("_grole", who)
	ARGS.Set("_action", action)

	role, ok := c.Roles[base.Role_value]
	is_admin := false
	if ok {
		glog.Infof("(%d) %s\n", os.Getpid(), "parsing role's ARGS")
		ARGS.Set("_gid_name", role.Id_name)
		ARGS.Set("_gtype_id", strconv.Itoa(role.Type_id))
		if role.Is_admin {
			ARGS.Set("_gadmin", "1")
			is_admin = true
		}
		h := r.Header
		ARGS.Set("_gwhen", h.Get("X-Forwarded-Time"))
		ARGS.Set("_gduration", h.Get("X-Forwarded-Duration"))
		if h.Get("X-Forwarded-User") == "" || h.Get("X-Forwarded-User") == "NULL" {
			return Err(401)
		}
		cipherSet := func(k, v string) {
			if k == role.Id_name && role.Id_cipher {
				id64, _ := strconv.ParseInt(v, 10, 64)
				ARGS.Set(k, strconv.FormatInt(int64(int_cipher.Decrypt(uint(id64), c.Secret)), 10))
				if k == role.Attributes[0] {
					ARGS.Set("_gid_cipher", v)
				}
			} else {
				ARGS.Set(k, v)
			}
		}
		cipherSet(role.Attributes[0], h.Get("X-Forwarded-User"))
		if len(role.Attributes) > 1 {
			groups := strings.Split(h.Get("X-Forwarded-Group"), "|")
			for i := 0; i < len(groups); i++ {
				cipherSet(role.Attributes[i+1], groups[i])
			}
		}
	}

	glog.Infof("(%d) %s\n", os.Getpid(), "role access control")
	if !is_admin && !Grep(actionHash["groups"], who) {
		return Err(401)
	}

	extra := make(url.Values)
	if !is_admin && ok {
		glog.Infof("(%d) %s\n", os.Getpid(), "check fk")
		err := self.assign_fk(who, fk, ARGS, extra)
		if err != nil {
			return err.(error)
		}
	}

	glog.Infof("(%d) %s\n", os.Getpid(), "preset")
	err := Invoke(filter, "Preset")[0].Interface()
	if err != nil {
		return err.(error)
	}

	glog.Infof("(%d) %s\n", os.Getpid(), "validation")
	validate, ok := actionHash["validate"]
	if ok {
		for _, field := range validate {
			if ARGS.Get(field) == "" {
				return Err(1092, field)
			}
		}
	}

	options, ok := actionHash["options"]
	if !ok || !Grep(options, "no_db") {
		Invoke0(model, "Set_db", self.Db)
	}

	nextextra := make(url.Values)
	glog.Infof("(%d) %s\n", os.Getpid(), "before")
	err = Invoke(filter, "Before", model, extra, nextextra)[0].Interface()
	if err != nil {
		return err.(error)
	}

	if !ok && !Grep(options, "no_method") {
		x := strings.ToUpper(action[:1]) + action[1:]
		glog.Infof("(%d) %s\n", os.Getpid(), "call model")
		err := Invoke(model, x, extra, nextextra)[0].Interface()
		if err != nil {
			return err.(error)
		}
		glog.Infof("(%d) %s\n", os.Getpid(), "call model OK")
	}

	if !is_admin && len(lists) > 0 {
		glog.Infof("(%d) %s\n", os.Getpid(), "fk tobe")
		self.assign_fk_tobe(who, fk, ARGS, lists)
	}

	glog.Infof("(%d) %s\n", os.Getpid(), "after")
	err = Invoke(filter, "After", model)[0].Interface()
	if err != nil {
		return err.(error)
	}

	glog.Infof("(%d) %s\n", os.Getpid(), "call blocks")
	err = c.Sendmail(lists, ARGS, other)
	if err != nil {
		return err.(error)
	}

	tmpl := &Tmpl{ARGS: ARGS, Lists: lists, Other: other, Success: true}
	chartag := c.Chartags[tag]
	if chartag.Case > 0 {
		glog.Infof("(%d) %s\n", os.Getpid(), "generate json")
		if ARGS.Get(role.Id_name) != "" && role.Id_cipher {
			ARGS.Set(role.Id_name, ARGS.Get("_gid_cipher"))
		}
		for k := range ARGS {
			if len(k) > 2 && k[0:2] == "_g" {
				ARGS.Del(k)
			}
		}
		b, eb := json.Marshal(tmpl)
		if eb != nil {
			return eb
		}
		base.Send_page(string(b))
		return nil
	}

	glog.Infof("(%d) call template\n", os.Getpid())
	other["Component"] = obj
	other["Tag"] = tag
	other["Role"] = who
	other["Action"] = action

	tmplname := action + "." + tag
	tmplfile := c.Template + "/" + who + "/" + obj + "/" + tmplname
	globfiles := c.Template + "/" + who + "/*." + tag
	T0 := template.New(tmplname).Option("missingkey=zero")
	var er error
	if T0, er = T0.ParseFiles(tmplfile); er == nil {
		if T0, er = T0.ParseGlob(globfiles); er == nil {
			glog.Infof("(%d) %s\n", os.Getpid(), "generate page")
			var output string
			if output, er = tmpl.Get_page(T0); er == nil {
				glog.Infof("(%d) %s\n", os.Getpid(), "sending page")
				base.Send_page(output)
			}
		}
	}
	return er
}

func (self *Controller) assign_fk(who string, fk []string, ARGS url.Values, extra url.Values) error {
	if fk == nil || self.C.Secret == "" {
		return nil
	}
	name := fk[0]
	if name == "" {
		return nil
	}

	value := ARGS.Get(name)
	if value == "" {
		return Err(1041)
	}
	extra.Set(name, value)
	roleid := ARGS.Get("_gid_name")
	if name == roleid {
		return nil
	}

	if fk[1] == "" {
		return Err(1054)
	}
	md5 := ARGS.Get(fk[1])
	if md5 == "" {
		return Err(1055)
	}

	stamp := ARGS.Get("_gwhen")
	value_roleid := ARGS.Get(roleid)
	if md5 != Digest(self.C.Secret, stamp, who, roleid, value_roleid, name, value) {
		return Err(1052)
	}
	if ARGS.Get("_gduration") != "" {
		gtime, _ := strconv.ParseInt(ARGS.Get("_gtime"), 10, 32)
		slast, _ := strconv.ParseInt(stamp, 10, 32)
		if gtime > slast {
			return Err(1053)
		}
	}

	return nil
}

func (self *Controller) fk_tobe(lists []map[string]interface{}, fk []string, stamp, who, roleid, value_roleid string) {
	if len(fk) <= 2 || fk[2] == "" || fk[2] == roleid || fk[3] == "" {
		return
	}
	name := fk[2]
	for _, item := range lists {
		item_name, ok := item[name]
		if !ok {
			continue
		}
		value := Interface2String(item_name)
		item[fk[3]] = Digest(self.C.Secret, stamp, who, roleid, value_roleid, name, value)
	}
	return
}

func (self *Controller) assign_fk_tobe(who string, fk0 []string, ARGS url.Values, lists []map[string]interface{}) error {
	if fk0 == nil || self.C.Secret == "" {
		return nil
	}
	roleid := ARGS.Get("_gid_name")

	stamp := ARGS.Get("_gwhen")
	value_roleid := ARGS.Get(roleid)

	fk := make([]string, len(fk0))
	copy(fk, fk0)

	self.fk_tobe(lists, fk, stamp, who, roleid, value_roleid)

	for len(fk) > 4 {
		fk = fk[3:]
		which := fk[1]
		if lists[0][which] == nil {
			return Err(1056)
		}
		for _, item := range lists {
			self.fk_tobe(item[which].([]map[string]interface{}), fk, stamp, who, roleid, value_roleid)
		}
	}
	return nil
}
