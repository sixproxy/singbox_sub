package util

import (
	"strconv"
)

func String2Int(str string) int {
	i, err := strconv.Atoi(str)
	if err != nil {
		return -999999
	}
	return i
}
