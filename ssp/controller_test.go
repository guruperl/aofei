package ssp

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/genelet/winter/ipsearch"
	"github.com/genelet/winter/match"
	"github.com/genelet/winter/pzutil"
	_ "github.com/go-sql-driver/mysql"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/nats-io/nats.go"
	"golang.org/x/net/publicsuffix"
)

func getSample(t *testing.T) *Controller {
	c := pzutil.NewConfig("../conf/gotest.conf")

	ips, err := ipsearch.LoadIPData(c.Ips)
	if err != nil {
		t.Fatalf("error opening Ip file: %v", err)
	}

	nc, err := nats.Connect(c.NatsURL)
	if err != nil {
		t.Fatalf("error nats: %v", err)
	}

	p, err := pool.New(c.Redis.Network, c.Redis.Addr, c.Redis.Size)
	if err != nil {
		t.Fatalf("error opening Redis pool: %v", err)
	}

	db, err := sql.Open(c.ConnectArray[0], c.ConnectArray[1])
	if err != nil {
		t.Fatalf("error opening Mysql handler: %v", err)
	}

	return &Controller{C: c, Ips: ips, Redis: p, Db: db, Nc: nc}
}

func TestStatus(t *testing.T) {
	handler := getSample(t)

	hash := map[string]int{
		"/../":           http.StatusBadRequest,
		"/index.html":    http.StatusMovedPermanently,
		"/abc.html":      http.StatusNotFound,
		"/pz/i/b/3.html": http.StatusBadRequest,
		"/pz/i/2/c.html": http.StatusBadRequest,
		"/pz/i/2.d":      http.StatusBadRequest,
		"/pz/":           http.StatusBadRequest,
	}

	for in, out := range hash {
		req, err := http.NewRequest("GET", in, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.RemoteAddr = "www.pzadx.com:443"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != out {
			t.Errorf("%s status code: got %v want %v. Body: %s", req.URL.Path, w.Code, out, w.Body.String())
		}
	}
}

func TestURLWrong(t *testing.T) {
	handler := getSample(t)
	hash := map[string]string{
		"/pz/3.html": "illegal base64 data at input byte 0",
		"/pz/c.d":    "illegal base64 data at input byte 0",
	}

	var uastr = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36"
	for in, out := range hash {
		req, err := http.NewRequest("GET", in, nil)
		req.Header.Set("User-Agent", uastr)
		req.RemoteAddr = "www.pzadx.com:80"
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		msg := w.Body.String()
		if !strings.Contains(msg, out) {
			t.Errorf("%s msg for %s: %s", in, out, msg)
		}
	}
}

/*
type mySuffixList struct {
}
func (self mySuffixList)PublicSuffix(domain string) string {
	return ""
}
func (self mySuffixList)String() string {
	return "test only"
}
*/

func setCommon(r *http.Request, u *url.URL, jar *cookiejar.Jar) {
	uastr := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36"
	r.Header.Set("User-Agent", uastr)
	r.Header.Set("Content-Type", "application/json")
	r.RemoteAddr = "210.51.200.123:80"
	for _, cookie := range jar.Cookies(u) {
		r.AddCookie(cookie)
	}
}

func getNewRequest(in string, u *url.URL, jar *cookiejar.Jar) *http.Request {
	r, err := http.NewRequest("GET", in, nil)
	if err != nil {
		panic(err)
	}
	setCommon(r, u, jar)
	return r
}

/*
func getPostRequest(in string, u *url.URL, json string, jar *cookiejar.Jar) *http.Request {
	r, err := http.NewRequest("POST", in, bytes.NewBuffer([]byte(json)))
	if err != nil {
		panic(err)
	}
	setCommon(r, u, jar)
	return r
}
*/

func TestController(t *testing.T) {
	control := getSample(t)

	current := time.Now()
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		t.Fatal(err)
	}

	rpub := match.RPub{
		PubID:  1,
		SiteID: 2,
		SlotID: 3,
		SizeID: 3,
	}
	pack, _ := rpub.Pack()
	hash := map[string]string{
		"http://site.002/pz/" + pack + ".html": `<iframe frameborder=0 src="data:text/html; charset=UTF-8,ads" width='0' height='3'></iframe>`,
	}

	for i := 0; i < 70; i++ {
		for in, out := range hash {
			u, err := url.Parse(in)
			if err != nil {
				t.Fatal(err)
			}
			req := getNewRequest(in, u, jar)
			w := httptest.NewRecorder()
			control.ServeHTTP(w, req)
			resp := w.Result()
			if cookies := resp.Cookies(); cookies != nil {
				jar.SetCookies(u, cookies)
				for _, cookie := range cookies {
					if cookie.Name == "i" {
						icaps, err := match.UnpackFcaps(current, cookie.Value)
						if err != nil {
							t.Fatal(err)
						}
						n := 0
						s := ""
						for cid, icap := range icaps {
							n += int(icap.Total)
							s += fmt.Sprintf("%d:%d,", cid, icap.Total)
						}
						//t.Errorf("%d => %d, %s", i, n, s)
						if i < 53 && i+1 != n {
							t.Errorf("%d => %d, %s", i, n, s)
						}
					}
				}
			}
			msg := w.Body.String()
			if w.Code != 200 || (i <= 53 && msg != out) || (i > 53 && !strings.Contains(msg, "psa")) {
				t.Errorf("%d:%v-%v", i, w.Code, msg)
			}
		}
	}

	// referers

	jar, err = cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		t.Fatal(err)
	}

	hash = map[string]string{
		"http://site.003/pz/" + pack + ".html": "bad domain\n",
	}

	for in, out := range hash {
		u, err := url.Parse(in)
		if err != nil {
			t.Fatal(err)
		}
		req := getNewRequest(in, u, jar)
		w := httptest.NewRecorder()
		control.ServeHTTP(w, req)
		msg := w.Body.String()
		if w.Code != 400 || out != msg {
			t.Errorf("-%s-%d-%s-", out, w.Code, msg)
		}
	}
	fmt.Println("Remember to load db and sample.sql, then go test summer and go test summer/weight, before go test match and go test ssp !")

}
