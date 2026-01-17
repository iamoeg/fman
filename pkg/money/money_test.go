package money

import (
	"errors"
	"fmt"
	"math"
	"testing"
)

// ============================================================================
// Constructor Tests
// ============================================================================

func TestFromCents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		cents int64
		want  Money
	}{
		{
			name:  "zero",
			cents: 0,
			want:  Money{cents: 0},
		},
		{
			name:  "positive value",
			cents: 100,
			want:  Money{cents: 100},
		},
		{
			name:  "negative value",
			cents: -999,
			want:  Money{cents: -999},
		},
		{
			name:  "max int64",
			cents: math.MaxInt64,
			want:  Money{cents: math.MaxInt64},
		},
		{
			name:  "min int64",
			cents: math.MinInt64,
			want:  Money{cents: math.MinInt64},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := FromCents(tt.cents)
			if got != tt.want {
				t.Errorf("FromCents(%d) = %v, want %v", tt.cents, got, tt.want)
			}
		})
	}
}

func TestFromMAD(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mad     float64
		want    Money
		wantErr error
	}{
		{
			name: "zero",
			mad:  0.0,
			want: Money{0},
		},
		{
			name: "simple whole number",
			mad:  100.0,
			want: Money{cents: 10_000},
		},
		{
			name: "negative value",
			mad:  -999.99,
			want: Money{-99_999},
		},
		{
			name: "value with decimals",
			mad:  13.37,
			want: Money{cents: 1_337},
		},
		{
			name: "rounding up",
			mad:  123.456789,
			want: Money{cents: 12_346},
		},
		{
			name: "rounding down",
			mad:  9.87123,
			want: Money{987},
		},
		{
			name: "small decimal",
			mad:  0.1234,
			want: Money{cents: 12},
		},
		{
			name: "negative zero",
			mad:  -0.0,
			want: Money{cents: 0},
		},
		{
			name: "typical Moroccan salary",
			mad:  8_500.50,
			want: Money{cents: 850_050},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := FromMAD(tt.mad)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("FromMAD(%f) error = %v, wantErr %v", tt.mad, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("FromMAD(%f) unexpected error: %v", tt.mad, err)
				return
			}
			if got != tt.want {
				t.Errorf("FromMAD(%f) = %v, want %v", tt.mad, got, tt.want)
			}
		})
	}
}

func TestFromMAD_InvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mad     float64
		wantErr error
	}{
		{
			name:    "NaN",
			mad:     math.NaN(),
			wantErr: ErrInvalidValue,
		},
		{
			name:    "positive infinity",
			mad:     math.Inf(1),
			wantErr: ErrInvalidValue,
		},
		{
			name:    "negative infinity",
			mad:     math.Inf(-1),
			wantErr: ErrInvalidValue,
		},
		{
			name:    "value too large (positive overflow)",
			mad:     float64(math.MaxInt64) / 50, // Will overflow when * 100
			wantErr: ErrOverflow,
		},
		{
			name:    "value too small (negative overflow)",
			mad:     float64(math.MinInt64) / 50, // Will overflow when * 100
			wantErr: ErrOverflow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := FromMAD(tt.mad)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("FromMAD(%f) error = %v, wantErr %v", tt.mad, err, tt.wantErr)
			}
		})
	}
}

// ============================================================================
// Accessor Tests
// ============================================================================

