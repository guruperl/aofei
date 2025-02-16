package pzutil

import (
    "testing"
)

func TestConfig(t *testing.T) {
	c := NewConfig("../conf/gotest.conf")
	if c.NatsURL != "nats://localhost:4222" {
		t.Errorf("%s wanted", c.NatsURL)
	}

    if c.Ucookie != "uid" {
        t.Errorf("%s, %s wanted", c.Ucookie, "uid")
    }
    if c.UcookieMaxAge != 15553000 {
        t.Errorf("%d, 15553000 wanted", c.UcookieMaxAge)
    }

	sample := c.Sizes[64225580]
	if sample.W != 980 || sample.H != 300 {
		t.Errorf("%#v", sample)
	}
}
