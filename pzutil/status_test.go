package pzutil

import (
	"testing"
)

func TestStatus(t *testing.T) {
	a := []bool{true, false}
	b := []bool{true, false}
	c := []bool{true, false}
	d := []TRequest{IMPR, REQS, CLIC, UnknownRequest}
	e := []TMime{JS, JSON, HTML, GIF, PNG}
	f := []TIDSource{SSPServer, IMEI, IDFA}
	g := []TSource{BROWSER, MOBILE, DSP, PREBID, AOFEI, SDK}
	h := []TStyle{BANNER, VIDEO, NATIVE, UnknownStyle}
	for _, x1 := range a {
		for _, x2 := range b {
			for _, x3 := range c {
				for _, x4 := range d {
					for _, x5 := range e {
						for _, x6 := range f {
							for _, x7 := range g {
								for _, x8 := range h {
									status := &Status{
										IsNew:      x1,
										IsDailyNew: x2,
										IsPSA:      x3,
										Request:    x4,
										Mime:       x5,
										IDSource:   x6,
										Source:     x7,
										Style:      x8,
									}
									pack := status.Pack()
									status0 := UnpackStatus(pack)
									if x1 != status0.IsNew ||
										x2 != status0.IsDailyNew ||
										x3 != status0.IsPSA ||
										x4 != status0.Request ||
										x5 != status0.Mime ||
										x6 != status0.IDSource ||
										x7 != status0.Source ||
										x8 != status0.Style {
										t.Errorf("%v %v", status, status0)
									}
								}
							}
						}
					}
				}
			}
		}
	}
}
