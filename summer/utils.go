package summer

import (
	"bytes"
	"encoding/gob"
	"os"
	"reflect"
)

func IsDigit(s string) bool {
	for _, r := range s {
		if r == '-' || r == '+' {
			continue
		}
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func Filtering(vs []string, f func(string) bool) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

func Map(vs []string, f func(string) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}

func Index(vs []string, t string) int {
	for i, v := range vs {
		if v == t {
			return i
		}
	}
	return -1
}

func Grep(vs []string, t string) bool {
	return Index(vs, t) >= 0
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

// SaveObject encodes via file
func SaveObject(path string, obj interface{}) error {
	file, err := os.Create(path)
	if err == nil {
		encoder := gob.NewEncoder(file)
		encoder.Encode(obj)
	}
	file.Close()
	return err
}

// LoadObject decodes via file
func LoadObject(path string, obj interface{}) error {
	file, err := os.Open(path)
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(obj)
	}
	file.Close()
	return err
}

func GrepGeneral(items interface{}, v interface{}) bool {
	val := reflect.Indirect(reflect.ValueOf(items))
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			if v == val.Index(i).Interface() {
				return true
			}
		}
	}
	return false
}
