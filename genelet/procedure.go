package genelet

// In_pars and Out_pars must match exactly what needed in procedure and 
// the names in Out_pars should follow those in Attributes otherwise it
// will not be reagconized
import (
	"strings"
	"net/url"
	"github.com/golang/glog"
	"database/sql"
)

type Procedure struct {
	Db *sql.DB
	Ticket
}

func New_Procedure(base Base, db *sql.DB, uri string, provider string) *Procedure {
    a := new(Procedure)
    a.CGI = a
    a.Base = base
    a.Db = db
	a.Uri = uri
	a.Provider = provider
	return a
}

func (self *Procedure)Run_sql(call_name string, in_vals []interface{}) error {
	role := self.C.Roles[self.Role_value]
	issuer := role.Issuers[self.Provider]
	if (issuer.Screen & 1) !=0 {in_vals= append(in_vals, Ip2int(self.Get_ip()))}
	if (issuer.Screen & 2) !=0 {in_vals= append(in_vals, self.Uri)}
	//	if (issuer.Screen & 4) !=0 {in_vals= append(in_vals, self.Get_ua())}
	//	if (issuer.Screen & 8) !=0 {in_vals= append(in_vals, self.Get_referer())}
	out_pars := issuer.Out_pars
	if out_pars == nil {
		out_pars = role.Attributes
	}

	self.Out_hash = make(map[string]interface{})
	var err error
	dbi := &DBI{Db:self.Db}
	if strings.ToLower(call_name[0:7])=="select " {
		err = dbi.Get_sql_label(self.Out_hash, call_name, out_pars, in_vals...)
	} else {
		err = dbi.Do_proc(self.Out_hash, out_pars, call_name, in_vals...)
	}
	if err != nil {
glog.Infof("%v\n", out_pars)
glog.Infof("%v\n", call_name)
glog.Infof("%v\n", in_vals)
glog.Infof("%v\n", err)
		return Err(1036, err.Error())
	}

	return nil
}

func (self *Procedure) Authenticate(login, passwd string) error {
	role := self.C.Roles[self.Role_value]
	issuer := role.Issuers[self.Provider]
	return self.Run_sql(issuer.Sql, []interface{}{login, passwd})
}

func (self *Procedure) Authenticate_as(login string) error {
	role := self.C.Roles[self.Role_value]
	issuer := role.Issuers[self.Provider]
	return self.Run_sql(issuer.Sql_as, []interface{}{login})
}

func (self *Procedure)Callback_address() string {
	http := "http"
	if (self.R.TLS != nil) {
		http += "s"
	}
	return http + "://" + self.R.Host + self.C.Script + "/" + self.Role_value + "/" + self.Chartag_value + "/" + self.Provider + "?" + self.C.Go_uri_name + "=" + url.QueryEscape(self.Uri)
}

func (self *Procedure)Fill_provider(back map[string]interface{}) error {
	role := self.C.Roles[self.Role_value]
	issuer := role.Issuers[self.Provider]
	in_vals := make([]interface{},0)
	for _, par := range issuer.In_pars {
		if val, ok := back[par]; ok {
			in_vals = append(in_vals, val)
		} else {
			in_vals = append(in_vals, "")
		}
	}

	if err := self.Run_sql(issuer.Sql, in_vals); err != nil { return err }

	for _, key := range role.Attributes {
		if _, ok := self.Out_hash[key]; !ok {
			if out, ok := back[key]; ok {
				self.Out_hash[key] = out
			}
		}
	}

	return nil
}
