package dsp

import (
	"testing"
)

func TestConfig(t *testing.T) {
	c, err := NewConfig("sample.json")
	if err != nil {
		panic(err)
	}
	if c.DocumentRoot != "/tmp/www" ||
		c.ServerURL != "http://localhost" ||
		c.Ips != "qq-pz.dat" ||
		c.ConnectArray[0] != "taosSql" ||
		c.ConnectArray[1] != "root:taosdata@/tcp(127.0.0.1:0)/holiday?parseTime=false" {
		t.Errorf("%s", c.DocumentRoot)
		t.Errorf("%s", c.ServerURL)
		t.Errorf("%s", c.Ips)
		t.Errorf("%s", c.Ips)
		t.Errorf("%v", c.ConnectArray)
	}
}
