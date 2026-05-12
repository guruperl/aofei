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

var addressTables []string = []string{"pub", "adv", "mid_dsp", "pay_cc", "pay_cheque", "testing"}

func (self *Model) Dashboard(extra ...url.Values) error {
	return self.Topics(extra...)
}

func (self *Model) Insert(extra ...url.Values) error {
	if !Grep(addressTables, self.CurrentTable) {
		return self.Model.Insert(extra...)
	}

	err := self.CallOnce(map[string]interface{}{"model": "address", "action": "insert"})
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
	if !Grep(addressTables, self.CurrentTable) {
		return self.Model.Edit(extra...)
	}

	err := self.Model.Edit(extra...)
	if err != nil {
		return err
	}
	lists := *self.LISTS
	return self.GetSQL(lists[0],
		"SELECT * FROM add_address WHERE address_id=?", lists[0]["address_id"])
}

func (self *Model) Update(extra ...url.Values) error {
	if Grep(addressTables, self.CurrentTable) {
		hash := make(map[string]interface{})
		err := self.GetSQL(hash,
			`SELECT address_id FROM `+self.CurrentTable+` WHERE `+self.CurrentKey+`=?`,
			self.ARGS.Get(self.CurrentKey))
		if err == nil {
			self.ARGS.Set("address_id", strconv.FormatInt(hash["address_id"].(int64), 10))
			err = self.CallOnce(map[string]interface{}{"model": "address", "action": "update"})
		}
		if err != nil {
			return err
		}
	}
	return self.Model.Update(extra...)
}

func (self *Model) Activate(extra ...url.Values) error {
	id := self.CurrentKey
	ARGS := self.ARGS

	return self.DoSQL(
		`UPDATE `+self.CurrentTable+` SET active='Yes' WHERE `+id+`=? AND email=?`,
		ARGS.Get(id), ARGS.Get("email"))
}

func (self *Model) Retrieve(extra ...url.Values) error {
	id := self.CurrentKey

	return self.SelectSQL(self.LISTS,
		`SELECT `+id+`, email, firstname, lastname FROM `+self.CurrentTable+`
WHERE email=? AND active IN ("New", "Yes")`, self.ARGS.Get("email"))
}

func (self *Model) Resetpass(extra ...url.Values) error {
	id := self.CurrentKey
	ARGS := self.ARGS

	return self.DoSQL(
		`UPDATE `+self.CurrentTable+` SET passwd=?, active='Yes' WHERE `+id+`=?`,
		ARGS.Get("passwd"), ARGS.Get(id))
}

func (self *Model) Updatepass(extra ...url.Values) error {
	id := self.CurrentKey
	ARGS := self.ARGS
	return self.DoSQL(
		`UPDATE  `+self.CurrentTable+` SET passwd=?
WHERE `+id+`=? AND passwd=SHA1(CONCAT(?, email))`,
		ARGS.Get("passwd"), ARGS.Get(id), ARGS.Get("passwd_old"))
}

func (self *Model) CleanupLogin(extra ...url.Values) error {
	return self.DoSQL(
		`DELETE FROM `+self.CurrentTable+`_ip
WHERE email=? AND ret='fail'
AND (UNIX_TIMESTAMP(updated) >= (UNIX_TIMESTAMP(NOW())-24*3600))`,
		self.ARGS.Get("email"))
}

func (self *Model) ChangeEmailAdmin(extra ...url.Values) error {
	id := self.CurrentKey
	ARGS := self.ARGS
	err := self.Existing(self.CurrentTable, "email", ARGS.Get("email"))
	if err != nil {
		return err
	}

	return self.DoSQL(
		`UPDATE `+self.CurrentTable+` SET email=? WHERE `+id+`=?`,
		ARGS.Get("email"), ARGS.Get(id))
}

func (self *Model) ChangePasswdAdmin(extra ...url.Values) error {
	table := self.CurrentTable
	id := self.CurrentKey
	ARGS := self.ARGS

	return self.DoSQL(
		`UPDATE `+table+` SET passwd=SHA1(?) WHERE `+id+`=?`,
		ARGS.Get("passwd"), ARGS.Get(id))
}
