package site

import (
	"net/url"

	"github.com/genelet/winter/summer"
)

type Model struct {
	summer.Model
}

func (self *Model) Startnew(extra ...url.Values) error {
	return self.ProcessAfter("startnew", extra...)
}
