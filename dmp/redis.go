// Package dmp provides functions to interact with Redis for Dmp objects.
package dmp

import (
	"context"

	"github.com/mediocregopher/radix/v4"
)

func GetRedisDmp(ctx context.Context, conn radix.Client, id string) (*Dmp, error) {
	var data []byte
	err := conn.Do(ctx, radix.Cmd(&data, "GET", "USER:"+id))
	if err != nil {
		return nil, err
	}

	return UnpackDmpSimple(data)
}

func (self *Dmp) SetRedis(ctx context.Context, conn radix.Client, id string) error {
	//	data, err := self.Pack()
	data, err := self.PackSimple()
	if err != nil {
		return err
	}

	return conn.Do(ctx, radix.Cmd(nil, "SET", "USER:"+id, string(data)))
}
