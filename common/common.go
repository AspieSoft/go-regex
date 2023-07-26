package common

import (
	"math"
)

// JoinBytes is an easy way to join multiple values into a single []byte
func JoinBytes(bytes ...interface{}) []byte {
	res := []byte{}
	for _, b := range bytes {
		res = append(res, ToString[[]byte](b)...)
	}
	return res
}

// formatMemoryUsage converts bytes to megabytes
func FormatMemoryUsage(b uint64) float64 {
	return math.Round(float64(b) / 1024 / 1024 * 100) / 100
}
