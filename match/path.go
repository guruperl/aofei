package match

/*
import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/genelet/winter/pzutil"
	openrtb2 "github.com/prebid/openrtb/v20/openrtb2"
)

// GetPathIds returns status, incoming, adImps, clk, bid, error
// clicks:
// handler/IMP_ID32/rpub/radv.click
// handler/c/site_id/slot_id.click
//
// notification click, using NOTIFICATION
// handler/IMP_ID32/rpub/radv.gif
// handler/c/site_id/slot.gif
//
// click generated dynamically, without '.click'	"fmt"
// handler/IMP_ID32/rpub/radv
// handler/c/site_id/slot_id
//
// impressions:
// handler/site_id/slot_id.(png|html|js|json)
// new impressions:
// handler/rpub.(png|html|js|json)
func GetPathIds(r *http.Request) (pzutil.Status, *Incoming, []*AdImp, *Clk, *openrtb2.BidRequest, error) {
	errall := func(err error) (pzutil.Status, *Incoming, []*AdImp, *Clk, *openrtb2.BidRequest, error) {
		return pzutil.Status{}, nil, nil, nil, nil, err
	}

	var status pzutil.Status
	var pid Pid
	var pub RPub
	var adv RAdv
	var clk *Clk
	var err error

	path := r.URL.Path
	if err = r.ParseForm(); err != nil {
		return errall(err)
	}

	if r.Method == "POST" {
		switch path {
		case pzutil.BIDHandler:
			status.IsNew = true
			status.IsDailyNew = true
			status.IDSource = pzutil.EXCHANGE
			status.Source = pzutil.DSP
			status.Request = pzutil.REQS
			status.Mime = pzutil.JSON

			bid := &openrtb2.BidRequest{}
			decoder := json.NewDecoder(r.Body)
			if err = decoder.Decode(bid); err != nil {
				return errall(err)
			}
			return status, nil, nil, nil, bid, nil
		case pzutil.SSPHandler:
			status.Request = pzutil.IMPR
			status.Mime = pzutil.JSON

			incoming := new(Incoming)
			decoder := json.NewDecoder(r.Body)
			if err = decoder.Decode(incoming); err != nil {
				return errall(err)
			}
			adImps, err := incoming.Unpack()
			if err != nil {
				return errall(err)
			}
			status.Source = pzutil.StringToSource(incoming.Platform)
			return status, incoming, adImps, nil, nil, nil
		default:
			return errall(errors.New("handler not found in POST"))
		}
	}

	if r.Method != "GET" {
		return errall(errors.New("method not supported"))
	}

	// this is generic new impressions:
	// handler/rpub.(png|html|js|json)
	if strings.HasPrefix(path, pzutil.SSPHandler+"/") {
		two := strings.Split(strings.TrimPrefix(path, pzutil.SSPHandler+"/"), ".")
		if len(two) != 2 {
			return errall(errors.New("wrong mime type"))
		}
		pub, err := UnpackRPub(two[0])
		if err != nil {
			return errall(err)
		}
		status.Source = pzutil.BROWSER
		status.Request = pzutil.IMPR
		status.Mime = pzutil.StringToMime(two[1])
		w, h := pzutil.GetSizes(pub.SizeID)
		banner := &BannerType{Size: []uint16{w, h}}
		return status, &Incoming{Platform: "browser"}, []*AdImp{{RPub: pub, Banner: banner}}, nil, nil, nil
	}

	// this is aofei win
	// handler/REQS_ID32/rpub/radv.gif
	// or url genereted dynamically
	// handler/rpub.gif
	if strings.HasPrefix(path, pzutil.WIN+"/") {
		three := strings.Split(strings.TrimPrefix(path, pzutil.WIN+"/"), "/")
		if len(three) != 3 {
			return errall(errors.New("wrong aofei win format"))
		}
		status.IDSource = pzutil.EXCHANGE
		status.Source = pzutil.AOFEI
		status.Request = pzutil.IMPR
		pid, err = UnpackPid(three[0])
		if err != nil {
			return errall(err)
		}
		pub, err = UnpackRPub(three[1])
		if err != nil {
			return errall(err)
		}
		two := strings.Split(three[2], ".")
		adv, err = UnpackRAdv(two[0])
		if err != nil {
			return errall(err)
		}
		status.Mime = pzutil.StringToMime(two[1])
		w, h := pzutil.GetSizes(pub.SizeID)
		banner := &BannerType{Size: []uint16{w, h}}
		clk = &Clk{Pid: pid, RAdv: adv}
		return status, &Incoming{Platform: "aofei"}, []*AdImp{
			{
				RPub:   pub,
				Banner: banner,
			},
		}, clk, nil, nil
	}

	// clicks:
	// handler/IMP_ID32/rpub/radv.click
	// handler/rpub.click
	//
	// click url generated dynamically
	// handler/IMP_ID32/rpub/radv
	// handler/rpub

	if strings.HasPrefix(path, pzutil.CLK+"/") {
		status.Request = pzutil.CLIC
		clk = new(Clk)
		arr := strings.Split(strings.TrimPrefix(path, pzutil.CLK+"/"), "/")
		var two []string
		switch len(arr) {
		case 1:
			two = strings.Split(arr[0], ".")
			pub, err = UnpackRPub(two[0])
			if err != nil {
				return errall(err)
			}
			status.Mime = pzutil.StringToMime(two[1])
		case 3:
			clk.Pid, err = UnpackPid(arr[0])
			if err != nil {
				return errall(err)
			}
			pub, err = UnpackRPub(arr[1])
			if err != nil {
				return errall(err)
			}
			two = strings.SplitN(arr[2], ".", 2)
			clk.RAdv, err = UnpackRAdv(two[0])
			if err != nil {
				return errall(err)
			}
		default:
			return errall(errors.New("wrong click format"))
		}
		if len(two) == 2 {
			clk.Click, err = url.QueryUnescape(two[1])
			if err != nil {
				return errall(err)
			}
		}
		w, h := pzutil.GetSizes(pub.SizeID)
		banner := &BannerType{Size: []uint16{w, h}}
		return status, &Incoming{Platform: "browser"}, []*AdImp{{RPub: pub, Banner: banner}}, clk, nil, nil
	}

	return status, nil, nil, nil, nil, nil
}


// clicks:
// handler/IMP_ID32/rpub/radv.click
// handler/c/site_id/slot_id.click

// notification click, using NOTIFICATION
// handler/IMP_ID32/rpub/radv.gif
// handler/c/site_id/slot.gif

// click generated dynamically, without '.click'
// handler/IMP_ID32/rpub/radv
// handler/c/site_id/slot_id

// impressions:
// handler/site_id/slot_id.(png|html|js|json)
// new impressions:
// handler/rpub.(png|html|js|json)

func errall(err error) (pzutil.Status, *Incoming, []*AdImp, *Clk, *openrtb2.BidRequest, error) {
	return pzutil.Status{}, nil, nil, nil, nil, err
}

func GetPathIds(r *http.Request, c *pzutil.Config) (pzutil.Status, *Incoming, []*AdImp, *Clk, *openrtb2.BidRequest, error) {
	status := pzutil.Status{}
	var err error
	if err = r.ParseForm(); err != nil {
		return errall(err)
	}

	path := r.URL.Path
	if r.Method == "POST" {
		switch path {
		case c.Handle["dsp"]:
			status.Source = pzutil.DSP
			status.Request = pzutil.REQS

			bid := &openrtb2.BidRequest{}
			decoder := json.NewDecoder(r.Body)
			if err = decoder.Decode(bid); err != nil {
				return errall(err)
			}
			return status, nil, nil, nil, bid, nil
		case c.Handle["ssp"]:
			status.Request = pzutil.IMPR
			switch r.Header.Get("Content-Type") {
			case "application/json":
				status.Mime = pzutil.JSON
			default:
			}

			incoming := &Incoming{}
			decoder := json.NewDecoder(r.Body)
			if err = decoder.Decode(incomiGetPathIdsng); err != nil {
				return errall(err)
			}

			adImps, err := incoming.Unpack()
			if err != nil {
				return errall(err)
			}
			switch incoming.Platform {
			case "browser":
				status.Source = pzutil.BROWSER
			case "mobile":
				status.Source = pzutil.MOBILE
			case "sdk":
				status.Source = pzutil.SDK
			default:
			}
			return status, incoming, adImps, nil, nil, nil
		default:
			return errall(errors.New("handler not found in POST"))
		}
	}

	// new impressions:
	// handler/rpub.(png|html|js|json)
	pub := RPub{}
	if strings.HasPrefix(path, c.Handle["ssp"]+"/") {
		arr := strings.Split(path, "/")
		if len(arr) != 3 {
			return errall(errors.New("wrong impression format"))
		}
		two := strings.Split(arr[2], ".")
		if len(two) != 2 {
			return errall(errors.New("wrong mime type"))
		}
		if rpub, e := UnpackRPub(two[0]); e == nil {
			pub = rpub
		}
		status.Source = pzutil.BROWSER
		status.Request = pzutil.IMPR
		switch two[1] {
		case "json":
			status.Mime = pzutil.JSON
		case "html":
			status.Mime = pzutil.HTML
		case "js":
			status.Mime = pzutil.JS
		case "gif":
			status.Mime = pzutil.GIF
		case "png":
			status.Mime = pzutil.PNG
		default:
		}
		w, h := pzutil.GetSizes(pub.SizeID)
		banner := &BannerType{Size: []uint16{w, h}}
		return status, &Incoming{Platform: "browser"}, []*AdImp{{RPub: pub, Banner: banner}}, nil, nil, nil
	}

	// clicks:
	// handler/IMP_ID32/rpub/radv.click
	// handler/rpub.click

	// notification
	// handler/IMP_ID32/rpub/radv.gif
	// handler/rpub.gif

	// click url generated dynamically
	// handler/IMP_ID32/rpub/radv
	// handler/rpub

	clk := &Clk{}
	if strings.HasPrefix(path, c.Handle["click"]+"/") {
		arr := strings.Split(path, "/")
		status.Request = pzutil.CLIC
		status.Source = pzutil.BROWSER
		var two []string
		if len(arr) == 3 {
			two = strings.Split(arr[2], ".")
			if rpub, e := UnpackRPub(two[0]); e == nil {
				pub = rpub
			}
		} else if len(arr) == 5 {
			two = strings.SplitN(arr[4], ".", 2)
			if rpub, e := UnpackRPub(arr[3]); e == nil {
				pub = rpub
			}
			if radv, e := UnpackRAdv(two[0]); e == nil {
				clk.RAdv = radv
			}
			if pid, e := UnpackPid(arr[2]); e == nil {
				clk.Pid = pid
			}
		} else {
			return errall(errors.New("wrong click format"))
		}
		if len(two) == 2 {
			switch two[1] {
			case "json":
				status.Mime = pzutil.JSON
			case "html":
				status.Mime = pzutil.HTML
			case "js":
				status.Mime = pzutil.JS
			case "gif":
				status.Mime = pzutil.GIF
			case "png":
				status.Mime = pzutil.PNG
			default:
				clk.Click, _ = url.QueryUnescape(two[1])
			}
		}
		w, h := pzutil.GetSizes(pub.SizeID)
		banner := &BannerType{Size: []uint16{w, h}}
		return status, &Incoming{Platform: "browser"}, []*AdImp{{RPub: pub, Banner: banner}}, clk, nil, nil
	}

	return status, nil, nil, nil, nil, nil
}
*/
