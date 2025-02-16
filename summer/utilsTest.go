package summer

import (
	"bytes"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"golang.org/x/net/publicsuffix"
)

func setCommon(r *http.Request, u *url.URL, jar *cookiejar.Jar) {
	uaStr := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36"
	r.Header.Set("User-Agent", uaStr)
	r.Header.Set("Content-Type", "application/json")
	r.RemoteAddr = "210.51.200.123:80"
	for _, cookie := range jar.Cookies(u) {
		r.AddCookie(cookie)
	}
}

func GetJar() *cookiejar.Jar {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		panic(err)
	}
	return jar
}

func GetNewRequest(in string, jar *cookiejar.Jar) *http.Request {
	u, err := url.Parse(in)
	if err != nil {
		panic(err)
	}

	r, err := http.NewRequest("GET", in, nil)
	if err != nil {
		panic(err)
	}
	setCommon(r, u, jar)
	return r
}

func GetPostRequest(in string, json string, jar *cookiejar.Jar) *http.Request {
	u, err := url.Parse(in)
	if err != nil {
		panic(err)
	}

	r, err := http.NewRequest("POST", in, bytes.NewBuffer([]byte(json)))
	if err != nil {
		panic(err)
	}
	setCommon(r, u, jar)
	return r
}
