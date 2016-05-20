package config

import (
	"errors"
	"time"

	"strconv"
	"strings"

	"github.com/getblank/blank-sr/bdb"
	"github.com/getblank/blank-sr/berror"
	"github.com/getblank/blank-sr/utils/comparer"

	log "github.com/Sirupsen/logrus"
)

var invalidPropValue = errors.New("Invalid prop value")

func (m *Store) ValidateData(data bdb.M) (bdb.M, error) {
	if len(m.Props) == 0 {
		return nil, errors.New("No props in objects")
	}

	for pName, p := range m.Props {
		val, ok := data[pName]
		if p.isRequired(data) {
			if !ok || val == nil || (p.Type == PropString && val == "") || (p.Type == PropRef && val == "") {
				return nil, errors.New("Missing required prop '" + pName + "'")
			}
		}
		if !ok && p.Default != nil {
			data[pName] = p.Default
		}
		if ok && (val == nil || (p.Type == PropRef && val == "")) {
			delete(data, pName)
		}
	}

	for k, v := range data {
		p, ok := m.Props[k]
		if !ok {
			delete(data, k)
			continue
		}

		switch p.Type {
		case PropVirtual, PropPassword, PropVirtualRefList:
			delete(data, k)
			continue
		case PropDynamic:
			continue
		}

		validated, err := p.validateData(v)
		if err != nil {
			return nil, errors.New("Wrong data in prop '" + k + "'")
		}
		data[k] = validated
	}

	return data, nil
}

func (p *Prop) validateData(pValue interface{}) (interface{}, error) {
	switch p.Type {
	case PropInt:
		switch pValue.(type) {
		case string:
			val, err := strconv.Atoi(pValue.(string))
			if err != nil {
				return nil, err
			}
			return validatePropInt(p, val)
		case float64:
			return validatePropInt(p, int(pValue.(float64)))
		case int:
			return validatePropInt(p, pValue.(int))
		case int64:
			return validatePropInt(p, int(pValue.(int64)))
		case int32:
			return validatePropInt(p, int(pValue.(int32)))
		case int16:
			return validatePropInt(p, int(pValue.(int16)))
		case int8:
			return validatePropInt(p, int(pValue.(int8)))
		default:
			return nil, berror.WrongData
		}
	case PropFloat:
		switch pValue.(type) {
		case string:
			val, err := strconv.ParseFloat(strings.Replace(pValue.(string), ",", ".", 1), 64)
			if err != nil {
				return nil, err
			}
			return validatePropFloat(p, val)
		case float64:
			return validatePropFloat(p, pValue.(float64))
		case int:
			return validatePropFloat(p, float64(pValue.(int)))
		case int64:
			return validatePropFloat(p, float64(pValue.(int64)))
		case int32:
			return validatePropFloat(p, float64(pValue.(int32)))
		case int16:
			return validatePropFloat(p, float64(pValue.(int16)))
		case int8:
			return validatePropFloat(p, float64(pValue.(int8)))
		default:
			return nil, berror.WrongData
		}
	case PropBool:
		if _, ok := pValue.(bool); !ok {
			return nil, berror.WrongData
		}
	case PropString:
		value, ok := pValue.(string)
		if !ok {
			return nil, berror.WrongData
		}
		return validatePropString(p, value)
	case PropDate:
		if strDate, ok := pValue.(string); ok {
			if strDate != "" {
				date, err := time.Parse(time.RFC3339, strDate)
				if err != nil {
					return nil, berror.WrongData
				}
				pValue = date.UTC().Format(time.RFC3339)
			}
		} else {
			return nil, berror.WrongData
		}
	case PropRef:
		_, ok := pValue.(string)
		if !ok {
			return nil, berror.WrongData
		}
	case PropRefList:
		switch pValue.(type) {
		case []string:
			return pValue, nil
		case []interface{}:
			list, _ := pValue.([]interface{})
			for _, v := range list {
				if _, ok := v.(string); !ok {
					return nil, berror.WrongData
				}
			}
		}
	case PropObject:
		value, ok := pValue.(bdb.M)
		if !ok {
			value, ok = pValue.(map[string]interface{})
			if !ok {
				return nil, berror.WrongData
			}
		}
		return validatePropObject(p, value)
	case PropObjectList:
		switch pValue.(type) {
		case []bdb.M:
			values := pValue.([]bdb.M)
			for k, v := range values {
				val, err := validatePropObject(p, v)
				if err != nil {
					return nil, err
				}
				values[k] = val
			}
			return values, nil
		case []interface{}:
			values := pValue.([]interface{})
			for i, v := range values {
				value, ok := v.(bdb.M)
				if !ok {
					value, ok = v.(map[string]interface{})
					if !ok {
						return nil, berror.WrongData
					}
				}
				val, err := validatePropObject(p, value)
				if err != nil {
					return nil, err
				}
				values[i] = val
			}
			return values, nil
		case []map[string]interface{}:
			values := pValue.([]map[string]interface{})
			for k, v := range values {
				val, err := validatePropObject(p, v)
				if err != nil {
					return nil, err
				}
				values[k] = val
			}
			return values, nil
		default:
			return nil, berror.WrongData
		}
	}
	return pValue, nil
}

