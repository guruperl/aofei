package address

import (
	"net/url"
	"github.com/genelet/winter/summer"
)

type Model struct {
	summer.Model
}

func (self *Model)States(extra ...url.Values) error {
	return self.Select_sql(self.LISTS,
`SELECT state_id, state_name
FROM def_state
INNER JOIN def_country USING (country_id)
WHERE def_country.active="Yes"`)
}
