package ssp

import (
	"github.com/genelet/winter/dmp"
	"github.com/genelet/winter/match"
	"github.com/genelet/winter/pzutil"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
)

func errp(p *pool.Pool, conn *redis.Client, err error) (*dmp.Dmp, *match.Site, error) {
	p.Put(conn)
	return errRedAll(err)
}
func errRedAll(err error) (*dmp.Dmp, *match.Site, error) {
	return nil, nil, err
}

func (self *Controller) RedisGetPublisher(siteID uint32, dmpStr string) (*dmp.Dmp, *match.Site, error) {
	c := self.C
	p := self.Redis
	db := self.Db
	conn, err := p.Get()
	if err != nil {
		return errRedAll(err)
	}

	isSite := false

	var dmmp *dmp.Dmp
	var site *match.Site

	keySite := pzutil.GetKeyName(c.SITE, siteID)
	keyUser := "DMP:" + dmpStr
	conn.Cmd("DEL", keySite)
	resp := conn.Cmd("MGET", keySite, keyUser)
	//glog.Infof("%v\n", resp.Err)
	if resp.Err != nil {
		return errp(p, conn, resp.Err)
	}
	if resp.IsType(redis.Array) {
		arr, err := resp.Array()
		if err != nil {
			return errp(p, conn, err)
		}
		if arr[0].IsType(redis.Str) {
			dSite, err := arr[0].Bytes()
			if err == nil {
				//glog.Infof("site from REDIS\n")
				site, err = match.UnpackSite(dSite)
			}
			if err != nil {
				return errp(p, conn, err)
			}
			isSite = true
		}
		if arr[1].IsType(redis.Str) {
			dDmp, err := arr[1].Bytes()
			if err == nil {
				//glog.Infof("dmp from REDIS\n")
				dmmp, err = dmp.UnpackDmpSimple(dDmp)
			}
			if err != nil {
				return errp(p, conn, err)
			}
		}
	}
	if !isSite {
		//glog.Infof("site from Database\n")
		site, err = match.DBGetSite(db, siteID)
		if err != nil {
			return errp(p, conn, err)
		}
		data, err := site.Pack()
		if err != nil {
			return errp(p, conn, err)
		}
		resp = conn.Cmd("SET", keySite, data)
		if resp.Err != nil {
			return errp(p, conn, resp.Err)
		}
	}

	p.Put(conn)
	return dmmp, site, nil
}

func errpSlot(p *pool.Pool, conn *redis.Client, err error) ([]*match.Slot, error) {
	p.Put(conn)
	return errRedSlot(err)
}
func errRedSlot(err error) ([]*match.Slot, error) {
	return nil, err
}

func (self *Controller) RedisGetSlots(adImps []*match.AdImp) ([]*match.Slot, error) {
	c := self.C
	p := self.Redis
	db := self.Db
	conn, err := p.Get()
	if err != nil {
		return errRedSlot(err)
	}

	n := len(adImps)

	slots := make([]*match.Slot, n)
	ids := make([]uint32, n)
	names := make([]interface{}, n+1)
	names[0] = c.SLOT
	for i, imp := range adImps {
		ids[i] = imp.SlotID
		names[i+1] = pzutil.IDStr(ids[i])
		conn.Cmd("HDEL", c.SLOT, names[i+1])
	}

	resp := conn.Cmd("HMGET", names...)
	//glog.Infof("%v\n", resp.Err)
	if resp.Err != nil {
		return errpSlot(p, conn, resp.Err)
	}

	if resp.IsType(redis.Array) {
		arr, err := resp.Array()
		if err != nil {
			return errpSlot(p, conn, err)
		}
		for i := 0; i < n; i++ {
			var slot *match.Slot
			if arr[i].IsType(redis.Str) {
				//glog.Infof("Item from REDIS\n")
				dslot, err := arr[i].Bytes()
				if err != nil {
					return errpSlot(p, conn, err)
				}
				slot, err = match.UnpackSlot(dslot)
				if err != nil {
					return errpSlot(p, conn, err)
				}
			} else {
				slot, err = match.DBGetNWeights(db, ids[i])
				//slot, err = match.DBMakeNWeights(db, ids[i])
				if err != nil {
					return errpSlot(p, conn, err)
				}
				data, err := slot.Pack()
				if err != nil {
					return errpSlot(p, conn, err)
				}
				resp = conn.Cmd("HSET", c.SLOT, names[i+1], data)
				if resp.Err != nil {
					return errpSlot(p, conn, resp.Err)
				}
			}
			slots[i] = slot
		}
	}

	p.Put(conn)
	return slots, nil
}

func errPool(p *pool.Pool, conn *redis.Client, err error) ([]*match.Audience, error) {
	p.Put(conn)
	return errAll(err)
}
func errAll(err error) ([]*match.Audience, error) {
	return nil, err
}

