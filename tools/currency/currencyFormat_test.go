package currency

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnitFormat(t *testing.T) {

	test := func(currency string, amount int64, expected string) {
		require.Equal(t, expected, UnitFormat(currency, amount))
	}

	test("USD", 123456, "$1234.56")
	test("USD", 12345, "$123.45")
	test("USD", 1234, "$12.34")
	test("USD", 123, "$1.23")
	test("USD", 12, "$0.12")
	test("USD", 1, "$0.01")
	test("USD", 0, "$0.00")
	test("USD", -1, "-$0.01")
	test("USD", -12, "-$0.12")
	test("USD", -123, "-$1.23")
	test("USD", -1234, "-$12.34")
	test("USD", -12345, "-$123.45")
	test("USD", -123456, "-$1234.56")
}
