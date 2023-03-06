package mkdeb

import (
	"fmt"
)

func pad(size uint64, shift uint) uint64 {
	x := (uint64(1) << shift) - 1
	return (size + x) &^ x
}

func padSigned(size int64, shift uint) int64 {
	neg := false
	if size < 0 {
		neg = true
		size = -size
	}
	result := int64(pad(uint64(size), shift))
	if result < 0 {
		panic(fmt.Errorf("out of range: pad(%d, %d) => %d", size, shift, result))
	}
	if neg {
		result = -result
	}
	return result
}
