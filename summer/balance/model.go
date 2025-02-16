//advertiser's campaign 41
//advertiser's item 42

package balance

import (
	"net/url"
	"github.com/genelet/winter/summer"
)

type Model struct {
	summer.Model
}

func (self *Model)Topics(extra ...url.Values) error {
	//ARGS := self.ARGS
	entity_id := self.ProperValue("entity_id",extra[0])
	table     := self.ProperValue("table",extra[0])
	idname    := self.ProperValue("idname",extra[0])

	return self.Select_sql(self.LISTS,
`SELECT "total_balance_id" AS which, b.balance_id, limit_spend, limit_imp, limit_cli, current_spend, current_imp, current_cli
FROM adv_balance b
INNER JOIN `+table+` c ON (b.balance_id=c.total_balance_id)
WHERE c.`+idname+`=?
UNION
SELECT "daily_balance_id" AS which, b.balance_id, limit_spend, limit_imp, limit_cli, current_spend, current_imp, current_cli
FROM adv_balance b
INNER JOIN `+table+` c ON (b.balance_id=c.daily_balance_id)
WHERE c.`+idname+`=?`, entity_id, entity_id)
}

func (self *Model)Insert(extra ...url.Values) error {
	ARGS := self.ARGS
	which     := ARGS.Get("which")
	entity_id := ARGS.Get("entity_id")
	table     := ARGS.Get("table")
	idname    := ARGS.Get("idname")

	err := self.Model.Insert(extra...)
	if err != nil { return err }

	return self.Do_sql(
`UPDATE `+table+` SET `+which+`=? WHERE `+idname+`=?`, self.Last_id, entity_id)
}
