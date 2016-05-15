package xlsparser

import (
	"github.com/tealeg/xlsx"
)

func Parse(bytes []byte) ([][][]string, error) {
	xls, err := xlsx.OpenBinary(bytes)
	if err != nil {
		return nil, err
	}
	return toSlice(xls)
}

func toSlice(file *xlsx.File) (output [][][]string, err error) {
	output = [][][]string{}
	for _, sheet := range file.Sheets {
		s := [][]string{}
		for _, row := range sheet.Rows {
			if row == nil {
				continue
			}
			r := []string{}
			for _, cell := range row.Cells {
				str, _ := cell.SafeFormattedValue()
				r = append(r, str)
			}
			s = append(s, r)
		}
		output = append(output, s)
	}
	return output, nil
}
