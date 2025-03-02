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

type Controller struct {
	C       *Config
	DB      *sql.DB
	Models  map[string]interface{}
	Filters map[string]interface{}
	Storage map[string]interface{}
}

func (self *Controller) staticPage(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.URL.Path, `..`) {
		http.NotFound(w, r)
	}

	c := self.C
	url := r.URL

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

	   		role := form.Get(c.RoleName)
	   		tag := form.Get(c.TagName)
	   		component := form.Get(c.ComponentName)

	   		role := form.Get(c.RoleName)
	   		tag := form.Get(c.TagName)
	   		component := form.Get(c.ComponentName)
	   		if pattern.Case==STATIC || pattern.Case==CACHE {
	   			if (role=="" || role==c.Pubrole) {
	   				break
	   			}
	   			if (tag=="") {
	   glog.Infof("(%d) %s\n", os.Getpid(), "static file matched not good")
	   				http.NotFound(w,r)
	   				return
	   			}
	   			base := &Base{C:c, R:r, W:w, RoleValue:role, ChartagValue:tag}
	   			gate := NewGate(*base)
	   			err := gate.Forbid()
	   			if (err != nil) {
	   glog.Infof("(%d) %s\n", os.Getpid(), "static file asks for login")
	   				self.loginPage(base)
	   				return
	   			}
	   			break
	   		} else {
	   			if role=="" || tag=="" || component=="" {
	   				http.NotFound(w,r)
	   			} else {
	   				url.Path = c.Script+"/"+role+"/"+tag+"/"+component
	   				form.Del(c.RoleName)
	   				form.Del(c.TagName)
	   				form.Del(c.ComponentName)
	   				url.RawQuery = form.Encode()
	   			}
	   			return
	   		}
	   	}

	*/
	http.ServeFile(w, r, c.DocumentRoot+url.Path)
}

func (self *Controller) loginPage(base *Base) {
	c := self.C
	uri := base.R.Form.Get(c.GoURIName)

	provider := base.R.Form.Get(c.ProviderName)
	glog.Infof("(%d) %s %s\n", os.Getpid(), "provider? ", provider)
	if provider == "" {
		provider = base.GetProvider()
		if provider == "" {
			http.NotFound(base.W, base.R)
			return
		}
	}

	db := self.DB

	var err error
	if Grep(c.Oauth2s, provider) {
		ticket := NewOauth2(*base, db, uri, provider)
		glog.Infof("(%d) %s %s\n", os.Getpid(), "oauth2 uses: ", provider)
		err = ticket.Handler_login()
		uri = ticket.Uri // use the same vriable for the targeting uri
	} else if Grep(c.Oauth1s, provider) {
		ticket := NewOauth1(*base, db, uri, provider)
		glog.Infof("(%d) %s %s\n", os.Getpid(), "oauth1 uses: ", provider)
		err = ticket.Handler_login()
		uri = ticket.Uri // use the same vriable for the targeting uri
	} else {
		glog.Infof("(%d) %s %s\n", os.Getpid(), "login uses: ", provider)
		ticket := NewProcedure(*base, db, uri, provider)
		err = ticket.Handler()
		uri = ticket.Uri // use the same vriable for the targeting uri
	}
	if err == nil {
		return
	}
	if base.ChartagValue == "json" {
		base.SendPage(c.Chartags[base.ChartagValue].CallChallenge())
		return
	}

	glog.Infof("(%d) %s %#v\n", os.Getpid(), "ticket returns error: ", err)
	gerr := err.(Gerror)
	if gerr.Code < 1000 {
		base.SendStatusPage(gerr.Code, gerr.Errstr)
		return
	}
	issuer := (c.Roles[base.RoleValue]).Issuers[provider]
	fn := c.LoginName + "." + base.ChartagValue
	T := template.New(fn).Option("missingkey=zero")
	T, err = T.ParseFiles(c.Template + "/" + base.RoleValue + "/" + c.LoginName + "." + base.ChartagValue)
	if err == nil {
		errstr := c.Errors[strconv.Itoa(gerr.Code)]
		if errstr == "" {
			errstr = gerr.Error()
		}
		var buffer bytes.Buffer
		err = T.Execute(&buffer, map[string]interface{}{
			"LoginName": c.LoginName, "GoURIName": c.GoURIName,
			"Errorstr": errstr, "GoURI": uri,
			"Login": issuer.Credential[0], "Password": issuer.Credential[1]})
		base.SendNocache(buffer.String())
	}
	if err != nil {
		http.Error(base.W, err.Error(), 500)
	}
}

