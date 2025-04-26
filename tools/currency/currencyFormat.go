package currency

import (
	"strings"

	"github.com/benpate/rosetta/convert"
)

func UnitFormat(currency string, units int64) string {

	var result string

	if units < 0 {
		result = "-"
		units = -units
	}

	result += Symbol(currency)

	// Insert a decimal point
	unitString := convert.String(units)

	if length := len(unitString); length < 3 {
		unitString = strings.Repeat("0", 3-length) + unitString
	}

	point := len(unitString) - 2
	result += unitString[:point] + "." + unitString[point:]

	return result
}
