package tencent

import (
	"time"
	"net/url"
	"net/http"
)

func GetClient(c *Config) *http.Client {
	client := &http.Client{}
	client.Timeout = time.Duration(c.Timeout)*time.Second
	return client
}

func Submit(c *Config, client *http.Client, url string, data url.Values) (*http.Response, error) {
	data.Set("dsp_id", c.Dsp_id)
	data.Set("token", c.Token)
	return client.PostForm(url, data)
}
