// Package summer provides models and methods for handling various operations
// related to addresses, user accounts, and administrative tasks.
package summer

import (
	"net/url"
	"strconv"

	"github.com/genelet/winter/genelet"
)

type Model struct {
	genelet.Model
}

var addressTables []string = []string{"pub", "adv", "pay_cc", "pay_cheque", "testing"}

func (self *Model) Dashboard(extra ...url.Values) error {
	return self.Topics(extra...)
}

func (self *Model) Insert(extra ...url.Values) error {
	if !Grep(addressTables, self.Current_table) {
		return self.Model.Insert(extra...)
	}

	err := self.Call_once(map[string]interface{}{"model": "address", "action": "insert"})
	if err != nil {
		return err
	}
	other := *self.OTHER
	data := other["address_insert"].([]map[string]interface{})
	extra[0].Set("address_id", data[0]["address_id"].(string))
	err = self.Model.Insert(extra...)
	if err != nil {
		return err
	}

	lists := *self.LISTS
	for _, field := range []string{"company", "street", "city", "state_id", "zip", "country_id", "contact", "contact_email", "phone", "fax", "url"} {
		lists[0][field] = data[0][field]
	}
	return nil
}

func (self *Model) Edit(extra ...url.Values) error {
	if !Grep(addressTables, self.Current_table) {
		return self.Model.Edit(extra...)
	}

	err := self.Model.Edit(extra...)
	if err != nil {
		return err
	}
	lists := *self.LISTS
	return self.Get_sql(lists[0],
		"SELECT * FROM add_address WHERE address_id=?", lists[0]["address_id"])
}

func (self *Model) Update(extra ...url.Values) error {
	if Grep(addressTables, self.Current_table) {
		hash := make(map[string]interface{})
		err := self.Get_sql(hash,
			`SELECT address_id FROM `+self.Current_table+` WHERE `+self.Current_key+`=?`,
			self.ARGS.Get(self.Current_key))
		if err == nil {
			self.ARGS.Set("address_id", strconv.FormatInt(hash["address_id"].(int64), 10))
			err = self.Call_once(map[string]interface{}{"model": "address", "action": "update"})
		}
		if err != nil {
			return err
		}
	}
	return self.Model.Update(extra...)
}

func (self *Model) Activate(extra ...url.Values) error {
	id := self.Current_key
	ARGS := self.ARGS

	return self.Do_sql(
		`UPDATE `+self.Current_table+` SET active='Yes' WHERE `+id+`=? AND email=?`,
		ARGS.Get(id), ARGS.Get("email"))
}

func (self *Model) Retrieve(extra ...url.Values) error {
	id := self.Current_key

	return self.Select_sql(self.LISTS,
		`SELECT `+id+`, email, firstname, lastname FROM `+self.Current_table+`
WHERE email=? AND active IN ("New", "Yes")`, self.ARGS.Get("email"))
}

func (self *Model) Resetpass(extra ...url.Values) error {
	id := self.Current_key
	ARGS := self.ARGS

	return self.Do_sql(
		`UPDATE `+self.Current_table+` SET passwd=?, active='Yes' WHERE `+id+`=?`,
		ARGS.Get("passwd"), ARGS.Get(id))
}

func (self *Model) Updatepass(extra ...url.Values) error {
	id := self.Current_key
	ARGS := self.ARGS
	return self.Do_sql(
		`UPDATE  `+self.Current_table+` SET passwd=?
WHERE `+id+`=? AND passwd=SHA1(CONCAT(?, email))`,
		ARGS.Get("passwd"), ARGS.Get(id), ARGS.Get("passwd_old"))
}

func (self *Model) CleanupLogin(extra ...url.Values) error {
	return self.Do_sql(
		`DELETE FROM `+self.Current_table+`_ip
WHERE email=? AND ret='fail'
AND (UNIX_TIMESTAMP(updated) >= (UNIX_TIMESTAMP(NOW())-24*3600))`,
		self.ARGS.Get("email"))
}

func (self *Model) ChangeEmailAdmin(extra ...url.Values) error {
	id := self.Current_key
	ARGS := self.ARGS
	err := self.Existing(self.Current_table, "email", ARGS.Get("email"))
	if err != nil {
		return err
	}

	return self.Do_sql(
		`UPDATE `+self.Current_table+` SET email=? WHERE `+id+`=?`,
		ARGS.Get("email"), ARGS.Get(id))
}

func (self *Model) ChangePasswdAdmin(extra ...url.Values) error {
	table := self.Current_table
	id := self.Current_key
	ARGS := self.ARGS

	return self.Do_sql(
		`UPDATE `+table+` SET passwd=SHA1(?) WHERE `+id+`=?`,
		ARGS.Get("passwd"), ARGS.Get(id))
}
