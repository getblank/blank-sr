package bjson

import (
	"math"
	"sort"
	"strings"

	"github.com/getblank/blank-sr/bdb"

	"github.com/ivahaev/go-logger"
)

const (
	number = iota
	str
	boolean
)

type SortableMaps struct {
	data         []bdb.M
	ids          []string
	stringValues []string
	numberValues []float64
	boolValues   []bool
	valType      int
	desc         bool
}

func SortMaps(data []bdb.M, prop, t string) {
	if prop == "" {
		return
	}
	sm := NewSortableMaps(data)
	sm.ids = make([]string, len(data), len(data))
	if strings.HasPrefix(prop, "-") {
		sm.desc = true
		prop = prop[1:]
	}
	switch t {
	case "string", "date":
		sm.valType = str
		sm.stringValues = make([]string, len(data), len(data))
		for i := range data {
			val, ok := data[i][prop]
			if !ok {
				sm.stringValues[i] = ""
			} else {
				if strVal, ok := val.(string); ok {
					sm.stringValues[i] = strVal
				}
			}
			if data[i]["_id"] == nil {
				logger.Warn("No _id in data", data[i])
				sm.ids[i] = ""
				continue
			}

			sm.ids[i] = data[i]["_id"].(string)
		}
	case "int", "float":
		sm.valType = number
		sm.numberValues = make([]float64, len(data), len(data))
		for i := range data {
			_n, ok := data[i][prop]
			if ok {
				if n, ok := _n.(float64); ok {
					sm.numberValues[i] = n
					continue
				}
			}
			sm.numberValues[i] = math.Inf(-1)
			if data[i]["_id"] == nil {
				logger.Warn("No _id in data", data[i])
				sm.ids[i] = ""
				continue
			}
			sm.ids[i] = data[i]["_id"].(string)
		}
	case "bool":
		sm.valType = boolean
		sm.boolValues = make([]bool, len(data), len(data))
		for i := range data {
			_b, ok := data[i][prop]
			if ok {
				if b, ok := _b.(bool); ok {
					sm.boolValues[i] = b
				}
			}
			sm.boolValues[i] = false
			if data[i]["_id"] == nil {
				logger.Warn("No _id in data", data[i])
				sm.ids[i] = ""
				continue
			}
			sm.ids[i] = data[i]["_id"].(string)
		}
	default:
		return
	}
	sm.Sort(prop)
	return
}

func NewSortableMaps(data []bdb.M) *SortableMaps {
	return &SortableMaps{data: data}
}

func (m *SortableMaps) Sort(prop string) {
	sort.Sort(m)
}

func (m *SortableMaps) Len() int {
	return len(m.data)
}

func (m *SortableMaps) Swap(i, j int) {
	m.data[i], m.data[j] = m.data[j], m.data[i]
	m.ids[i], m.ids[j] = m.ids[j], m.ids[i]
	switch m.valType {
	case str:
		m.stringValues[i], m.stringValues[j] = m.stringValues[j], m.stringValues[i]
	case number:
		m.numberValues[i], m.numberValues[j] = m.numberValues[j], m.numberValues[i]
	case boolean:
		m.boolValues[i], m.boolValues[j] = m.boolValues[j], m.boolValues[i]
	}
}

func (m *SortableMaps) Less(i, j int) (less bool) {
	switch m.valType {
	case str:
		if m.stringValues[i] == m.stringValues[j] {
			less = m.ids[i] < m.ids[j]
		} else {
			less = m.stringValues[i] < m.stringValues[j]
		}
	case number:
		if m.numberValues[i] == m.numberValues[j] {
			less = m.ids[i] < m.ids[j]
		} else {
			less = m.numberValues[i] < m.numberValues[j]
		}
	case boolean:
		if m.boolValues[i] == m.boolValues[j] {
			less = m.ids[i] < m.ids[j]
		} else {
			less = !m.boolValues[i] && m.boolValues[j]
		}
	}
	if m.desc {
		return !less
	}
	return less
}

type sortableJSONs struct {
	data         [][]byte
	ids          []string
	stringValues []string
	numberValues []float64
	boolValues   []bool
	valType      int
	desc         bool
}

func SortJSONs(data [][]byte, prop string, t string) {
	if prop == "" {
		return
	}
	if len(data) < 2 {
		return
	}
	ss := newSortableJSONs(data)
	ss.ids = make([]string, len(data), len(data))
	if strings.HasPrefix(prop, "-") {
		ss.desc = true
		prop = prop[1:]
	}
	switch t {
	case "string", "date":
		ss.valType = str
		ss.stringValues = make([]string, len(data), len(data))
		for i := range data {
			ss.stringValues[i], _ = ExtractString(data[i], prop)
			ss.ids[i], _ = ExtractString(data[i], "_id")
		}
	case "int", "float":
		ss.valType = number
		ss.numberValues = make([]float64, len(data), len(data))
		for i := range data {
			n, ok := ExtractFloat64(data[i], prop)
			if ok {
				ss.numberValues[i] = n
			} else {
				ss.numberValues[i] = math.Inf(-1)
			}
			ss.ids[i], _ = ExtractString(data[i], "_id")
		}
	case "bool":
		ss.valType = boolean
		ss.boolValues = make([]bool, len(data), len(data))
		for i := range data {
			ss.boolValues[i], _ = ExtractBool(data[i], prop)
			ss.ids[i], _ = ExtractString(data[i], "_id")
		}
	default:
		return
	}
	ss.Sort(prop)
	return
}

