package number

import (
	"errors"
	"strconv"
)

func GetFloatFromInterface(val interface{}) (float64, error) {
	var result float64
	switch val.(type) {
	case float64:
		result = val.(float64)
	case float32:
		result = float64(val.(float32))
	case int:
		result = float64(val.(int))
	case int8:
		result = float64(val.(int8))
	case int16:
		result = float64(val.(int16))
	case int32:
		result = float64(val.(int32))
	case int64:
		result = float64(val.(int64))
	case uint:
		result = float64(val.(uint))
	case uint8:
		result = float64(val.(uint8))
	case uint16:
		result = float64(val.(uint16))
	case uint32:
		result = float64(val.(uint32))
	case uint64:
		result = float64(val.(uint64))
	case string:
		var err error
		result, err = strconv.ParseFloat(val.(string), 64)
		if err != nil {
			return 0, err
		}
	default:
		return 0, errors.New("VAL_IS_NOT_A_NUMBER")
	}
	return result, nil
}
