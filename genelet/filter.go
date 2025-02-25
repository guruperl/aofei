package genelet

import (
	"database/sql"
	"net/url"
	// "github.com/golang/glog"
)

type Filter struct {
	Base
	Action    string
	Component string
	Actions   map[string]map[string][]string
	Fks       map[string][]string
	OTHER     *map[string]interface{}
}

func (self *Filter) Initialize(comp *Component) {
	self.Actions = comp.Actions
	self.Fks = comp.Fks
}

func (self *Filter) Set_all(base Base, action string, component string, other *map[string]interface{}) {
	self.Base = base
	self.Action = action
	self.Component = component
	self.OTHER = other
}

func (self *Filter) GetAll() (map[string][]string, []string) {
	actionHash, found := self.Actions[self.Action]
	if !found {
		return nil, nil
	}

	if self.Fks == nil {
		return actionHash, nil
	}
	fk, found := self.Fks[self.RoleValue]
	if found {
		return actionHash, fk
	}
	return actionHash, nil
}

func (self *Filter) Set_login_as(role_value, login, uri string, db *sql.DB) error {
	base := &Base{C: self.Base.C, W: self.Base.W, R: self.Base.R, RoleValue: role_value, ChartagValue: self.Base.ChartagValue}
	provider := base.GetProvider()
	ticket := NewProcedure(*base, db, uri, provider)
	if err := ticket.Authenticate_as(login); err != nil {
		return err
	}
	fields := ticket.GetAttributes()
	signed := ticket.Signature(fields...)
	role := self.C.Roles[role_value]
	self.SetCookie(role.Surface, signed, role.MaxAge)

	return Gerror{303, uri}
}

func (self *Filter) Preset() error {
	return nil
}

func (self *Filter) Before(model *Model, extra url.Values, nextextra url.Values) error {
	return nil
}

func (self *Filter) After(model *Model) error {
	return nil
}
