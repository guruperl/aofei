// Package dmp provides functions to interact with Redis for Dmp objects.
package dmp

import (
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
)

func GetRedisDmp(id []byte, p *pool.Pool) (*Dmp, error) {
	conn, err := p.Get()
	if err != nil {
		return nil, err
	}

	resp := conn.Cmd("GET", "USER:"+string(id))
	if resp.Err != nil {
		p.Put(conn)
		return nil, resp.Err
	}

	var data []byte
	if resp.IsType(redis.SimpleStr | redis.BulkStr) {
		data, err = resp.Bytes()
		p.Put(conn)
		if err != nil {
			return nil, err
		}
		//		return UnpackDmp(data)
		return UnpackDmpSimple(data)
	}

	p.Put(conn)
	return nil, nil
}

func (self *Dmp) SetRedis(id []byte, p *pool.Pool) error {
	//	data, err := self.Pack()
	data, err := self.PackSimple()
	if err != nil {
		return err
	}

	conn, err := p.Get()
	if err != nil {
		return err
	}

	resp := conn.Cmd("SET", "USER:"+string(id), data)
	p.Put(conn)
	return resp.Err
}
