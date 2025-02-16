package slot

import (
	"net/url"
	"github.com/genelet/winter/summer"
)

type Model struct {
	summer.Model
}

func (self *Model)Startnew(extra ...url.Values) error {
    return self.Process_after("startnew", extra...)
}
