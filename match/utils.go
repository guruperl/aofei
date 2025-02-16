package match

import (
	"math/rand"
)

func SelectOne(weights []float32) int {
	total := float32(0.0)
	n := len(weights)
	for i := 0; i < n; i++ {
		total += weights[i]
	}
	for i := 0; i < n; i++ {
		weights[i] /= total
	}
	randp := rand.Float32()
	sump := float32(0.0)
	for i := 0; i < n; i++ {
		sump += weights[i]
		if sump > randp {
			return i
		}
	}
	return -1
}
