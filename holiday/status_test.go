package holiday

import (
	"testing"
)

func TestStatus(t *testing.T) {
	a := []bool{true, false}
	b := []bool{true, false}
	c := []bool{true, false}
	d := []R_TYPE{IMPR, REQS, CLIC, RequestUnknown}
	e := []M_TYPE{JSMime, JsonMime, H5Mime, ImageMime, VideoMime}
	f := []ID_SOURCE{SSPServer, AndroidID, IDFA}
	g := []T_SOURCE{Browser, MobileNative, DSP, AOFEI, PREBID}
	for _, x1 := range a {
		for _, x2 := range b {
			for _, x3 := range c {
				for _, x4 := range d {
					for _, x5 := range e {
						for _, x6 := range f {
							for _, x7 := range g {
								status := &Status{x1, x2, x3, x4, x5, x6, x7}
								pack := status.Pack()
								status0 := UnpackStatus(pack)
								if x1 != status0.IsUser ||
									x2 != status0.IsDevice ||
									x3 != status0.IsIp ||
									x4 != status0.RequestType ||
									x5 != status0.MimeType ||
									x6 != status0.IdSource ||
									x7 != status0.Source {
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
