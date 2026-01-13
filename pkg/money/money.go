package money

import (
	"errors"
	"fmt"
	"math"
)

// Error definitions
var (
	ErrOverflow     = errors.New("money: operation would overflow")
	ErrDivByZero    = errors.New("money: division by zero")
	ErrInvalidValue = errors.New("money: invalid value (NaN or Inf)")
)

// Money represents an amount of money in Moroccan Dirhams (MAD).
// It stores the value as integer cents to avoid floating-point precision errors.
// All arithmetic operations maintain exact precision and check for overflow.
//
// Example:
//
//	salary := money.FromMAD(10234.56)  // 10,234.56 MAD
//	bonus := money.FromMAD(500.00)
//	total, err := salary.Add(bonus)
//	if err != nil {
//	    log.Fatal(err)
//	}
type Money struct {
	cents int64
}

// ============================================================================
// Constructors
// ============================================================================

// FromCents creates a Money value from an integer number of cents.
// This is the most direct constructor and never fails.
//
// Example:
//
//	m := money.FromCents(12345)  // 123.45 MAD
func FromCents(cents int64) Money {
	return Money{cents: cents}
}

// FromMAD creates a Money value from a floating-point MAD amount.
// The value is rounded to the nearest cent.
//
// Returns ErrInvalidValue if mad is NaN or Inf.
// Returns ErrOverflow if the conversion would overflow int64.
//
// Example:
//
//	salary, err := money.FromMAD(8500.50)
//	if err != nil {
//	    log.Fatal(err)
//	}
func FromMAD(mad float64) (Money, error) {
	// Check for invalid values
	if math.IsNaN(mad) || math.IsInf(mad, 0) {
		return Money{}, ErrInvalidValue
	}

	// Check if conversion would overflow
	// We multiply by 100 to convert to cents, so check before multiplication
	madCents := mad * 100
	if madCents > float64(math.MaxInt64) || madCents < float64(math.MinInt64) {
		return Money{}, fmt.Errorf("%w: %f MAD is too large to represent", ErrOverflow, mad)
	}

	cents := int64(math.Round(madCents))
	return Money{cents: cents}, nil
}

// ============================================================================
// Accessors
// ============================================================================

// Cents returns the raw cent value as an int64.
// This is useful for storage in databases or serialization.
//
// Example:
//
//	m := money.FromMAD(123.45)
//	cents := m.Cents()  // 12345
func (m Money) Cents() int64 {
	return m.cents
}

// ToMAD converts the Money value to a floating-point MAD amount.
// This should only be used for display purposes, not for calculations.
//
// Example:
//
//	m := money.FromCents(12345)
//	mad := m.ToMAD()  // 123.45
func (m Money) ToMAD() float64 {
	return float64(m.cents) / 100.0
}

// String returns a formatted string representation of the money value.
// Implements the fmt.Stringer interface.
//
// Example:
//
//	m := money.FromMAD(1234.56)
//	fmt.Println(m)  // "1234.56 MAD"
func (m Money) String() string {
	return fmt.Sprintf("%.2f MAD", m.ToMAD())
}

// ============================================================================
// Arithmetic Operations
// ============================================================================

// Add returns the sum of two Money values.
// Returns ErrOverflow if the result would overflow int64.
//
// Example:
//
//	salary, _ := money.FromMAD(8500.00)
//	bonus, _ := money.FromMAD(500.00)
//	total, err := salary.Add(bonus)
func (m Money) Add(other Money) (Money, error) {
	// Check for positive overflow
	if other.cents > 0 && m.cents > math.MaxInt64-other.cents {
		return Money{}, fmt.Errorf("%w: %v + %v", ErrOverflow, m, other)
	}

	// Check for negative underflow
	if other.cents < 0 && m.cents < math.MinInt64-other.cents {
		return Money{}, fmt.Errorf("%w: %v + %v", ErrOverflow, m, other)
	}

	return Money{cents: m.cents + other.cents}, nil
}

// Subtract returns the difference of two Money values (m - other).
// Returns ErrOverflow if the result would overflow int64.
//
// Example:
//
//	gross, _ := money.FromMAD(8500.00)
//	deductions, _ := money.FromMAD(850.00)
//	net, err := gross.Subtract(deductions)
func (m Money) Subtract(other Money) (Money, error) {
	// Check for underflow (result too negative)
	// m - other; if other > 0, result gets smaller (more negative)
	if other.cents > 0 && m.cents < math.MinInt64+other.cents {
		return Money{}, fmt.Errorf("%w: %v - %v", ErrOverflow, m, other)
	}

	// Check for overflow (result too positive)
	// m - other; if other < 0, subtracting negative = adding positive
	if other.cents < 0 && m.cents > math.MaxInt64+other.cents {
		return Money{}, fmt.Errorf("%w: %v - %v", ErrOverflow, m, other)
	}

	return Money{cents: m.cents - other.cents}, nil
}

