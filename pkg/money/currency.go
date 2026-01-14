package money

import "errors"

// Currency represents a supported currency code.
type Currency string

const (
	// MAD represents the Moroccan Dirham.
	MAD Currency = "MAD"
)

// SupportedCurrencies is the set of all supported currency codes.
// Currently only MAD is implemented.
var SupportedCurrencies = map[Currency]struct{}{
	MAD: struct{}{},
}

var (
	// ErrCurrencyNotSupported is returned when an unsupported currency is used.
	ErrCurrencyNotSupported = errors.New("money: currency not supported")
)

// IsSupported returns true if the currency is supported.
func (c Currency) IsSupported() bool {
	_, ok := SupportedCurrencies[c]
	return ok
}

func (c Currency) String() string {
	return string(c)
}
