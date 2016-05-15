package comparer

import (
	"reflect"
	"strconv"
	"strings"
)

func CompareInterfaces(v1, v2 interface{}, operator string) (result bool) {
	if v1 == nil && v2 == nil {
		return operator == "=" || operator == ">=" || operator == "<=" || operator == "contains"
	}
	if (v1 == nil && v2 != nil) || (v1 != nil && v2 == nil) {
		return operator == "!=" || operator == ">" || operator == "="
	}
	switch reflect.TypeOf(v1).Kind() {
	case reflect.Array, reflect.Slice:
		strArray, ok := v1.([]string)
		if ok {
			if operator == "$in" {
				operator = "="
			}
			for _, v := range strArray {
				result = CompareInterfaces(v, v2, operator)
				if result {
					return result
				}
			}
			return false
		}
		intArray, ok := v1.([]int)
		if ok {
			if operator == "$in" {
				operator = "="
			}
			for _, v := range intArray {
				result = CompareInterfaces(v, v2, operator)
				if result {
					return result
				}
			}
			return false
		}
		floatArray, ok := v1.([]float64)
		if ok {
			if operator == "$in" {
				operator = "="
			}
			for _, v := range floatArray {
				result = CompareInterfaces(v, v2, operator)
				if result {
					return result
				}
			}
			return false
		}
		boolArray, ok := v1.([]bool)
		if ok {
			if operator == "$in" {
				operator = "="
			}
			for _, v := range boolArray {
				result = CompareInterfaces(v, v2, operator)
				if result {
					return result
				}
			}
			return false
		}
	case reflect.String:
		val1, ok := v1.(string)
		if !ok {
			return
		}
		val2, ok := v2.(string)
		if !ok {
			_val2, ok := v2.(float64)
			if !ok {
				return
			}
			val2 = strconv.FormatFloat(_val2, 'f', -1, 64)
		}
		switch operator {
		case "=":
			return strings.ToLower(val1) == strings.ToLower(val2)
		case "contains":
			if strings.HasPrefix(val2, "=") {
				return val1 == strings.TrimLeft(val2, "=")
			}
			return strings.Contains(strings.ToLower(val1), strings.ToLower(val2))
		case "!=":
			return val1 != val2
		case ">":
			return val1 > val2
		case ">=":
			return val1 >= val2
		case "<":
			return val1 < val2
		case "<=":
			return val1 <= val2
		}
	case reflect.Int:
		val1, ok := v1.(int)
		if !ok {
			_val1, ok := v1.(float64)
			if !ok {
				return
			}
			val1 = int(_val1)
		}
		val2, ok := v2.(int)
		if !ok {
			if !ok {
				_val2, ok := v2.(float64)
				if !ok {
					strVal2, ok := v2.(string)
					if !ok {
						return
					}
					var err error
					val2, err = strconv.Atoi(strVal2)
					if err != nil {
						return
					}
				} else {
					val2 = int(_val2)
				}
			}
		}
		switch operator {
		case "=", "contains":
			return val1 == val2
		case "!=":
			return val1 != val2
		case ">":
			return val1 > val2
		case ">=":
			return val1 >= val2
		case "<":
			return val1 < val2
		case "<=":
			return val1 <= val2
		}
	case reflect.Float64:
		val1, ok := v1.(float64)
		if !ok {
			return
		}
		val2, ok := v2.(float64)
		if !ok {
			strVal2, ok := v2.(string)
			if !ok {
				return
			}
			var err error
			val2, err = strconv.ParseFloat(strVal2, 10)
			if err != nil {
				return
			}
		}
		switch operator {
		case "=", "contains":
			return val1 == val2
		case "!=":
			return val1 != val2
		case ">":
			return val1 > val2
		case ">=":
			return val1 >= val2
		case "<":
			return val1 < val2
		case "<=":
			return val1 <= val2
		}
	case reflect.Bool:
		val1, ok := v1.(bool)
		if !ok {
			return
		}
		val2, ok := v2.(bool)
		if !ok {
			return
		}
		switch operator {
		case "=", "contains":
			return val1 == val2
		case "!=":
			return val1 != val2
		}
	}
	return
}