// Multiply returns the product of a Money value and a floating-point factor.
// The result is rounded to the nearest cent.
// Returns ErrInvalidValue if factor is NaN or Inf.
// Returns ErrOverflow if the result would overflow int64.
//
// Example:
//
//	salary, _ := money.FromMAD(8500.00)
//	tax, err := salary.Multiply(0.10)  // 10% tax
func (m Money) Multiply(factor float64) (Money, error) {
	// Check for invalid factor
	if math.IsNaN(factor) || math.IsInf(factor, 0) {
		return Money{}, fmt.Errorf("%w: factor is NaN or Inf", ErrInvalidValue)
	}

	// Calculate result as float to check bounds
	result := float64(m.cents) * factor

	// Check if result fits in int64
	if result > float64(math.MaxInt64) || result < float64(math.MinInt64) {
		return Money{}, fmt.Errorf("%w: %v * %f", ErrOverflow, m, factor)
	}

	cents := int64(math.Round(result))
	return Money{cents: cents}, nil
}

// Divide returns the quotient of a Money value divided by a floating-point divisor.
// The result is rounded to the nearest cent.
// Returns ErrDivByZero if divisor is zero.
// Returns ErrInvalidValue if divisor is NaN or Inf.
// Returns ErrOverflow if the result would overflow int64.
//
// Example:
//
//	monthly, _ := money.FromMAD(8500.00)
//	daily, err := monthly.Divide(30.0)  // Split across 30 days
func (m Money) Divide(divisor float64) (Money, error) {
	// Check for division by zero
	if divisor == 0.0 {
		return Money{}, ErrDivByZero
	}

	// Check for invalid divisor
	if math.IsNaN(divisor) || math.IsInf(divisor, 0) {
		return Money{}, fmt.Errorf("%w: divisor is NaN or Inf", ErrInvalidValue)
	}

	// Calculate result as float to check bounds
	result := float64(m.cents) / divisor

	// Check if result fits in int64
	if result > float64(math.MaxInt64) || result < float64(math.MinInt64) {
		return Money{}, fmt.Errorf("%w: %v / %f", ErrOverflow, m, divisor)
	}

	cents := int64(math.Round(result))
	return Money{cents: cents}, nil
}

// ============================================================================
// Comparison Operations
// ============================================================================

// Equals returns true if two Money values are equal.
//
// Example:
//
//	m1, _ := money.FromMAD(100.00)
//	m2, _ := money.FromMAD(100.00)
//	if m1.Equals(m2) {
//	    fmt.Println("Equal")
//	}
func (m Money) Equals(other Money) bool {
	return m.cents == other.cents
}

// LessThan returns true if m is less than other.
//
// Example:
//
//	salary, _ := money.FromMAD(8500.00)
//	threshold, _ := money.FromMAD(10000.00)
//	if salary.LessThan(threshold) {
//	    fmt.Println("Below threshold")
//	}
func (m Money) LessThan(other Money) bool {
	return m.cents < other.cents
}

// GreaterThan returns true if m is greater than other.
//
// Example:
//
//	salary, _ := money.FromMAD(8500.00)
//	minimum, _ := money.FromMAD(3000.00)
//	if salary.GreaterThan(minimum) {
//	    fmt.Println("Above minimum")
//	}
func (m Money) GreaterThan(other Money) bool {
	return m.cents > other.cents
}

// IsZero returns true if the Money value is zero.
//
// Example:
//
//	balance := money.FromCents(0)
//	if balance.IsZero() {
//	    fmt.Println("No balance")
//	}
func (m Money) IsZero() bool {
	return m.cents == 0
}

// IsPositive returns true if the Money value is greater than zero.
//
// Example:
//
//	profit, _ := money.FromMAD(500.00)
//	if profit.IsPositive() {
//	    fmt.Println("Made a profit")
//	}
func (m Money) IsPositive() bool {
	return m.cents > 0
}

// IsNegative returns true if the Money value is less than zero.
//
// Example:
//
//	balance, _ := money.FromMAD(-100.00)
//	if balance.IsNegative() {
//	    fmt.Println("Negative balance")
//	}
func (m Money) IsNegative() bool {
	return m.cents < 0
}
