package dmp

import (
	"context"
	"testing"

	"github.com/mediocregopher/radix/v4"
)

func TestRedis(t *testing.T) {
	cfg := radix.PoolConfig{
		Dialer: radix.Dialer{
			AuthPass: "zH8zkBqBcw",
		},
	}
	ctx := context.Background()
	conn, err := cfg.New(ctx, "tcp", "r-t4nfucwzprzrmsqrnn.redis.singapore.rds.aliyuncs.com:6379")
	if err != nil {
		t.Fatal(err)
	}

	uid := "0123456789012345"
	dmp := GetDmpSample()
	err = dmp.SetRedis(ctx, conn, uid)
	if err != nil {
		t.Fatal(err)
	}

	dmp0, err := GetRedisDmp(ctx, conn, uid)
	if err != nil {
		t.Fatal(err)
	}

	t.Errorf("%v", dmp)
	t.Errorf("%v", dmp0)
}
