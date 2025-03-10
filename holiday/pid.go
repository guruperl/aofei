package holiday

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"io"

	"github.com/genelet/winter/genelet"
)


func (self *Model) Insupd(extra ...map[string]interface{}) error {
	ARGS := self.ARGS
	var u1, u2 int64

	device := ARGS.Get("device")
	switch device {
	case "iphone":
		bts, err := hex.DecodeString(ARGS.Get("idfa"))
		if err != nil {
			return err
		}
		u1, u2 = Byte2Int(bts)
	case "android":
		bts, err := hex.DecodeString(ARGS.Get("android_id"))
		if err != nil {
			return err
		}
		u1, u2 = Byte2Int(bts)
	case "wechat":
		h := md5.New()
		io.WriteString(h, ARGS.Get("openid"))
		u1, u2 = Byte2Int(h.Sum(nil))
		self.CurrentTable = "device_wechat"
		self.CurrentKey = "ts"
		//self.Tags = nil
		fv := map[string]interface{}{"u1": u1, "u2": u2, "openid": ARGS.Get("openid")}
		uniques := []string{"u1", "u2"}
		err := self.InsupdHash(fv, uniques)
		if err != nil {
			return err
		}
	case "h5":
		h := md5.New()
		h.Write(Uint32Byte(ARGS.Get("ip32"))
		if ua, ok := ARGS["ua_str"]; ok {
			io.WriteString(h, ua.(string))
		}
		u1, u2 = Byte2Int(h.Sum(nil))
		self.CurrentTable = "device_h5"
		self.CurrentKey = "ts"
		self.Tags = nil
		fv := map[string]interface{}{"u1": u1, "u2": u2, "ip32": int32(ARGS["ip32"].(uint32)), "pzua": int32(ARGS["pzua"].(uint32))}
		uniques := []string{"u1", "u2"}
		err := self.InsupdHash(fv, uniques)
		if err != nil {
			return err
		}
	default:
	}

	self.CurrentTable = "pid"
	self.CurrentKey = "user_id"
	self.Tags = []string{"pub_id", "device", "encrypt"}
	ARGS["u1"] = u1
	ARGS["u2"] = u2
	return self.Model.Insupd(extra...)
}
