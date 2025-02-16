package pzutil

import (
	"testing"
	//"encoding/hex"
	//"fmt"
)

type Hello struct {
	A string
	B []uint32
}

func TestUtils(t *testing.T) {
	for _, i := range []uint32{65536, 789, 4294967295} {
		for _, j := range []uint32{65535, 788, 4294967294} {
			str := PackTwo(i, j)
			x, y, err := UnpackTwo(str)
			if err != nil {
				t.Fatal(err)
			}
			//t.Errorf("%d %d %d %s %d %d", i, j, (uint64(i)<<32) | uint64(j), str, x, y)
			if i != x || j != y {
				t.Errorf("%d %d %d %s %d %d", i, j, (uint64(i)<<32)|uint64(j), str, x, y)
			}
		}
	}

	v := []uint32{1, 2, 3, 4, 5}
	for i := uint32(0); i < 10; i++ {
		if (i == 0 || i > 5) && Grep(v, i) {
			t.Errorf("Should not found: %d", i)
		} else if i > 0 && i <= 5 && !Grep(v, i) {
			t.Errorf("Should found: %d", i)
		}
	}

	for i := uint32(0); i < 10; i++ {
		idx := IndexUint32(v, i)
		if i == 0 || i > 5 {
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

	u := []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if !GrepAndN(u, v) {
		t.Errorf("grep result: %v", GrepAndN(u, v))
	}
	u = []uint32{2, 3, 4, 5, 6, 7, 8, 9, 10}
	if GrepAndN(u, v) {
		t.Errorf("grep result: %v", GrepAndN(u, v))
	}

	a := "I am ok"
	b := []uint32{1, 2, 3, 4, 5}
	h := &Hello{a, b}

	bs, err := PackObject(h)
	if err != nil {
		t.Fatal(err)
	}
	hnew := new(Hello)
	err = UnpackObject(bs, hnew)
	if hnew.A != a || hnew.B[0] != b[0] {
		t.Errorf("%v", hnew)
	}

	/*
	   weights := []float32{0.1,0.2,0.3,0.4}
	   probs := make(map[int]int)

	   	for i:=0; i<100000; i++ {
	   		k := SelectOne(weights)
	   		probs[k]++
	   	}

	   if probs[0] <  9000 || probs[0] > 11000 ||

	   	   probs[1] < 19000 || probs[1] > 21000 ||
	   	   probs[2] < 29000 || probs[1] > 31000 ||
	   	   probs[3] < 39000 || probs[1] > 41000 {
	   		t.Errorf("%v", probs)
	   	}
	*/
}

/*
func TestIdname(t *testing.T) {
	for i:=0;i<300;i++ {
		name := GetKeyName("XXX", uint32(i))
		dst := make([]byte, hex.EncodedLen(len(name)))
		hex.Encode(dst, []byte(name))
		if string(dst) != fmt.Sprintf("5858583a0000%04x",i) {
			t.Errorf("%s,5858583a0000%04x", dst, i)
		}
	}
}
*/
