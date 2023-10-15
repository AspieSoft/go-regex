package common

import (
	"math"
	"syscall"
)

// JoinBytes is an easy way to join multiple values into a single []byte
func JoinBytes(bytes ...interface{}) []byte {
	res := []byte{}
	for _, b := range bytes {
		res = append(res, ToString[[]byte](b)...)
	}
	return res
}

// SysFreeMemory returns the amount of memory available in megabytes
func SysFreeMemory() float64 {
	in := &syscall.Sysinfo_t{}
	err := syscall.Sysinfo(in)
	if err != nil {
		return 0
	}

	// If this is a 32-bit system, then these fields are
	// uint32 instead of uint64.
	// So we always convert to uint64 to match signature.
	return math.Round(float64(uint64(in.Freeram) * uint64(in.Unit)) / 1024 / 1024 * 100) / 100
}
