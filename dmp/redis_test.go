package dmp;

import (
	"testing"
	"github.com/mediocregopher/radix.v2/pool"
)

func TestRedis(t *testing.T) {
	p, err := pool.New("tcp", "vm0:6379", 3)
	if err != nil {
		t.Fatal(err)
	}

	uid := "0123456789012345"
	dmp := GetDmpSample()
	data, err := dmp.Pack()
    if err != nil { t.Fatal(err) }

	conn, err := p.Get()
    if err != nil { t.Fatal(err) }

	resp := conn.Cmd("SET", "USER:"+uid, string(data))
	if resp.Err != nil {
		p.Put(conn)
		t.Errorf("%v", resp.Err)
	}

	dmp0, err := GetRedisDmp([]byte(uid), p)
	p.Empty();

	if err != nil {
		t.Errorf("%v",dmp0)
		t.Fatal(err)
	}
}
