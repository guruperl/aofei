// Package pid provides models and methods for handling pid operations
package pid

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"io"

	"github.com/genelet/winter/genelet"
)

type Model struct {
	genelet.Model
}

func Uint32Byte(u uint32) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, u)
	return buf
}
func Byte32Uint(bs []byte) uint32 {
	return binary.LittleEndian.Uint32(bs)
}

func Int2Byte(u1 int64, u2 int64) []byte {
	buf1 := make([]byte, 8)
	buf2 := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf1, uint64(u1))
	binary.LittleEndian.PutUint64(buf2, uint64(u2))
	return append(buf1, buf2...)
}
func Byte2Int(bs []byte) (int64, int64) {
	return int64(binary.LittleEndian.Uint64(bs[0:8])), int64(binary.LittleEndian.Uint64(bs[8:]))
}

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
