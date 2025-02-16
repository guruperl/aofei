package pzutil

import (
	"bytes"
	"encoding/base32"
	"encoding/binary"
	"encoding/gob"
	"net"
	"os"
	"reflect"
	"strconv"
)

func PackTwo(x, y uint32) string {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, x)
	binary.Write(buf, binary.LittleEndian, y)

	//return base64.RawStdEncoding.EncodeToString(buf.Bytes())
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf.Bytes())
}

func UnpackTwo(text string) (uint32, uint32, error) {
	data, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(text)
	if err != nil {
		return 0, 0, err
	}
	x := make([]uint32, 2)
	buf := bytes.NewReader([]byte(data))
	err = binary.Read(buf, binary.LittleEndian, &x)
	if err != nil {
		return 0, 0, err
	}

	return x[0], x[1], nil
}

func IsDigit(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func IndexString(vs []string, t string) int {
	for i, v := range vs {
		if v == t {
			return i
		}
	}
	return -1
}

func MapUint32(vs []uint32, f func(uint32) uint32) []uint32 {
	vsm := make([]uint32, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}

func IndexUint32(vs []uint32, t uint32) int {
	for i, v := range vs {
		if v == t {
			return i
		}
	}
	return -1
}

func GrepAndN(vs []uint32, items []uint32) bool {
	if vs == nil || len(vs) < 1 {
		return true
	}
	if items == nil || len(items) < 1 {
		return false
	}
	for _, t := range items {
		if IndexUint32(vs, t) == -1 {
			return false
		}
	}
	return true
}

// GrepOrN returns true if any in items matchs any in vs
func GrepOrN(vs []uint32, items []uint32) bool {
	if vs == nil || len(vs) < 1 {
		return true
	}
	if items == nil || len(items) < 1 {
		return false
	}
	for _, t := range items {
		if GrepUint32(vs, t) {
			return true
		}
	}
	return false
}

func GrepUint32(vs []uint32, t uint32) bool {
	if vs == nil || len(vs) < 1 {
		return true
	}
	return IndexUint32(vs, t) >= 0
}

func Grep(items interface{}, v interface{}) (ok bool) {
	val := reflect.Indirect(reflect.ValueOf(items))
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			if ok = v == val.Index(i).Interface(); ok {
				return
			}
		}
	}
	return
}

func PackObject(obj interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(obj)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func UnpackObject(packed []byte, obj interface{}) error {
	buf := bytes.NewReader(packed)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(obj)
	if err != nil {
		return err
	}
	return nil
}

// SaveObject encode via Gob to file
func SaveObject(path string, obj interface{}) error {
	file, err := os.Create(path)
	if err == nil {
		encoder := gob.NewEncoder(file)
		encoder.Encode(obj)
	}
	file.Close()
	return err
}

// LoadObject decodes Gob file
func LoadObject(path string, obj interface{}) error {
	file, err := os.Open(path)
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(obj)
	}
	file.Close()
	return err
}

func GetKeyName(name string, id uint32) string {
	//	b := make([]byte,4)
	//	binary.BigEndian.PutUint32(b, id)
	//	return name + ":" + string(b)
	return name + ":" + IDStr(id)
}

func IDStr(id uint32) string {
	return strconv.FormatUint(uint64(id), 10)
}

func IP2Uint(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func Uint2IP(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}
