package pub

import (
	"net/url"
    "github.com/genelet/winter/summerbeacon"
)

func NewBeacon(role string) (*Beacon, error) {
	bc, err := summerbeacon.NewBeacon(role)
	if err != nil { return nil, err }
	return &Beacon{*bc}, nil
}

type Beacon struct {
	summerbeacon.Beacon
}

func (self *Beacon) GET(args string) error {
	return self.GetMock("pub", args)
}

func (self *Beacon) POST(args url.Values) error {
	return self.PostMock("pub", args)
}