func validatePropInt(p *Prop, pData int) (int, error) {
	if p.Min != nil {
		min, ok := p.Min.(float64)
		if !ok {
			log.Error("Wrong config for prop", p)
			return 0, invalidPropValue
		}
		if pData < int(min) {
			return 0, invalidPropValue
		}
	}
	if p.Max != nil {
		max, ok := p.Max.(float64)
		if !ok {
			log.Error("Wrong config for prop", p)
			return 0, invalidPropValue
		}
		if pData > int(max) {
			return 0, invalidPropValue
		}
	}
	return pData, nil
}

func validatePropFloat(p *Prop, pData float64) (float64, error) {
	if p.Min != nil {
		min, ok := p.Min.(float64)
		if !ok {
			log.Error("Wrong config for prop", p)
			return 0, invalidPropValue
		}
		if pData < min {
			return 0, invalidPropValue
		}
	}
	if p.Max != nil {
		max, ok := p.Max.(float64)
		if !ok {
			log.Error("Wrong config for prop", p)
			return 0, invalidPropValue
		}
		if pData > max {
			return 0, invalidPropValue
		}
	}
	return pData, nil
}

func validatePropObject(p *Prop, data map[string]interface{}) (map[string]interface{}, error) {
	for k, _p := range p.Props {
		val, ok := data[k]
		if _p.isRequired(data) {
			if !ok || val == nil {
				return nil, errors.New("Missing required prop '" + k + "'")
			}
		}
		if !ok && _p.Default != nil {
			data[k] = p.Default
		}
	}
	for k, v := range data {
		p, ok := p.Props[k]
		if !ok || v == nil {
			delete(data, k)
			continue
		}
		if p.Type == PropVirtual || p.Type == PropPassword {
			delete(data, k)
			continue
		}

		val, err := p.validateData(v)
		if err != nil {
			return nil, errors.New("Wrong data in prop '" + k + "'")
		}
		data[k] = val
	}
	return data, nil
}

func validatePropString(p *Prop, pData string) (string, error) {
	if p.MaxLength > 0 && len([]rune(pData)) > p.MaxLength {
		return "", invalidPropValue
	}
	if p.MinLength > 0 && len([]rune(pData)) < p.MinLength {
		return "", invalidPropValue
	}
	if p.PatternCompiled != nil && !p.PatternCompiled.MatchString(pData) {
		return "", invalidPropValue
	}
	return pData, nil
}

func (p *Prop) isRequired(data map[string]interface{}) (required bool) {
	if p.requiredBool {
		return true
	}
	var values []interface{}
	var err error
	for _, c := range p.requiredConditions {
		values, err = getValues(data, c.Property)
		if err != nil {
			if c.Operator != "!=" {
				return
			}
			continue
		}
		for _, v := range values {
			required = comparer.CompareInterfaces(v, c.Value, c.Operator)
		}
		if !required {
			return
		}
	}
	return required
}
