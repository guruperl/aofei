// this simulates traffic to the dsp, as if it were an adx. It then sends a win or loss, 50% of each.
// If it wins, it sends an impression tracker as well, plus the 50% chance of a click URL.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/genelet/winter/dsp"
	"github.com/genelet/winter/match"
	"github.com/prebid/openrtb/v20/openrtb2"

	_ "github.com/go-sql-driver/mysql"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: winloss --s=dsp_config --bid=address [win|loss|imp|clk]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var sConf string
var address string
var how string

func init() {
	flag.Usage = usage
	flag.StringVar(&sConf, "s", os.Getenv("AOFEI"), "DSP Config")
	flag.StringVar(&address, "bid", "/bid/exchange.example.test", "bid url")
}

func main() {
	flag.Parse()
	if flag.NArg() >= 1 {
		how = flag.Arg(0)
	} else {
		how = "random"
	}

	ctx := context.Background()
	sc, err := dsp.NewControllerWithOptions(ctx, sConf, dsp.WithoutNATS(), dsp.WithoutMaxMind())
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()

	buf := bytes.NewBuffer(jsonBid)
	host := strings.Replace(sc.C.ServerURL, "https", "http", 1)
	req, err := http.NewRequest(http.MethodPost, host+address, buf)
	if err != nil {
		log.Fatal(err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNoContent {
		log.Printf("status code: %d", res.StatusCode)
		os.Exit(0)
	} else if res.StatusCode != http.StatusOK {
		log.Fatalf("status code: %d", res.StatusCode)
	}

	bidResponse := new(openrtb2.BidResponse)
	err = json.NewDecoder(res.Body).Decode(bidResponse)
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}

	trackers, err := extractBidResponseTrackers(bidResponse)
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)
	var rspns, impRspns, clkRspns *http.Response
	switch how {
	case "win":
		log.Printf("win %s", trackers.nurl.String())
		rspns, err = http.Get(trackers.nurl.String())
		if err == nil {
			time.Sleep(1 * time.Second)
			log.Printf("impression %s", trackers.imp.String())
			impRspns, err = http.Get(trackers.imp.String())
		}
	case "loss":
		log.Printf("loss %s", trackers.lurl.String())
		rspns, err = http.Get(trackers.lurl.String())
	case "imp":
		log.Printf("impression %s", trackers.imp.String())
		impRspns, err = http.Get(trackers.imp.String())
	case "clk":
		log.Printf("clicking %s", trackers.click.String())
		clkRspns, err = http.Get(trackers.click.String())
	default:
		if rand.Intn(10) < 5 {
			if rspns, err = http.Get(trackers.nurl.String()); err == nil {
				time.Sleep(1 * time.Second)
				if impRspns, err = http.Get(trackers.imp.String()); err == nil && rand.Intn(10) < 5 {
					time.Sleep(2 * time.Second)
					clkRspns, err = http.Get(trackers.click.String())
				}
			}
		} else {
			rspns, err = http.Get(trackers.lurl.String())
		}
	}
	if err != nil {
		log.Fatal(err)
	}

	if rspns != nil {
		rspns.Body.Close()
	}
	if impRspns != nil {
		impRspns.Body.Close()
	}
	if clkRspns != nil {
		clkRspns.Body.Close()
	}
}

type bidResponseTrackers struct {
	nurl  fmt.Stringer
	lurl  fmt.Stringer
	imp   fmt.Stringer
	click fmt.Stringer
}

func extractBidResponseTrackers(bidResponse *openrtb2.BidResponse) (*bidResponseTrackers, error) {
	if bidResponse == nil {
		return nil, fmt.Errorf("bid response is nil")
	}
	if len(bidResponse.SeatBid) == 0 {
		return nil, fmt.Errorf("bid response has no seat bids")
	}
	if len(bidResponse.SeatBid[0].Bid) == 0 {
		return nil, fmt.Errorf("bid response has no bids")
	}

	bid := bidResponse.SeatBid[0].Bid[0]
	if bid.NURL == "" {
		return nil, fmt.Errorf("bid response missing nurl")
	}
	nurlObject, err := dsp.UnpackURLString(bid.NURL, bidResponse)
	if err != nil {
		return nil, fmt.Errorf("unpack nurl: %w", err)
	}

	if bid.LURL == "" {
		return nil, fmt.Errorf("bid response missing lurl")
	}
	lurlObject, err := dsp.UnpackURLString(bid.LURL, bidResponse)
	if err != nil {
		return nil, fmt.Errorf("unpack lurl: %w", err)
	}

	if bid.AdM == "" {
		return nil, fmt.Errorf("bid response missing native markup")
	}
	native, err := match.UnpackAdm([]byte(bid.AdM))
	if err != nil {
		return nil, fmt.Errorf("unpack native markup: %w", err)
	}
	if native == nil {
		return nil, fmt.Errorf("bid response native markup has no native object")
	}
	if len(native.ImpTrackers) == 0 {
		return nil, fmt.Errorf("bid response native markup missing impression trackers")
	}
	impObject, err := dsp.UnpackURLString(native.ImpTrackers[0])
	if err != nil {
		return nil, fmt.Errorf("unpack impression tracker: %w", err)
	}

	if native.Link == nil {
		return nil, fmt.Errorf("bid response native markup missing link")
	}
	clickURL := native.Link.URL
	if clickURL == "" && len(native.Link.Clicktrackers) > 0 {
		clickURL = native.Link.Clicktrackers[0]
	}
	if clickURL == "" {
		return nil, fmt.Errorf("bid response native markup missing click URL")
	}
	clickObject, err := dsp.UnpackURLString(clickURL)
	if err != nil {
		return nil, fmt.Errorf("unpack click URL: %w", err)
	}

	return &bidResponseTrackers{nurl: nurlObject, lurl: lurlObject, imp: impObject, click: clickObject}, nil
}

var jsonBid = []byte(`
{
    "id": "a4fbe69285880f67f3f33c2465bfb8ab",
    "imp": [
        {
            "id": "5c6e168cd0494bf8b71d7d92030bda18",
            "native": {
                "request": "{\"native\":{\"ver\":\"1.1\",\"assets\":[{\"id\":1,\"required\":1,\"title\":{\"len\":100}},{\"id\":2,\"required\":1,\"img\":{\"type\":1,\"wmin\":64,\"hmin\":64}}]}}",
                "ver": "1.1"
            },
            "tagid": "1004202",
            "bidfloor": 0.01,
            "bidfloorcur": "USD",
            "secure": 0,
            "ext": {
                "adCounts": 10
            }
        }
    ],
    "app": {
        "id": "1004078",
        "bundle": "com.example.publisher"
    },
    "device": {
        "ua": "Mozilla/5.0 (Linux; Android 13; RMX3511 Build/TP1A.220624.014; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/131.0.6778.260 Mobile Safari/537.36",
        "geo": {
            "country": "RUS"
        },
        "lmt": 0,
        "ip": "178.214.246.172",
        "devicetype": 1,
        "make": "OPPO",
        "model": "rmx3511",
        "os": "Android",
        "osv": "13",
        "connectiontype": 2,
        "ifa": "8e9e9c7e-fb9f-46ae-bd9b-d29fe784bebf"
    },
    "user": {
        "id": "",
        "gender": "O"
    },
    "test": 0,
    "at": 1,
    "tmax": 700,
    "allimps": 0,
    "cur": [
        "USD"
    ],
    "ext": {
        "request_domain": "exchange.example.test",
        "x-openrtb-version": "2.5"
    }
}`)
