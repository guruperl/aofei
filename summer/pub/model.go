package pub

import (
	"net/url"
	"github.com/genelet/winter/summer"
)

type Model struct {
	summer.Model
}

func (self *Model)Dashboard(extra ...url.Values) error {
	return self.Edit(extra...)
}
