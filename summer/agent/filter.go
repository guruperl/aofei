package agent

import (
    "crypto/sha1"
    "encoding/hex"
	"net/url"
	"github.com/genelet/winter/summer"
)

type Filter struct {
	summer.Filter
}

func (self *Filter)Preset() error {
    if err := self.Filter.Preset(); err != nil { return err }

    ARGS := self.R.Form
    action := self.Action
    who := self.Role_value

    if who=="admin" && action=="insert" {
		h := sha1.New()
		h.Write([]byte(ARGS.Get("passwd")+ARGS.Get("login")))
		ARGS.Set("passwd", hex.EncodeToString(h.Sum(nil)))
	}

	return nil
}

func (self *Filter)Before(model *Model, extra url.Values, nextextra url.Values)  error {
	if err := self.Filter.Before(&model.Model, extra, nextextra); err != nil { return err }

	return nil
}

func (self *Filter)After(model *Model) error {
    if err := self.Filter.After(&model.Model); err != nil { return err }

	return nil
}