func checkForm(r *http.Request, dir string) error {
	reader, err := r.MultipartReader()
	if reader != nil && err != nil {
		return err
	}

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
			switch s := value.(type) {
			case []string:
				for _, v := range s {
					form.Add(key, v)
				}
			case []uint8:
				form.Add(key, string(s))
			case []interface{}:
				for _, v := range s {
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

			fieldName := part.FormName()
			fileName := part.FileName()
			if fileName == "" {
				scanner := bufio.NewScanner(part)
				scanner.Scan()
				form.Add(fieldName, scanner.Text())
			} else {
				fullname := dir + "/" + fileName
				dst, err := os.Create(fullname)
				if err != nil {
					return err
				}
				defer dst.Close()
				if _, err := io.Copy(dst, part); err != nil {
					return err
				}
				form.Add(fieldName, fileName)
			}
			part.Close()
		}
	}

	return nil
}

func (self *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := self.C
	length := len(c.Script)

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
	if r.Method == "OPTIONS" || r.Method == "HEAD" {
		w.WriteHeader(200)
		return
	}

	if !strings.HasPrefix(r.URL.Path, c.Script+"/") {
		glog.Infof("(%d) %s %s, [static]\n\n", os.Getpid(), r.Method, r.URL.Path)
		self.staticPage(w, r)
		return
	}

	glog.Infof("(%d) %s %s, %v\n", os.Getpid(), r.Method, r.URL.Path, r.URL.Query())
	var methodFound bool
	for k := range c.DefaultActions {
		if k == r.Method {
			methodFound = true
			break
		}
	}
	if !methodFound {
		http.Error(w, "The http method is not supported", http.StatusMethodNotAllowed)
		return
	}

	pathInfo := strings.Split(r.URL.Path[length+1:], "/")
	if len(pathInfo) == 4 {
		r.Header.Add("X-Forwarded-ID", pathInfo[3])
	} else if len(pathInfo) != 3 {
		glog.Infof("(%d) %s\n", os.Getpid(), "not genelet url")
		http.Error(w, "Bad Request", 400)
		return
	}

	chartag, ok := c.Chartags[pathInfo[1]]
	if !ok {
		glog.Infof("(%d) %s\n", os.Getpid(), "check chartag")
		http.Error(w, "Bad Request", 400)
		return
	}

	base := &Base{C: c, W: w, R: r, RoleValue: pathInfo[0], ChartagValue: pathInfo[1]}
	gate := NewGate(*base)
	obj := pathInfo[2]

	glog.Infof("(%d) %s\n", os.Getpid(), "parse form")
	err := checkForm(r, c.UploadDir)
	if err != nil {
		http.Error(w, "Bad Request: "+err.Error(), 400)
		return
	}

	glog.Infof("(%d) %s\n", os.Getpid(), "if role "+base.RoleValue+", is defined or public")
	_, ok = c.Roles[base.RoleValue]
	if !ok && (gate.RoleValue != c.Pubrole) {
		http.NotFound(w, r)
		return
	}

	glog.Infof("(%d) %s %s\n", os.Getpid(), "object is", obj)
	if obj == c.LoginName || Grep(c.Oauth2s, obj) || Grep(c.Oauth1s, obj) {
		glog.Infof("(%d) %s %s\n", os.Getpid(), "start login for", obj)
		if obj != c.LoginName {
			r.Form.Set(c.ProviderName, obj)
		}
		self.loginPage(base)
		glog.Infof("(%d) %s\n\n", os.Getpid(), "end login ...")
		return
	} else if obj == c.LogoutName {
		glog.Infof("(%d) %s\n", os.Getpid(), "start logout")
		err = gate.HandleLogout()
		if err != nil {
			gate.SendStatusPage(err.(Gerror).Code, err.(Gerror).Errstr)
			glog.Infof("(%d) %s\n\n", os.Getpid(), "end logout ...")
		}
		return
	}

	if gate.RoleValue != c.Pubrole {
		err = gate.Forbid()
		if err != nil {
			glog.Infof("(%d) forbidden ... %d : %s\n\n", os.Getpid(), err.(Gerror).Code, err.(Gerror).Errstr)
			gate.SendStatusPage(err.(Gerror).Code, err.(Gerror).Errstr)
			return
		}
	}

	glog.Infof("(%d) %s\n", os.Getpid(), "starting genelet handler ...")
	err = self.Handle(obj, *base, r.Method)
	if err != nil {
		switch g := err.(type) {
		case Gerror:
			if g.Code == 303 {
				base.SendStatusPage(303, err.Error())
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

		tmplfile := c.Template + "/" + base.RoleValue + "/error." + base.ChartagValue
		T0, er := template.ParseFiles(tmplfile)
		if er != nil {
			base.SendPage(addJSON(chartag.Case, er.Error()))
			return
		}
		var buffer bytes.Buffer
		er = T0.Execute(&buffer, err)
		if er != nil {
			base.SendPage(addJSON(chartag.Case, er.Error()))
		} else if err.(Gerror).Code < 1000 {
			base.SendStatusPage(err.(Gerror).Code, buffer.String())
		} else {
			base.SendPage(buffer.String())
		}
	}
	glog.Infof("(%d) %s\n\n", os.Getpid(), "genelet handler ended.")
}

func addJSON(c int8, msg string) string {
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
	who := base.RoleValue
	tag := base.ChartagValue

	c := self.C
	r := base.R
	ARGS := r.Form

	ARGS.Set("_gobj", obj)
	ARGS.Set("_gtime", strconv.FormatInt(time.Now().Unix(), 10))

	lists := make([]map[string]interface{}, 0)
	other := make(map[string]interface{})
	glog.Infof("(%d) %s\n", os.Getpid(), "set defaults")
	Invoke0(model, "SetDefaults", ARGS, &lists, &other, self.Storage)

	action := ARGS.Get(c.ActionName)
	if action == "" {
		action = c.DefaultActions[method]
		if method == "GET" && r.Header.Get("X-Forwarded-ID") != "" {
			action = c.DefaultActions["GET_item"]
		}
	}
	if r.Header.Get("X-Forwarded-ID") != "" {
		ARGS.Set("_gid_url", r.Header.Get("X-Forwarded-ID"))
	}
	glog.Infof("(%d) %s\n", os.Getpid(), "get action: "+action)
	Invoke0(filter, "Set_all", base, action, obj, &other)
	ret := Invoke(filter, "GetAll")
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

	role, ok := c.Roles[base.RoleValue]
	var isAdmin bool
	if ok {
		glog.Infof("(%d) %s\n", os.Getpid(), "parsing role's ARGS")
		ARGS.Set("_gid_name", role.Id_name)
		ARGS.Set("_gtype_id", strconv.Itoa(role.Type_id))
		if role.Is_admin {
			ARGS.Set("_gadmin", "1")
			isAdmin = true
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
	if !isAdmin && !Grep(actionHash["groups"], who) {
		return Err(401)
	}

	extra := make(url.Values)
	if !isAdmin && ok {
		glog.Infof("(%d) %s\n", os.Getpid(), "check fk")
		err := self.assignFK(who, fk, ARGS, extra)
		if err != nil {
			return err
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
		Invoke0(model, "SetDB", self.DB)
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

	if !isAdmin && len(lists) > 0 {
		glog.Infof("(%d) %s\n", os.Getpid(), "fk tobe")
		self.assignFKTobe(who, fk, ARGS, lists)
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
		base.SendPage(string(b))
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
				base.SendPage(output)
			}
		}
	}
	return er
}

func (self *Controller) assignFK(who string, fk []string, ARGS url.Values, extra url.Values) error {
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
	valueRoleID := ARGS.Get(roleid)
	if md5 != Digest(self.C.Secret, stamp, who, roleid, valueRoleID, name, value) {
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

func (self *Controller) fkTobe(lists []map[string]interface{}, fk []string, stamp, who, roleid, valueRoleID string) {
	if len(fk) <= 2 || fk[2] == "" || fk[2] == roleid || fk[3] == "" {
		return
	}
	name := fk[2]
	for _, item := range lists {
		itemName, ok := item[name]
		if !ok {
			continue
		}
		value := Interface2String(itemName)
		item[fk[3]] = Digest(self.C.Secret, stamp, who, roleid, valueRoleID, name, value)
	}
}

func (self *Controller) assignFKTobe(who string, fk0 []string, ARGS url.Values, lists []map[string]interface{}) error {
	if fk0 == nil || self.C.Secret == "" {
		return nil
	}
	roleid := ARGS.Get("_gid_name")

	stamp := ARGS.Get("_gwhen")
	valueRoleID := ARGS.Get(roleid)

	fk := make([]string, len(fk0))
	copy(fk, fk0)

	self.fkTobe(lists, fk, stamp, who, roleid, valueRoleID)

	for len(fk) > 4 {
		fk = fk[3:]
		which := fk[1]
		if lists[0][which] == nil {
			return Err(1056)
		}
		for _, item := range lists {
			self.fkTobe(item[which].([]map[string]interface{}), fk, stamp, who, roleid, valueRoleID)
		}
	}
	return nil
}
