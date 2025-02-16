package match

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/genelet/winter/pzutil"
)

func TestPath(t *testing.T) {
	//	c := pzutil.NewConfig("../conf/gotest.conf")
	//
	for _, pubID := range []uint32{1, 2} {
		for _, siteID := range []uint32{33, 44} {
			for _, slotID := range []uint32{555, 666} {
				for _, w := range []uint16{700, 800} {
					for _, h := range []uint16{900, 1000} {
						for _, mime := range []string{"html", "js", "png", "json"} {
							sizeID := pzutil.GetSizeID(w, h)
							rpub, _ := RPub{pubID, siteID, slotID, sizeID}.Pack()
							r, _ := http.NewRequest("GET", "http://www.u2link.com:8000/pz/"+rpub+"."+mime+"?age=25", bytes.NewBuffer([]byte("")))
							status, incoming, adImp, _, _, err := GetPathIds(r)
							if err != nil {
								t.Fatal(err)
							}
							if status.Request != pzutil.IMPR || status.Source != pzutil.BROWSER {
								t.Errorf("%d<=>%d, %#v", pzutil.IMPR, pzutil.BROWSER, status)
							}
							if mime == "html" && status.Mime != pzutil.HTML {
								t.Errorf("%#v", incoming.Platform)
							}
							if len(adImp) != 1 {
								t.Errorf("%#v", adImp)
							}
							banner := adImp[0].Banner
							if pubID != adImp[0].PubID ||
								siteID != adImp[0].SiteID ||
								slotID != adImp[0].SlotID ||
								w != banner.Size[0] ||
								h != banner.Size[1] {
								t.Errorf("%#v", adImp[0])
							}
						}
					}
				}
			}
		}
	}

	for _, pubID := range []uint32{1, 2} {
		for _, siteID := range []uint32{33, 44} {
			for _, slotID := range []uint32{555, 666} {
				for _, w := range []uint16{700, 800} {
					for _, h := range []uint16{900, 1000} {
						for _, mime := range []string{"html", "js", "png", "json"} {
							sizeID := pzutil.GetSizeID(w, h)
							rpub, _ := RPub{pubID, siteID, slotID, sizeID}.Pack()
							r, _ := http.NewRequest("GET", "http://www.u2link.com:8000/cz/"+rpub+"."+mime+"?age=25", bytes.NewBuffer([]byte("")))
							status, incoming, adImp, clk, _, err := GetPathIds(r)
							if err != nil {
								t.Fatal(err)
							}
							if status.Request != pzutil.CLIC {
								t.Errorf("%#v", status)
							}
							if mime == "html" && status.Mime != pzutil.HTML {
								t.Errorf("%#v", incoming.Platform)
							}
							if len(adImp) != 1 {
								t.Errorf("%#v", adImp)
							}
							if clk.StartNano != 0 || clk.StartIP != 0 || clk.StartUa != 0 {
								t.Errorf("%v", clk)
							}
							banner := adImp[0].Banner
							if pubID != adImp[0].PubID ||
								siteID != adImp[0].SiteID ||
								slotID != adImp[0].SlotID ||
								w != banner.Size[0] ||
								h != banner.Size[1] {
								t.Errorf("%#v", adImp[0])
							}
						}
					}
				}
			}
		}
	}

	rpub, _ := RPub{1, 2, 3, pzutil.GetSizeID(4, 5)}.Pack()
	pid32, _ := Pid{11, 22, 33}.Pack()

	for _, advID := range []uint32{1, 2} {
		for _, campID := range []uint32{33, 44} {
			for _, itemID := range []uint32{555} {
				for _, creativeID := range []uint32{666} {
					for _, price := range []float32{7.77, 8.88} {
						for _, mime := range []string{"html", "js", "png", "json", "aaa", "bbb"} {
							radv, _ := RAdv{advID, campID, itemID, creativeID, price}.Pack()
							r, _ := http.NewRequest("GET", "http://www.u2link.com:8000/cz/"+pid32+"/"+rpub+"/"+radv+"."+mime+"?age=25", bytes.NewBuffer([]byte("")))
							status, incoming, adImp, clk, _, err := GetPathIds(r)
							if err != nil {
								t.Fatal(err)
							}
							if status.Request != pzutil.CLIC {
								t.Errorf("%#v", status)
							}
							if (mime == "aaa" || mime == "bbb") && status.Request != pzutil.CLIC {
								t.Errorf("%#v", status)
							}
							if (mime == "aaa" || mime == "bbb") && clk.Click != mime {
								t.Errorf("%#v", clk)
							}
							if (mime == "aaa" || mime == "bbb") && status.Mime != pzutil.UnknownMime {
								t.Errorf("%#v %#v", status.Mime, incoming.Platform)
							}
							if len(adImp) != 1 {
								t.Errorf("%#v", adImp)
							}
							if clk.StartNano != 11 || clk.StartIP != 22 || clk.StartUa != 33 {
								t.Errorf("%v", clk)
							}
							if adImp[0].PubID != 1 || adImp[0].SiteID != 2 || adImp[0].SlotID != 3 {
								t.Errorf("%#v", adImp[0])
							}
							if clk.AdvID != advID || clk.CampaignID != campID || clk.ItemID != itemID || clk.CreativeID != creativeID || clk.Price != price {
								t.Errorf("%#v", clk)
							}
						}
					}
				}
			}
		}
	}

	r, _ := http.NewRequest("POST", "http://www.u2link.com:8000/pz", bytes.NewBuffer([]byte(ADs)))
	status, incoming, adImp, _, _, err := GetPathIds(r)
	if err != nil {
		t.Fatal(err)
	}
	if status.Request != pzutil.IMPR || status.Source != pzutil.BROWSER {
		t.Errorf("%#v", status)
	}
	if len(adImp) != 3 {
		t.Errorf("%#v", adImp)
	}
	if incoming.Platform != "browser" {
		t.Errorf("%#v", incoming)
	}
	if adImp[0].Banner == nil {
		t.Errorf("%#v", adImp[0])
	}
	if adImp[1].Video == nil {
		t.Errorf("%#v", adImp[1])
	}
	if adImp[2].Native == nil {
		t.Errorf("%#v", adImp[2])
	}
	banner := adImp[0].Banner
	if adImp[0].PubID != 65536 ||
		adImp[0].SiteID != 65535 ||
		adImp[0].SlotID != 65536 ||
		banner.Size[0] != 300 ||
		banner.Size[1] != 250 {
		t.Errorf("%v", banner)
	}

}