func (self *Controller) RedisGetAudiences(ids []uint32) ([]*match.Audience, error) {
	c := self.C
	p := self.Redis
	db := self.Db
	conn, err := p.Get()
	if err != nil {
		return errAll(err)
	}

	n := len(ids)
	if n == 0 {
		return nil, nil
	}
	audiences := make([]*match.Audience, n)

	names := make([]interface{}, n+1)
	names[0] = c.AUDIENCE
	for i := 0; i < n; i++ {
		name := pzutil.IDStr(ids[i])
		conn.Cmd("HDEL", c.AUDIENCE, name)
		names[i+1] = name
	}
	resp := conn.Cmd("HMGET", names...)
	if resp.Err != nil {
		return errPool(p, conn, resp.Err)
	}
	if resp.IsType(redis.Array) {
		arr, err := resp.Array()
		if err != nil {
			return errPool(p, conn, err)
		}
		for i := 0; i < n; i++ {
			if arr[i].IsType(redis.Str) {
				data, err := arr[i].Bytes()
				if err != nil {
					return errPool(p, conn, err)
				}
				//glog.Infof("Audience from REDIS %d\n", i)
				if len(data) > 1 { // see the follow one blank " " HSET
					audience, err := match.UnpackAudience(data)
					if err != nil {
						return errPool(p, conn, err)
					}
					audiences[i] = audience
				}
			} else {
				//glog.Infof("Audience from Database %d\n", i)
				audience, err := match.DBGetAudience(db, ids[i])
				if err != nil {
					return errPool(p, conn, err)
				}
				//glog.Infof("database audience: %v\n", audience)
				if audience == nil {
					resp = conn.Cmd("HSET", c.AUDIENCE, names[i+1], []byte(" "))
				} else {
					audiences[i] = audience
					data, err := audience.Pack()
					if err != nil {
						return errPool(p, conn, err)
					}
					resp = conn.Cmd("HSET", c.AUDIENCE, names[i+1], data)
				}
				if resp.Err != nil {
					return errPool(p, conn, resp.Err)
				}
			}
		}
	}

	/*
		resp = conn.Cmd("HMGET", name2...)
		const longForm = "20060102150405"
		loc := user.FullTime.Location()
		if resp.Err == nil {
			if arr, err := resp.Array(); err == nil {
				for i, item := range arr {
					if str, err := item.Str(); err == nil { // err means none exist
						t, _ := time.ParseInLocation(longForm, str, loc)
						if t.Sub(user.FullTime) > 0 {
							if audiences[i]==nil {
								audiences[i] = &match.Audience{Paused:true}
							} else {
								audiences[i].Paused = true
							}
						}
					}
				}
			}
		}
	*/

	p.Put(conn)
	return audiences, nil
}

func ep(p *pool.Pool, conn *redis.Client, err error) (*match.Item, error) {
	p.Put(conn)
	return eall(err)
}
func eall(err error) (*match.Item, error) {
	return nil, err
}

func (self *Controller) RedisGetItem(user *User, id uint32) (*match.Item, error) {
	c := self.C
	p := self.Redis
	db := self.Db
	conn, err := p.Get()
	if err != nil {
		return eall(err)
	}

	var item *match.Item
	conn.Cmd("HDEL", c.ITEM, pzutil.IDStr(id))
	resp := conn.Cmd("HGET", c.ITEM, pzutil.IDStr(id))
	if resp.Err != nil {
		return ep(p, conn, resp.Err)
	}
	if resp.IsType(redis.Str) {
		//glog.Infof("Item from REDIS\n")
		if data, err := resp.Bytes(); err == nil {
			item, err = match.UnpackItem(data)
			if err != nil {
				return ep(p, conn, err)
			}
		} else {
			return ep(p, conn, err)
		}
	} else {
		//glog.Infof("Item from Database\n")
		item, err = match.DBGetItem(db, id)
		if err != nil {
			return ep(p, conn, err)
		}
		data, err := item.Pack()
		if err != nil {
			return ep(p, conn, err)
		}
		resp = conn.Cmd("HSET", c.ITEM, pzutil.IDStr(id), data)
		if resp.Err != nil {
			return ep(p, conn, resp.Err)
		}
	}

	p.Put(conn)
	return item, nil
}

func esp(p *pool.Pool, conn *redis.Client, err error) ([]*match.Item, error) {
	p.Put(conn)
	return esall(err)
}
func esall(err error) ([]*match.Item, error) {
	return nil, err
}

func (self *Controller) RedisGetItems(user *User, ids []uint32) ([]*match.Item, error) {
	c := self.C
	p := self.Redis
	db := self.Db
	conn, err := p.Get()
	if err != nil {
		return esall(err)
	}

	n := len(ids)
	if n == 0 {
		return nil, nil
	}
	items := make([]*match.Item, n)

	names := make([]interface{}, n+1)
	names[0] = c.ITEM
	for i := 0; i < n; i++ {
		name := pzutil.IDStr(ids[i])
		conn.Cmd("HDEL", c.ITEM, name)
		names[i+1] = name
	}
	resp := conn.Cmd("HMGET", names...)
	if resp.Err != nil {
		return esp(p, conn, resp.Err)
	}

	if resp.IsType(redis.Array) {
		arr, err := resp.Array()
		if err != nil {
			return esp(p, conn, err)
		}
		for i := 0; i < n; i++ {
			if arr[i].IsType(redis.Str) {
				data, err := arr[i].Bytes()
				if err != nil {
					return esp(p, conn, err)
				}
				//glog.Infof("Item from REDIS %d", i)
				if len(data) > 1 { // see the follow one blank " " HSET
					item, err := match.UnpackItem(data)
					if err != nil {
						return esp(p, conn, err)
					}
					items[i] = item
				}
			} else {
				//glog.Infof("Item from Database %d", i)
				item, err := match.DBGetItem(db, ids[i])
				if err != nil {
					return esp(p, conn, err)
				}
				//glog.Infof("Database item: %v", item)
				if item == nil {
					resp = conn.Cmd("HSET", c.ITEM, names[i+1], []byte(" "))
				} else {
					items[i] = item
					data, err := item.Pack()
					if err != nil {
						return esp(p, conn, err)
					}
					resp = conn.Cmd("HSET", c.ITEM, names[i+1], data)
				}
				if resp.Err != nil {
					return esp(p, conn, resp.Err)
				}
			}
		}
	}

	p.Put(conn)
	return items, nil
}
