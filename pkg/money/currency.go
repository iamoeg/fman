package money

import (
	"errors"

	"github.com/iamoeg/bootdev-capstone/pkg/util"
)

// Currency represents a supported currency code.
type Currency string

const (
	// MAD represents the Moroccan Dirham.
	MAD Currency = "MAD"
)

// supportedCurrencies is the set of all supported currency codes.
// Currently only MAD is implemented.
var supportedCurrencies = map[Currency]struct{}{
	MAD: {},
}

// SupportedCurrenciesStr is a comma-separated list of supported currency values.
var SupportedCurrenciesStr = util.EnumMapToString(supportedCurrencies)

var (
	// ErrCurrencyNotSupported is returned when an unsupported currency is used.
	ErrCurrencyNotSupported = errors.New("money: currency not supported")
)

// IsSupported returns true if the currency is supported.
func (c Currency) IsSupported() bool {
	_, ok := supportedCurrencies[c]
	return ok
}

func (c Currency) String() string {
	return string(c)
}
