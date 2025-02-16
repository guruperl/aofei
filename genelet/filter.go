package genelet

import (
	"net/url"
	"database/sql"
//	"github.com/golang/glog"
)

type Filter struct {
	Base
	Action		string
	Component	string
	Actions		map[string]map[string][]string
	Fks			map[string][]string
	OTHER		*map[string]interface{}
}

func (self *Filter)Initialize(comp *Component) {
	self.Actions = comp.Actions
	self.Fks = comp.Fks
}

func (self *Filter)Set_all(base Base, action string, component string, other *map[string]interface{}) {
	self.Base = base
	self.Action = action
	self.Component = component
	self.OTHER = other
}

func (self *Filter)Get_all() (map[string][]string, []string) {
	actionHash, found := self.Actions[self.Action]
	if (!found) {
		return nil, nil
	}

	if (self.Fks == nil) {
		return actionHash, nil
	}
	fk, found := self.Fks[self.Role_value]
	if (found) {
		return actionHash, fk
	}
	return actionHash, nil
}

func (self *Filter)Set_login_as(role_value, login, uri string, db *sql.DB) error {
	base := &Base{C:self.Base.C, W:self.Base.W, R:self.Base.R, Role_value:role_value, Chartag_value:self.Base.Chartag_value}
	provider := base.Get_provider()
	ticket := New_Procedure(*base, db, uri, provider)
	if err := ticket.Authenticate_as(login); err != nil {
		return err
	}
	fields := ticket.Get_attributes()
	signed := ticket.Signature(fields...)
	role := self.C.Roles[role_value]
	self.Set_cookie(role.Surface, signed, role.Max_age)

	return Gerror{303, uri}
}

func (self *Filter)Preset() error {
    return nil
}

func (self *Filter)Before(model *Model, extra url.Values, nextextra url.Values)  error {
    return nil
}

func (self *Filter)After(model *Model) error {
    return nil
}
