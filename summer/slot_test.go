package summer

import (
	"testing"
)

func TestSlot(t *testing.T) {
	s := CreateSlot("", "", "", "", "", "", "", "", "", "", "")
	if s.TotalScore() != float32(0.0) {
		t.Errorf("%v", s)
	}
	if s.Pack() != 2996629 {
		t.Errorf("%v", s.Pack())
	}
}
