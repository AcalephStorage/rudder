package util

import (
	"strconv"
)

// ToInt32 is a convenience function to convert a string to int32
func ToInt32(in string) (out int32) {
	val, _ := strconv.ParseInt(in, 10, 32)
	return int32(val)
}
