package match

import (
	"testing"
)

func TestRAdvSelectOne(t *testing.T) {
	weights := []float32{0.1, 0.2, 0.3, 0.4}
	probs := make(map[int]int)
	for i := 0; i < 100000; i++ {
		k := selectOne(weights)
		probs[k]++
	}
	if probs[0] < 9000 || probs[0] > 11000 ||
		probs[1] < 19000 || probs[1] > 21000 ||
		probs[2] < 29000 || probs[1] > 31000 ||
		probs[3] < 39000 || probs[1] > 41000 {
		t.Errorf("%v", probs)
	}
}
