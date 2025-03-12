package ssp

/*
import (
	"context"

	"github.com/genelet/winter/dmp"
	"github.com/genelet/winter/match"
	"github.com/genelet/winter/pzutil"
	"github.com/mediocregopher/radix/v4"
)

func (self *Controller) RedisGetSite(ctx context.Context, siteID uint32) (*match.Site, error) {
	c := self.C
	db := self.DB
	conn := self.Redis
	keySite := pzutil.GetKeyName(c.SITE, siteID)

	var data []byte
	err := conn.Do(ctx, radix.Cmd(&data, "GET", keySite))
	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		return match.UnpackSite(data)
	}

	site, err := match.DBGetSite(db, siteID)
	if err != nil {
		return nil, err
	}
	data, err = site.Pack()
	if err != nil {
		return nil, err
	}
	err = conn.Do(ctx, radix.Cmd(nil, "SET", keySite, string(data)))
	return site, err
}

func (self *Controller) RedisGetDmp(ctx context.Context, dmpStr string) (*dmp.Dmp, error) {
	conn := self.Redis
	keyUser := "DMP:" + dmpStr
	var data []byte
	err := conn.Do(ctx, radix.Cmd(&data, "GET", keyUser))
	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		return dmp.UnpackDmpSimple(data)
	}
	return nil, nil
}

func (self *Controller) RedisGetPublisher(ctx context.Context, siteID uint32, dmpStr string) (*dmp.Dmp, *match.Site, error) {
	site, err := self.RedisGetSite(ctx, siteID)
	if err == nil {
		dmp, err := self.RedisGetDmp(ctx, dmpStr)
		return dmp, site, err
	}
	return nil, nil, err
}

func (self *Controller) RedisGetSlots(ctx context.Context, ids []uint32) ([]*match.Slot, error) {
	c := self.C
	conn := self.Redis
	db := self.DB

	n := len(ids)
	if n == 0 {
		return nil, nil
	}

	hash := map[string]string{}
	items := make([]*match.Slot, n)

	names := []string{c.SLOT}
	for i := 0; i < n; i++ {
		names = append(names, pzutil.IDStr(ids[i]))
	}
	err := conn.Do(ctx, radix.Cmd(&hash, "HMGET", names...))
	if err != nil {
		return nil, err
	}

	backHash := map[string]string{}
	for i := 0; i < n; i++ {
		name := names[i+1]
		var item *match.Slot
		itemStr, ok := hash[name]
		if !ok || itemStr == "" {
			item, err = match.DBGetNWeights(db, ids[i])
			if err != nil {
				return nil, err
			}
			if item != nil {
				data, err := item.Pack()
				if err != nil {
					return nil, err
				}
				backHash[name] = string(data)
			}
		} else {
			item, err = match.UnpackSlot([]byte(itemStr))
			if err != nil {
				return nil, err
			}
		}
		items[i] = item
	}
	if len(backHash) > 0 {
		err = conn.Do(ctx, radix.FlatCmd(nil, "HMSET", c.SLOT, backHash))
	}

	return items, err
}

func (self *Controller) RedisGetAudiences(ctx context.Context, ids []uint32) ([]*match.Audience, error) {
	c := self.C
	conn := self.Redis
	db := self.DB

	n := len(ids)
	if n == 0 {
		return nil, nil
	}

	hash := map[string]string{}
	items := make([]*match.Audience, n)

	names := []string{c.AUDIENCE}
	for i := 0; i < n; i++ {
		names = append(names, pzutil.IDStr(ids[i]))
	}
	err := conn.Do(ctx, radix.Cmd(&hash, "HMGET", names...))
	if err != nil {
		return nil, err
	}

	backHash := map[string]string{}
	for i := 0; i < n; i++ {
		name := names[i+1]
		var item *match.Audience
		itemStr, ok := hash[name]
		if !ok || itemStr == "" {
			item, err = match.DBGetAudience(db, ids[i])
			if err != nil {
				return nil, err
			}
			if item != nil {
				data, err := item.Pack()
				if err != nil {
					return nil, err
				}
				backHash[name] = string(data)
			}
		} else {
			item, err = match.UnpackAudience([]byte(itemStr))
			if err != nil {
				return nil, err
			}
		}
		items[i] = item
	}
	if len(backHash) > 0 {
		err = conn.Do(ctx, radix.FlatCmd(nil, "HMSET", c.SLOT, backHash))
	}

	return items, err
}

func (self *Controller) RedisGetItem(ctx context.Context, id uint32) (*match.Item, error) {
	c := self.C
	conn := self.Redis
	db := self.DB

	var data []byte
	err := conn.Do(ctx, radix.Cmd(&data, "HGET", c.ITEM, pzutil.IDStr(id)))
	if err != nil {
		return nil, err
	}

	if len(data) > 0 {
		return match.UnpackItem(data)
	}
	item, err := match.DBGetItem(db, id)
	if err != nil {
		return nil, err
	}
	data, err = item.Pack()
	if err != nil {
		return nil, err
	}
	err = conn.Do(ctx, radix.Cmd(nil, "HSET", c.ITEM, pzutil.IDStr(id), string(data)))

	return item, err
}

func (self *Controller) RedisGetItems(ctx context.Context, ids []uint32) ([]*match.Item, error) {
	c := self.C
	conn := self.Redis
	db := self.DB

	n := len(ids)
	if n == 0 {
		return nil, nil
	}

	hash := map[string]string{}
	items := make([]*match.Item, n)

	names := []string{c.ITEM}
	for i := 0; i < n; i++ {
		names = append(names, pzutil.IDStr(ids[i]))
	}
	err := conn.Do(ctx, radix.Cmd(&hash, "HMGET", names...))
	if err != nil {
		return nil, err
	}

	backHash := map[string]string{}
	for i := 0; i < n; i++ {
		name := names[i+1]
		var item *match.Item
		itemStr, ok := hash[name]
		if !ok || itemStr == "" {
			item, err = match.DBGetItem(db, ids[i])
			if err != nil {
				return nil, err
			}
			if item != nil {
				data, err := item.Pack()
				if err != nil {
					return nil, err
				}
				backHash[name] = string(data)
			}
		} else {
			item, err = match.UnpackItem([]byte(itemStr))
			if err != nil {
				return nil, err
			}
		}
		items[i] = item
	}
	if len(backHash) > 0 {
		err = conn.Do(ctx, radix.FlatCmd(nil, "HMSET", c.ITEM, backHash))
	}

	return items, err
}
*/