func TestCents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    Money
		want int64
	}{
		{
			name: "zero",
			m:    Money{0},
			want: 0,
		},
		{
			name: "positive",
			m:    Money{12345},
			want: 12345,
		},
		{
			name: "negative",
			m:    Money{-9876},
			want: -9876,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.m.Cents(); got != tt.want {
				t.Errorf("Cents() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestToMAD(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    Money
		want float64
	}{
		{
			name: "zero",
			m:    Money{0},
			want: 0.0,
		},
		{
			name: "one dirham",
			m:    Money{100},
			want: 1.0,
		},
		{
			name: "fractional",
			m:    Money{999},
			want: 9.99,
		},
		{
			name: "large amount",
			m:    Money{123_456},
			want: 1_234.56,
		},
		{
			name: "negative",
			m:    Money{-987_654_321},
			want: -9_876_543.21,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.m.ToMAD()
			if got != tt.want {
				t.Errorf("ToMAD() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    Money
		want string
	}{
		{
			name: "zero",
			m:    Money{0},
			want: "0.00 MAD",
		},
		{
			name: "positive",
			m:    Money{850_050},
			want: "8500.50 MAD",
		},
		{
			name: "negative",
			m:    Money{-123_456},
			want: "-1234.56 MAD",
		},
		{
			name: "small amount",
			m:    Money{5},
			want: "0.05 MAD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.m.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ============================================================================
// Arithmetic Tests - Add
// ============================================================================

func TestAdd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b Money
		want Money
	}{
		{
			name: "zero + zero",
			a:    Money{0},
			b:    Money{0},
			want: Money{0},
		},
		{
			name: "zero + positive",
			a:    Money{0},
			b:    Money{1},
			want: Money{1},
		},
		{
			name: "carry over 1000",
			a:    Money{999},
			b:    Money{1},
			want: Money{1_000},
		},
		{
			name: "typical addition",
			a:    Money{12345},
			b:    Money{98765},
			want: Money{111_110},
		},
		{
			name: "typical addition (small values)",
			a:    Money{847},
			b:    Money{184},
			want: Money{1_031},
		},
		{
			name: "negative + positive = zero",
			a:    Money{-45},
			b:    Money{45},
			want: Money{0},
		},
		{
			name: "negative + negative",
			a:    Money{-89},
			b:    Money{-11},
			want: Money{-100},
		},
		{
			name: "positive + negative = positive",
			a:    Money{74},
			b:    Money{-42},
			want: Money{32},
		},
		{
			name: "salary + bonus (realistic)",
			a:    Money{850_000}, // 8,500 MAD salary
			b:    Money{50_000},  // 500 MAD bonus
			want: Money{900_000}, // 9,000 MAD total
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.a.Add(tt.b)
			if err != nil {
				t.Errorf("Add() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Add() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdd_Overflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b Money
	}{
		{
			name: "positive overflow at boundary",
			a:    FromCents(math.MaxInt64 - 100),
			b:    FromCents(200),
		},
		{
			name: "positive overflow at max",
			a:    FromCents(math.MaxInt64),
			b:    FromCents(1),
		},
		{
			name: "negative underflow at boundary",
			a:    FromCents(math.MinInt64 + 100),
			b:    FromCents(-200),
		},
		{
			name: "negative underflow at min",
			a:    FromCents(math.MinInt64),
			b:    FromCents(-1),
		},
		{
			name: "large positive values",
			a:    FromCents(math.MaxInt64 / 2),
			b:    FromCents(math.MaxInt64/2 + 100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tt.a.Add(tt.b)
			if !errors.Is(err, ErrOverflow) {
				t.Errorf("Add() error = %v, wantErr %v", err, ErrOverflow)
			}
		})
	}
}

// ============================================================================
// Arithmetic Tests - Subtract
// ============================================================================

func TestSubtract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b Money
		want Money
	}{
		{
			name: "zero - zero",
			a:    Money{0},
			b:    Money{0},
			want: Money{0},
		},
		{
			name: "positive - zero",
			a:    Money{1},
			b:    Money{0},
			want: Money{1},
		},
		{
			name: "same values",
			a:    Money{1},
			b:    Money{1},
			want: Money{0},
		},
		{
			name: "zero - positive = negative",
			a:    Money{0},
			b:    Money{1},
			want: Money{-1},
		},
		{
			name: "simple subtraction",
			a:    Money{999},
			b:    Money{99},
			want: Money{900},
		},
		{
			name: "result is negative",
			a:    Money{99},
			b:    Money{999},
			want: Money{-900},
		},
		{
			name: "large values",
			a:    Money{98765},
			b:    Money{12345},
			want: Money{86420},
		},
		{
			name: "negative - positive",
			a:    Money{-800},
			b:    Money{11},
			want: Money{-811},
		},
		{
			name: "positive - negative = larger positive",
			a:    Money{500},
			b:    Money{-400},
			want: Money{900},
		},
		{
			name: "negative - negative",
			a:    Money{-500},
			b:    Money{-400},
			want: Money{-100},
		},
		{
			name: "salary - deductions (realistic)",
			a:    Money{850_000}, // 8,500 MAD gross
			b:    Money{85_000},  // 850 MAD deductions
			want: Money{765_000}, // 7,650 MAD net
		},
		{
			name: "zero minus large negative (edge case - should succeed)",
			a:    FromCents(0),
			b:    FromCents(math.MinInt64 + 1), // 0 - (MinInt64+1) = MaxInt64
			want: FromCents(math.MaxInt64),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.a.Subtract(tt.b)
			if err != nil {
				t.Errorf("Subtract() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Subtract() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubtract_Overflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b Money
	}{
		{
			name: "underflow at boundary (result too negative)",
			a:    FromCents(math.MinInt64 + 100),
			b:    FromCents(200),
		},
		{
			name: "underflow at min",
			a:    FromCents(math.MinInt64),
			b:    FromCents(1),
		},
		{
			name: "overflow when subtracting negative (effectively adding)",
			a:    FromCents(math.MaxInt64 - 100),
			b:    FromCents(-200),
		},
		{
			name: "overflow at max minus negative",
			a:    FromCents(math.MaxInt64),
			b:    FromCents(-1),
		},
		{
			name: "small positive minus MinInt64 (overflow)",
			a:    FromCents(1),
			b:    FromCents(math.MinInt64), // 1 - MinInt64 = 1 + (MaxInt64+1) overflows
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tt.a.Subtract(tt.b)
			if !errors.Is(err, ErrOverflow) {
				t.Errorf("Subtract() error = %v, wantErr %v", err, ErrOverflow)
			}
		})
	}
}

// ============================================================================
// Arithmetic Tests - Multiply
// ============================================================================

func TestMultiply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		m      Money
		factor float64
		want   Money
	}{
		{
			name:   "multiply by 1",
			m:      Money{123},
			factor: 1.0,
			want:   Money{123},
		},
		{
			name:   "multiply by 0",
			m:      Money{123},
			factor: 0.0,
			want:   Money{0},
		},
		{
			name:   "multiply by 2",
			m:      Money{123},
			factor: 2.0,
			want:   Money{246},
		},
		{
			name:   "multiply by 0.5",
			m:      Money{123},
			factor: 0.5,
			want:   Money{62},
		},
		{
			name:   "multiply by decimal",
			m:      Money{123},
			factor: 3.5,
			want:   Money{431},
		},
		{
			name:   "multiply by large decimal",
			m:      Money{123},
			factor: 9.90,
			want:   Money{1_218},
		},
		{
			name:   "multiply by negative",
			m:      Money{123},
			factor: -1.2345,
			want:   Money{-152},
		},
		{
			name:   "negative money times positive factor",
			m:      Money{-500},
			factor: 2.0,
			want:   Money{-1_000},
		},
		{
			name:   "negative money times negative factor",
			m:      Money{-500},
			factor: -2.0,
			want:   Money{1_000},
		},
		{
			name:   "tax calculation (realistic - 10% tax)",
			m:      Money{850_000}, // 8,500 MAD salary
			factor: 0.10,           // 10% tax
			want:   Money{85_000},  // 850 MAD tax
		},
		{
			name:   "CNSS calculation (realistic - 4.48% employee contribution)",
			m:      Money{850_000}, // 8,500 MAD salary
			factor: 0.0448,         // 4.48%
			want:   Money{38_080},  // 380.80 MAD
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.m.Multiply(tt.factor)
			if err != nil {
				t.Errorf("Multiply() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Multiply() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMultiply_Overflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		m      Money
		factor float64
	}{
		{
			name:   "positive overflow",
			m:      FromCents(math.MaxInt64 / 2),
			factor: 10.0,
		},
		{
			name:   "negative underflow",
			m:      FromCents(math.MinInt64 / 2),
			factor: 10.0,
		},
		{
			name:   "multiply by very large factor",
			m:      FromCents(1_000_000),
			factor: float64(math.MaxInt64),
		},
		{
			name:   "negative times very large factor",
			m:      FromCents(-1_000_000),
			factor: float64(math.MaxInt64),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tt.m.Multiply(tt.factor)
			if !errors.Is(err, ErrOverflow) {
				t.Errorf("Multiply() error = %v, wantErr %v", err, ErrOverflow)
			}
		})
	}
}

func TestMultiply_InvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		m       Money
		factor  float64
		wantErr error
	}{
		{
			name:    "multiply by NaN",
			m:       FromCents(100),
			factor:  math.NaN(),
			wantErr: ErrInvalidValue,
		},
		{
			name:    "multiply by positive infinity",
			m:       FromCents(100),
			factor:  math.Inf(1),
			wantErr: ErrInvalidValue,
		},
		{
			name:    "multiply by negative infinity",
			m:       FromCents(100),
			factor:  math.Inf(-1),
			wantErr: ErrInvalidValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tt.m.Multiply(tt.factor)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Multiply() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ============================================================================
// Arithmetic Tests - Divide
// ============================================================================

func TestDivide(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		m       Money
		divisor float64
		want    Money
	}{
		{
			name:    "divide by 1",
			m:       Money{100},
			divisor: 1.0,
			want:    Money{100},
		},
		{
			name:    "divide by 2",
			m:       Money{100},
			divisor: 2.0,
			want:    Money{50},
		},
		{
			name:    "divide with rounding",
			m:       Money{100},
			divisor: 3.0,
			want:    Money{33}, // Rounds 33.333...
		},
		{
			name:    "divide by decimal",
			m:       Money{100},
			divisor: 0.5,
			want:    Money{200},
		},
		{
			name:    "negative divided by positive",
			m:       Money{-100},
			divisor: 2.0,
			want:    Money{-50},
		},
		{
			name:    "positive divided by negative",
			m:       Money{100},
			divisor: -2.0,
			want:    Money{-50},
		},
		{
			name:    "negative divided by negative",
			m:       Money{-100},
			divisor: -2.0,
			want:    Money{50},
		},
		{
			name:    "split salary among days (realistic)",
			m:       Money{850_000}, // 8,500 MAD monthly
			divisor: 30.0,           // 30 days
			want:    Money{28_333},  // 283.33 MAD per day
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.m.Divide(tt.divisor)
			if err != nil {
				t.Errorf("Divide() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Divide() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDivide_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		m       Money
		divisor float64
		wantErr error
	}{
		{
			name:    "divide by zero",
			m:       FromCents(100),
			divisor: 0.0,
			wantErr: ErrDivByZero,
		},
		{
			name:    "divide by NaN",
			m:       FromCents(100),
			divisor: math.NaN(),
			wantErr: ErrInvalidValue,
		},
		{
			name:    "divide by positive infinity",
			m:       FromCents(100),
			divisor: math.Inf(1),
			wantErr: ErrInvalidValue,
		},
		{
			name:    "divide by negative infinity",
			m:       FromCents(100),
			divisor: math.Inf(-1),
			wantErr: ErrInvalidValue,
		},
		{
			name:    "overflow (divide by very small number)",
			m:       FromCents(math.MaxInt64 / 2),
			divisor: 0.0001,
			wantErr: ErrOverflow,
		},
		{
			name:    "underflow (negative divided by very small)",
			m:       FromCents(math.MinInt64 / 2),
			divisor: 0.0001,
			wantErr: ErrOverflow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tt.m.Divide(tt.divisor)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Divide() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ============================================================================
// Comparison Tests
// ============================================================================

func TestEquals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b Money
		want bool
	}{
		{
			name: "equal positive values",
			a:    Money{100},
			b:    Money{100},
			want: true,
		},
		{
			name: "equal negative values",
			a:    Money{-100},
			b:    Money{-100},
			want: true,
		},
		{
			name: "both zero",
			a:    Money{0},
			b:    Money{0},
			want: true,
		},
		{
			name: "different values",
			a:    Money{100},
			b:    Money{200},
			want: false,
		},
		{
			name: "positive vs negative",
			a:    Money{100},
			b:    Money{-100},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.a.Equals(tt.b); got != tt.want {
				t.Errorf("Equals() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLessThan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b Money
		want bool
	}{
		{
			name: "smaller < larger",
			a:    Money{100},
			b:    Money{200},
			want: true,
		},
		{
			name: "equal values",
			a:    Money{100},
			b:    Money{100},
			want: false,
		},
		{
			name: "larger < smaller",
			a:    Money{200},
			b:    Money{100},
			want: false,
		},
		{
			name: "negative < positive",
			a:    Money{-100},
			b:    Money{100},
			want: true,
		},
		{
			name: "more negative < less negative",
			a:    Money{-200},
			b:    Money{-100},
			want: true,
		},
		{
			name: "zero < positive",
			a:    Money{0},
			b:    Money{1},
			want: true,
		},
		{
			name: "negative < zero",
			a:    Money{-1},
			b:    Money{0},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.a.LessThan(tt.b); got != tt.want {
				t.Errorf("LessThan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLessThanOrEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b Money
		want bool
	}{
		{
			name: "less than",
			a:    Money{100},
			b:    Money{200},
			want: true,
		},
		{
			name: "equal",
			a:    Money{100},
			b:    Money{100},
			want: true,
		},
		{
			name: "greater than",
			a:    Money{200},
			b:    Money{100},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.a.LessThanOrEqual(tt.b); got != tt.want {
				t.Errorf("LessThanOrEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGreaterThan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b Money
		want bool
	}{
		{
			name: "larger > smaller",
			a:    Money{200},
			b:    Money{100},
			want: true,
		},
		{
			name: "equal values",
			a:    Money{100},
			b:    Money{100},
			want: false,
		},
		{
			name: "smaller > larger",
			a:    Money{100},
			b:    Money{200},
			want: false,
		},
		{
			name: "positive > negative",
			a:    Money{100},
			b:    Money{-100},
			want: true,
		},
		{
			name: "less negative > more negative",
			a:    Money{-100},
			b:    Money{-200},
			want: true,
		},
		{
			name: "positive > zero",
			a:    Money{1},
			b:    Money{0},
			want: true,
		},
		{
			name: "zero > negative",
			a:    Money{0},
			b:    Money{-1},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.a.GreaterThan(tt.b); got != tt.want {
				t.Errorf("GreaterThan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGreaterThanOrEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b Money
		want bool
	}{
		{
			name: "greater than",
			a:    Money{200},
			b:    Money{100},
			want: true,
		},
		{
			name: "equal",
			a:    Money{100},
			b:    Money{100},
			want: true,
		},
		{
			name: "less than",
			a:    Money{100},
			b:    Money{200},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.a.GreaterThanOrEqual(tt.b); got != tt.want {
				t.Errorf("GreaterThanOrEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    Money
		want bool
	}{
		{
			name: "zero",
			m:    Money{0},
			want: true,
		},
		{
			name: "positive",
			m:    Money{1},
			want: false,
		},
		{
			name: "negative",
			m:    Money{-1},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.m.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPositive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    Money
		want bool
	}{
		{
			name: "positive",
			m:    Money{1},
			want: true,
		},
		{
			name: "zero",
			m:    Money{0},
			want: false,
		},
		{
			name: "negative",
			m:    Money{-1},
			want: false,
		},
		{
			name: "large positive",
			m:    Money{1_000_000},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.m.IsPositive(); got != tt.want {
				t.Errorf("IsPositive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNegative(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    Money
		want bool
	}{
		{
			name: "negative",
			m:    Money{-1},
			want: true,
		},
		{
			name: "zero",
			m:    Money{0},
			want: false,
		},
		{
			name: "positive",
			m:    Money{1},
			want: false,
		},
		{
			name: "large negative",
			m:    Money{-1_000_000},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.m.IsNegative(); got != tt.want {
				t.Errorf("IsNegative() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// Edge Cases and Integration Tests
// ============================================================================

func TestFloatingPointPrecision(t *testing.T) {
	t.Parallel()

	// The classic floating-point problem: 0.1 + 0.2 != 0.3
	// Our Money type should handle this correctly
	m1, _ := FromMAD(0.1)
	m2, _ := FromMAD(0.2)
	expected, _ := FromMAD(0.3)

	result, err := m1.Add(m2)
	if err != nil {
		t.Errorf("Add() unexpected error: %v", err)
		return
	}

	if !result.Equals(expected) {
		t.Errorf("0.1 + 0.2 = %v, want %v (floating point precision issue)", result, expected)
	}
}

func TestRoundTripConversion(t *testing.T) {
	t.Parallel()

	// Test that converting MAD -> Money -> MAD preserves value within 0.01 MAD
	tests := []float64{
		0.0,
		1.0,
		10.50,
		100.99,
		1234.56,
		-50.25,
		8500.50, // Typical salary
	}

	for _, original := range tests {
		t.Run(fmt.Sprintf("%.2f MAD", original), func(t *testing.T) {
			t.Parallel()

			m, err := FromMAD(original)
			if err != nil {
				t.Errorf("FromMAD(%f) error: %v", original, err)
				return
			}

			reconstructed := m.ToMAD()
			diff := math.Abs(original - reconstructed)

			// Allow 0.01 MAD difference due to rounding
			if diff > 0.01 {
				t.Errorf("round-trip failed: %f -> %v -> %f (diff: %f)",
					original, m, reconstructed, diff)
			}
		})
	}
}

func TestRealisticPayrollCalculation(t *testing.T) {
	t.Parallel()

	// Simulate a realistic Moroccan payroll calculation
	baseSalary, _ := FromMAD(8500.00)    // 8,500 MAD base salary
	seniorityBonus, _ := FromMAD(425.00) // 5% seniority
	cnssEmployee, _ := FromMAD(380.80)   // 4.48% CNSS employee
	amoEmployee, _ := FromMAD(170.00)    // 2% AMO employee
	incomeTax, _ := FromMAD(850.00)      // ~10% income tax
	rounding := FromCents(20)            // 0.20 MAD rounding

	// Calculate gross
	gross, err := baseSalary.Add(seniorityBonus)
	if err != nil {
		t.Fatalf("calculating gross: %v", err)
	}

	// Calculate net
	net := gross
	net, err = net.Subtract(cnssEmployee)
	if err != nil {
		t.Fatalf("subtracting CNSS: %v", err)
	}
	net, err = net.Subtract(amoEmployee)
	if err != nil {
		t.Fatalf("subtracting AMO: %v", err)
	}
	net, err = net.Subtract(incomeTax)
	if err != nil {
		t.Fatalf("subtracting tax: %v", err)
	}
	net, err = net.Add(rounding)
	if err != nil {
		t.Fatalf("adding rounding: %v", err)
	}

	// Expected: 8925.00 - 380.80 - 170.00 - 850.00 + 0.20 = 7,524.40 MAD
	expected, _ := FromMAD(7524.40)

	if !net.Equals(expected) {
		t.Errorf("payroll calculation: got %v, want %v", net, expected)
		t.Logf("Base: %v", baseSalary)
		t.Logf("+ Seniority: %v", seniorityBonus)
		t.Logf("= Gross: %v", gross)
		t.Logf("- CNSS: %v", cnssEmployee)
		t.Logf("- AMO: %v", amoEmployee)
		t.Logf("- Tax: %v", incomeTax)
		t.Logf("+ Rounding: %v", rounding)
		t.Logf("= Net: %v", net)
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkFromMAD(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = FromMAD(1234.56)
	}
}

func BenchmarkAdd(b *testing.B) {
	m1 := FromCents(123456)
	m2 := FromCents(654321)
	for i := 0; i < b.N; i++ {
		_, _ = m1.Add(m2)
	}
}

func BenchmarkMultiply(b *testing.B) {
	m := FromCents(850000)
	for i := 0; i < b.N; i++ {
		_, _ = m.Multiply(0.0448)
	}
}

func BenchmarkToMAD(b *testing.B) {
	m := FromCents(123456)
	for i := 0; i < b.N; i++ {
		_ = m.ToMAD()
	}
}
