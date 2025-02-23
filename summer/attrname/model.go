package attrname

import (
	"net/url"

	"github.com/genelet/winter/summer"
)

type Model struct {
	summer.Model
}

func (self *Model) Insert(extra ...url.Values) error {
	ARGS := self.ARGS
	if err := self.Model.Insert(extra...); err != nil {
		return err
	}

	for _, v := range ARGS["values"] {
		err := self.Do_sql(
			`INSERT INTO adv_attrvalue (attrname_id, value) VALUES (?,?)`,
			ARGS.Get("attrname_id"), v)
		if err != nil {
			return err
		}
	}

	return nil
}

func (self *Model) Topics(extra ...url.Values) error {
	return self.Select_sql(self.LISTS,
		`SELECT attrname_id, attrname, 
GROUP_CONCAT(attrvalue_id) AS id,
GROUP_CONCAT(value) AS value
FROM (
	SELECT i.attrname_id, i.attrname, v.attrvalue_id, v.value
	FROM adv_attrname i
	LEFT JOIN adv_attrvalue v USING (attrname_id)
	WHERE i.adv_id=?
) tmp
GROUP BY attrname_id`, self.ARGS.Get("adv_id"))
}