func newSortableJSONs(data [][]byte) *sortableJSONs {
	return &sortableJSONs{data: data}
}

func (m *sortableJSONs) Sort(prop string) {
	sort.Sort(m)
}

func (m *sortableJSONs) Len() int {
	return len(m.data)
}

func (m *sortableJSONs) Swap(i, j int) {
	m.data[i], m.data[j] = m.data[j], m.data[i]
	m.ids[i], m.ids[j] = m.ids[j], m.ids[i]
	switch m.valType {
	case str:
		m.stringValues[i], m.stringValues[j] = m.stringValues[j], m.stringValues[i]
	case number:
		m.numberValues[i], m.numberValues[j] = m.numberValues[j], m.numberValues[i]
	case boolean:
		m.boolValues[i], m.boolValues[j] = m.boolValues[j], m.boolValues[i]
	}
}

func (m *sortableJSONs) Less(i, j int) (less bool) {
	switch m.valType {
	case str:
		if m.stringValues[i] == m.stringValues[j] {
			less = m.ids[i] < m.ids[j]
		} else {
			less = m.stringValues[i] < m.stringValues[j]
		}
	case number:
		if m.numberValues[i] == m.numberValues[j] {
			less = m.ids[i] < m.ids[j]
		} else {
			less = m.numberValues[i] < m.numberValues[j]
		}
	case boolean:
		if m.boolValues[i] == m.boolValues[j] {
			less = m.ids[i] < m.ids[j]
		} else {
			less = !m.boolValues[i] && m.boolValues[j]
		}
	}
	if m.desc {
		return !less
	}
	return less
}

type sortableStrings struct {
	data         []string
	ids          []string
	stringValues []string
	numberValues []float64
	boolValues   []bool
	valType      int
	desc         bool
}

func SortStrings(data []string, prop string, t string) {
	if prop == "" {
		return
	}
	ss := newSortableStrings(data)
	ss.ids = make([]string, len(data), len(data))
	if strings.HasPrefix(prop, "-") {
		ss.desc = true
		prop = prop[1:]
	}
	switch t {
	case "string", "date":
		ss.valType = str
		ss.stringValues = make([]string, len(data), len(data))
		for i := range data {
			ss.stringValues[i], _ = ExtractString([]byte(data[i]), prop)
			ss.ids[i], _ = ExtractString([]byte(data[i]), "_id")
		}
	case "int", "float":
		ss.valType = number
		ss.numberValues = make([]float64, len(data), len(data))
		for i := range data {
			n, ok := ExtractFloat64([]byte(data[i]), prop)
			if ok {
				ss.numberValues[i] = n
			} else {
				ss.numberValues[i] = math.Inf(-1)
			}
			ss.ids[i], _ = ExtractString([]byte(data[i]), "_id")
		}
	case "bool":
		ss.valType = boolean
		ss.boolValues = make([]bool, len(data), len(data))
		for i := range data {
			ss.boolValues[i], _ = ExtractBool([]byte(data[i]), prop)
			ss.ids[i], _ = ExtractString([]byte(data[i]), "_id")
		}
	default:
		return
	}
	ss.Sort(prop)
	return
}

func newSortableStrings(data []string) *sortableStrings {
	return &sortableStrings{data: data}
}

func (m *sortableStrings) Sort(prop string) {
	sort.Sort(m)
}

func (m *sortableStrings) Len() int {
	return len(m.data)
}

func (m *sortableStrings) Swap(i, j int) {
	m.data[i], m.data[j] = m.data[j], m.data[i]
	m.ids[i], m.ids[j] = m.ids[j], m.ids[i]
	switch m.valType {
	case str:
		m.stringValues[i], m.stringValues[j] = m.stringValues[j], m.stringValues[i]
	case number:
		m.numberValues[i], m.numberValues[j] = m.numberValues[j], m.numberValues[i]
	case boolean:
		m.boolValues[i], m.boolValues[j] = m.boolValues[j], m.boolValues[i]
	}
}

func (m *sortableStrings) Less(i, j int) (less bool) {
	switch m.valType {
	case str:
		if m.stringValues[i] == m.stringValues[j] {
			less = m.ids[i] < m.ids[j]
		} else {
			less = m.stringValues[i] < m.stringValues[j]
		}
	case number:
		if m.numberValues[i] == m.numberValues[j] {
			less = m.ids[i] < m.ids[j]
		} else {
			less = m.numberValues[i] < m.numberValues[j]
		}
	case boolean:
		if m.boolValues[i] == m.boolValues[j] {
			less = m.ids[i] < m.ids[j]
		} else {
			less = !m.boolValues[i] && m.boolValues[j]
		}
	}
	if m.desc {
		return !less
	}
	return less
}
