package holiday

import (
	"testing"
	//"encoding/hex"
	//"fmt"
)

type Hello struct {
	A string
	B []uint32
}

type Fixed struct {
	A uint32
	B int32
	C int32
}

func TestUtils(t *testing.T) {
	for _, i := range []uint32{65536,789,4294967295} {
		for _, j := range []uint32{65535,788,4294967294} {
			str := PackTwo(i,j)
			x,y,err:=UnpackTwo(str)
			if err != nil {
				t.Fatal(err)
			}
			if i!=x || j!=y {
				t.Errorf("%d %d %d %s %d %d", i, j, (uint64(i)<<32) | uint64(j), str, x, y)
			}
		}
	}

	for _, i := range []uint32{1,789,4294967295} {
		for _, j := range []int32{-2147483647,-1,2147483647} {
			for _, k := range []int32{-2147483646,1,2147483646} {
				obj := &Fixed{i, j, k}
				packed, err := PackFixedURL(obj)
				if err != nil { t.Fatal(err) }
				new_obj := new(Fixed)
				UnpackFixedURL(new_obj, packed)
				if obj.A!=new_obj.A || obj.B!=new_obj.B || obj.C!=new_obj.C {
					t.Errorf("%v=>%v", obj, new_obj)
				}
			}
		}
	}

	v := []uint32{1,2,3,4,5}
	for i:=uint32(0);i<10;i++ {
		if (i==0 || i > 5) && GrepUint32(v, i) {
			t.Errorf("Should not found: %d", i)
		} else if i>0 && i<=5 && !GrepUint32(v, i) {
			t.Errorf("Should found: %d", i)
		}
	}

	for i:=uint32(0);i<10;i++ {
		idx := IndexUint32(v, i)
		if i==0 || i > 5 {
			if idx != -1 {
				t.Errorf("index should be -1: %d", idx)
			}
        } else {
			if idx != int(i-1) {
				t.Errorf("index should be i-1: %d %d", i, idx)
			}
		}
    }

	f := func(ins uint32) uint32 {
		return ins * ins
	}

	outs := MapUint32(v, f)
	for i, out := range v {
		if outs[i] != out*out {
			t.Errorf("grep result: %v", outs)
		}
	}

	u := []uint32{1,2,3,4,5,6,7,8,9,10}
	if !GrepAndN(u, v) {
		t.Errorf("grep result: %v", GrepAndN(u, v))
	}
	u = []uint32{2,3,4,5,6,7,8,9,10}
	if GrepAndN(u, v) {
		t.Errorf("grep result: %v", GrepAndN(u, v))
	}

	a := "I am ok"
	b := []uint32{1,2,3,4,5}
	h := &Hello{a,b}

	bs, err := PackObject(h)
	if err != nil {
		t.Fatal(err)
	}
	h_new := new(Hello)
	err = UnpackObject(bs, h_new)
	if h_new.A != a || h_new.B[0] != b[0] {
		t.Errorf("%v", h_new)
	}

	weights := []float32{0.1,0.2,0.3,0.4}
	probs := make(map[int]int)
	for i:=0; i<100000; i++ {
		k := SelectOne(weights)
		probs[k]++
	}
	if probs[0] <  9000 || probs[0] > 11000 ||
	   probs[1] < 19000 || probs[1] > 21000 ||
	   probs[2] < 29000 || probs[2] > 31000 ||
	   probs[3] < 39000 || probs[3] > 41000 {
		t.Errorf("%v", probs)
	}
}

func TestMoreUtils(t *testing.T) {
	for _, ip := range []string{
		"192.168.29.1","255.255.255.255","1.0.0.0","255.255.255.0",
		"111.222.33.44","254.253.0.1","1.2.3.4"} {
		u := Ip32Uint(ip)
		new_ip := Uint32Ip(u)
		if ip != new_ip {
			t.Errorf("%s=>%s", ip, new_ip)
		}
	}

	for _, u1 := range []int64{-9223372036854775808,-9223372036854775807,9223372036854775807,-4294967296,4294967296,4294967295,-1,1,0} {
		for _, u2 := range []int64{-9223372036854775808,-9223372036854775807,9223372036854775807,-4294967296,4294967296,4294967295,-1,1,0} {
			bs := Int2Byte(u1,u2)
			v1,v2 := Byte2Int(bs)
			if v1 != u1 || v2 != u2 {
				t.Errorf("%d=>%d", u1, v1)
				t.Errorf("%d=>%d", u2, v2)
			}
		}
	}
}
