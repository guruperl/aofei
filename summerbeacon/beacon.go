package summerbeacon

import (
	"github.com/genelet/winter/genelet"
)

type Beacon struct {
	genelet.Beacon
}

func NewBeacon(role string) (*Beacon, error) {
	controller, err := NewController("/srv/aofei/winter/conf/summer.json")
	if err != nil {
		return nil, err
	}

	tag := "json"
	headers := map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}, "Cookie": {"go_probe=/"}}

	p, err := genelet.NewBeacon(*controller, role, tag, headers)
	return &Beacon{*p}, err
}
