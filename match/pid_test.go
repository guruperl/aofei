package match

import (
	"testing"
)

func TestPid(t *testing.T) {
	entry := Pid{1, 2, 3}
	packed, err := entry.Pack()
	if err != nil { t.Fatal(err) }
	entry1, err := UnpackPid(packed)
	if err != nil { t.Fatal(err) }
	if entry1 != entry {
		t.Errorf("%v", entry)
		t.Errorf("%v", entry1)
	}
}
