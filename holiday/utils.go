package holiday

import (
	"os"
	"math/rand"
	"bytes"
	"net"
	"encoding/base64"
	"encoding/base32"
	"encoding/gob"
	"encoding/binary"
	"encoding/hex"
)

func GetSizeId(w, h uint16) uint32 {
	return (uint32(w) << 16) | uint32(h)
}

func GetSizes(size_id uint32) (uint16, uint16) {
	return uint16(size_id>>16), uint16(0xFFFF & size_id)
}

func PackTwo(x, y uint32) string {
    buf := new(bytes.Buffer);
    binary.Write(buf, binary.LittleEndian, x)
    binary.Write(buf, binary.LittleEndian, y)

	//return base64.RawURLEncoding.EncodeToString(buf.Bytes())
    return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf.Bytes())
}

func UnpackTwo(text string) (uint32, uint32, error) {
    //data, err := base64.RawURLEncoding.DecodeString(text)
    data, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(text)
    if err != nil { return 0, 0, err}
    x := make([]uint32,2)
    buf := bytes.NewReader([]byte(data))
    err = binary.Read(buf, binary.LittleEndian, &x);
    if err != nil { return 0, 0, err}

    return x[0], x[1], nil
}

func IsDigit(s string) bool {
	for _, r := range s {
		if (r < '0' || r > '9') { return false }
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
	if vs == nil || len(vs)<1 { return true }
	if items == nil || len(items)<1 { return false }
	for _, t := range items {
		if IndexUint32(vs, t)==-1 { return false }
	}
	return true
}

// any in items matchs any in vs
func GrepOrN(vs []uint32, items []uint32) bool {
	if vs == nil || len(vs)<1 { return true }
	if items == nil || len(items)<1 { return false }
	for _, t := range items {
		if GrepUint32(vs, t) { return true }
	}
	return false
}

func GrepUint32(vs []uint32, t uint32) bool {
	if vs == nil || len(vs)<1 { return true }
    return IndexUint32(vs, t) >= 0
}

func PackFixedURL(obj interface{}) (string, error) {
    buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj)
    if err != nil { return "", err }
    return base64.RawURLEncoding.EncodeToString(buf.Bytes()), nil
}

// pass a pointer as obj
func UnpackFixedURL(obj interface{}, text string) error {
	data, err := base64.RawURLEncoding.DecodeString(text)
	if err != nil { return err }
	buf := bytes.NewReader([]byte(data))
	return binary.Read(buf, binary.LittleEndian, obj)
}

func PackObject(obj interface{}) ([]byte, error) {
    buf := new(bytes.Buffer)
    enc := gob.NewEncoder(buf)
    err := enc.Encode(obj)
    if err != nil { return nil, err }
    return buf.Bytes(), nil
}

func UnpackObject(packed []byte, obj interface{}) error {
    buf := bytes.NewReader(packed)
    dec := gob.NewDecoder(buf)
    err := dec.Decode(obj)
    if err != nil { return err }
    return nil
}

// Encode via Gob to file
func SaveObject(path string, obj interface{}) error {
	file, err := os.Create(path)
	if err == nil {
		encoder := gob.NewEncoder(file)
		encoder.Encode(obj)
	}
	file.Close()
	return err
 }

// Decode Gob file
func LoadObject(path string, obj interface{}) error {
	file, err := os.Open(path)
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(obj)
	}
	file.Close()
	return err
}

func Ip32Uint(ip_str string) uint32 {
	return Ip2uint(net.ParseIP(ip_str))
}
func Uint32Ip(ip uint32) string {
	netip := Uint2ip(ip)
	return netip.String()
}

func Ip2uint(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}
func Uint2ip(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}

func Uint32Byte(u uint32) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, u)
	return buf
}
func Byte32Uint(bs []byte) uint32 {
	return binary.LittleEndian.Uint32(bs)
}

func Int64Byte(u int64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(u))
	return buf
}
func Byte64Int(bs []byte) int64 {
	return int64(binary.LittleEndian.Uint64(bs))
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

func Hex2Byte(src string) ([]byte, error) {
	return hex.DecodeString(src)
}
func Byte2Hex(src []byte) string {
	return hex.EncodeToString(src)
}

func SelectOne(weights []float32) int {
	total := float32(0.0);
	n := len(weights);
	for i:=0; i<n; i++ {
		total += weights[i];
	}
	for i:=0; i<n; i++ {
		weights[i] /= total;
	}
	rand_p := rand.Float32();
	sum_p := float32(0.0);
	for i:=0; i<n; i++ {
		sum_p += weights[i];
		if sum_p > rand_p {
			return i;
		}
	}
	return -1;
}

func IsPrivateIP(ip_str string) bool {
    ip := net.ParseIP(ip_str)
    if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
        return true
    }

    for _, cidr := range []string{
        "127.0.0.0/8",    // IPv4 loopback
        "10.0.0.0/8",     // RFC1918
        "172.16.0.0/12",  // RFC1918
        "192.168.0.0/16", // RFC1918
        "169.254.0.0/16", // RFC3927 link-local
        "::1/128",        // IPv6 loopback
        "fe80::/10",      // IPv6 link-local
        "fc00::/7",       // IPv6 unique local addr
    } {
        _, block, _ := net.ParseCIDR(cidr)
        if block.Contains(ip) {
            return true
        }
    }

    return false
}
