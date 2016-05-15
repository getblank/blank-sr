package array

import (
	"sort"

	"github.com/getblank/blank-sr/bdb"
	"github.com/getblank/blank-sr/berror"
)

func InArrayInt(arr []int, s int) int {
	pos := -1
	for i, value := range arr {
		if value == s {
			return i
		}
	}
	return pos
}

// IndexOfSortedStrings searches for s in a sorted slice of strings and returns the index
// or -1 if not found
func IndexOfSortedStrings(arr []string, s string) int {
	i := sort.SearchStrings(arr, s)
	if i < len(arr) && arr[i] == s {
		return i
	}
	return -1
}

func InArrayString(arr []string, s string) int {
	pos := -1
	for i, value := range arr {
		if value == s {
			return i
		}
	}
	return pos
}

func InBothArraysString(arr1, arr2 []string) bool {
	for _, v := range arr1 {
		if InArrayString(arr2, v) != -1 {
			return true
		}
	}
	return false
}

func StringArraysEquals(a, b []string) bool {
	if a == nil && b == nil {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func CreateMapFromInterface(i interface{}) (m bdb.M, err error) {
	var ok bool
	m, ok = i.(bdb.M)
	if ok {
		return m, nil
	}
	m, ok = i.(map[string]interface{})
	if !ok {
		return nil, berror.WrongData
	}
	return m, nil
}

func CreateStringSliceFromInterface(i interface{}) (result []string, err error) {
	result = []string{}
	var ok bool
	result, ok = i.([]string)
	if ok {
		return result, nil
	}
	_result, ok := i.([]interface{})
	if !ok {
		return nil, berror.WrongData
	}
	for _, v := range _result {
		val, ok := v.(string)
		if !ok {
			return nil, berror.WrongData
		}
		result = append(result, val)
	}
	return result, nil
}
