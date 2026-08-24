// Package cachegeneration owns private coordination values used while a
// complete Redis cache generation is staged and installed.
package cachegeneration

const (
	// MarkerField cannot collide with numeric cache fields. If an invalid
	// publisher domain collides with it, its packed value replaces MarkerValue
	// and publication fails closed.
	MarkerField = "\x00aofei:cache-generation"
	MarkerValue = "complete"
)
