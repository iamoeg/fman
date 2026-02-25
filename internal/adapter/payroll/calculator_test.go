package calculator

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/iamoeg/bootdev-capstone/internal/domain"
	"github.com/iamoeg/bootdev-capstone/pkg/money"
)

// ============================================================================
// Helpers
// ============================================================================

func mad(amount float64) money.Money {
	m, err := money.FromMAD(amount)
	if err != nil {
		panic(err)
	}
	return m
}

func cents(c int64) money.Money {
	return money.FromCents(c)
}

// ============================================================================
// Unit tests: completedYears
// ============================================================================

func TestCompletedYears(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		from time.Time
		to   time.Time
		want int
	}{
		{
			name: "exactly 3 years",
			from: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			to:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			want: 3,
		},
		{
			name: "3 years and 11 months is still 3",
			from: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			to:   time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
			want: 3,
		},
		{
			name: "anniversary not yet reached this year",
			from: time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC),
			to:   time.Date(2023, 6, 14, 0, 0, 0, 0, time.UTC),
			want: 2,
		},
		{
			name: "anniversary exactly reached",
			from: time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC),
			to:   time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC),
			want: 3,
		},
		{
			name: "less than one year",
			from: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			to:   time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC),
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := completedYears(tt.from, tt.to)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==========================================================================
// Unit tests: seniorityRate
// ============================================================================

func TestSeniorityRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		years int
		want  float64
	}{
		{0, 0.00},
		{1, 0.00},
		{2, 0.05}, // exactly 2 → 2–5 bracket
		{4, 0.05},
		{5, 0.10}, // exactly 5 → 5–12 bracket
		{11, 0.10},
		{12, 0.15}, // exactly 12 → 12–20 bracket
		{19, 0.15},
		{20, 0.20}, // exactly 20 → 20–25 bracket
		{24, 0.20},
		{25, 0.25}, // exactly 25 → 25+ bracket
		{30, 0.25},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			got := seniorityRate(tt.years)
			assert.Equal(t, tt.want, got, "years=%d", tt.years)
		})
	}
}

// ==========================================================================
// Unit tests: calculateSeniorityBonus
// ==========================================================================

func TestCalculateSeniorityBonus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		base      money.Money
		years     int
		wantBonus money.Money
	}{
		{
			name:      "0% — under 2 years",
			base:      mad(10_000),
			years:     1,
			wantBonus: mad(0),
		},
		{
			name:      "5% — 3 years",
			base:      mad(10_000),
			years:     3,
			wantBonus: mad(500),
		},
		{
			name:      "10% — 6 years",
			base:      mad(10_000),
			years:     6,
			wantBonus: mad(1_000),
		},
		{
			name:      "25% — 30 years",
			base:      mad(10_000),
			years:     30,
			wantBonus: mad(2_500),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := calculateSeniorityBonus(tt.base, tt.years)
			require.NoError(t, err)
			assert.Equal(t, tt.wantBonus, got)
		})
	}
}

// ==========================================================================
// Unit tests: calculateCNSSEmployee
// ==========================================================================

func TestCalculateCNSSEmployee(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		gross               money.Money
		wantSocialAllowance money.Money
		wantJobLossComp     money.Money
		wantTotal           money.Money
	}{
		{
			name:                "below ceiling — 5,000 MAD",
			gross:               mad(5_000),
			wantSocialAllowance: mad(224),    // 5,000 × 4.48%
			wantJobLossComp:     mad(9.50),   // 5,000 × 0.19%
			wantTotal:           mad(233.50), // 224 + 9.50
		},
		{
			name:                "at ceiling — 6,000 MAD",
			gross:               mad(6_000),
			wantSocialAllowance: mad(268.80), // 6,000 × 4.48%
			wantJobLossComp:     mad(11.40),  // 6,000 × 0.19%
			wantTotal:           mad(280.20), // 268.80 + 11.40
		},
		{
			name:                "above ceiling — 11,000 MAD (capped at 6,000)",
			gross:               mad(11_000),
			wantSocialAllowance: mad(268.80), // 6,000 × 4.48%
			wantJobLossComp:     mad(11.40),  // 6,000 × 0.19%
			wantTotal:           mad(280.20),
		},
		{
			name:                "above ceiling — 21,000 MAD (capped at 6,000)",
			gross:               mad(21_000),
			wantSocialAllowance: mad(268.80),
			wantJobLossComp:     mad(11.40),
			wantTotal:           mad(280.20),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := calculateCNSSEmployee(tt.gross)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSocialAllowance, got.socialAllowance, "social allowance")
			assert.Equal(t, tt.wantJobLossComp, got.jobLossCompensation, "job loss compensation")
			assert.Equal(t, tt.wantTotal, got.total, "total")
		})
	}
}

// ==========================================================================
// Unit tests: calculateCNSSEmployer
// ==========================================================================

func TestCalculateCNSSEmployer(t *testing.T) {
	t.Parallel()

	// Gross 11,000 MAD (capped base = 6,000)
	gross := mad(11_000)
	got, err := calculateCNSSEmployer(gross)
	require.NoError(t, err)

	assert.Equal(t, mad(704), got.familyBenefits, "family benefits: 11,000 × 6.40%")
	assert.Equal(t, mad(538.80), got.socialAllowance, "social allowance: 6,000 × 8.98%")
	assert.Equal(t, mad(22.80), got.jobLossCompensation, "job loss compensation: 6,000 × 0.38%")
	assert.Equal(t, mad(176), got.trainingTax, "training tax: 11,000 × 1.60%")
}

// ==========================================================================
// Unit tests: calculateAMOEmployee
// ==========================================================================

func TestCalculateAMOEmployee(t *testing.T) {
	t.Parallel()

	got, err := calculateAMOEmployee(mad(11_000))
	require.NoError(t, err)
	assert.Equal(t, mad(248.60), got) // 11,000 × 2.26%
}

// ==========================================================================
// Unit tests: calculateProfessionalExpenseDeduction
// ==========================================================================

func TestCalculateProfessionalExpenseDeduction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		gross money.Money
		want  money.Money
	}{
		{
			name:  "annual gross <= 78,000 — 35% rate, under cap",
			gross: mad(5_000), // annual = 60,000 ≤ 78,000 → 35%; 5,000×35%=1,750 < 2,500
			want:  mad(1_750),
		},
		{
			name:  "annual gross <= 78,000 — 35% rate, hits cap",
			gross: mad(6_500), // annual = 78,000 ≤ 78,000 → 35%; 6,500×35%=2,275 < 2,500
			want:  mad(2_275),
		},
		{
			name:  "annual gross > 78,000 — 20% rate, under cap",
			gross: mad(11_000), // annual = 132,000 > 78,000 → 20%; 11,000×20%=2,200 < 2,500
			want:  mad(2_200),
		},
		{
			name:  "annual gross > 78,000 — 20% rate, hits cap",
			gross: mad(21_000), // annual = 252,000 > 78,000 → 20%; 21,000×20%=4,200 > 2,500
			want:  mad(2_500),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := calculateProfessionalExpenseDeduction(tt.gross)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==========================================================================
// Unit tests: calculateFamilyChargeDeduction
// ==========================================================================

func TestCalculateFamilyChargeDeduction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		dependents int
		want       money.Money
	}{
		{0, mad(0)},
		{1, mad(40)},
		{2, mad(80)},
		{6, mad(240)},
		{7, mad(240)}, // capped at 6
		{10, mad(240)},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			got, err := calculateFamilyChargeDeduction(tt.dependents)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got, "dependents=%d", tt.dependents)
		})
	}
}

// ==========================================================================
// Unit tests: calculateIncomeTax
// ==========================================================================

func TestCalculateIncomeTax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		monthlyNetTaxable    money.Money
		wantMonthlyIncomeTax money.Money
	}{
		{
			name:                 "0% bracket — annual taxable 24,000",
			monthlyNetTaxable:    mad(2_000), // annual = 24,000 ≤ 40,000
			wantMonthlyIncomeTax: mad(0),
		},
		{
			name:                 "10% bracket — annual taxable 48,000",
			monthlyNetTaxable:    mad(4_000), // annual = 48,000; income tax=(48,000×10%)−4,000=800; monthly=66.67
			wantMonthlyIncomeTax: cents(6667),
		},
		{
			name:                 "20% bracket — annual taxable 70,000 (example from DOMAIN.md)",
			monthlyNetTaxable:    mad(5_833.33), // annual ≈ 70,000; income tax=(70,000×20%)−10,000=4,000; monthly=333.33
			wantMonthlyIncomeTax: cents(33333),
		},
		{
			name:                 "30% bracket — annual taxable 98,294.40 (worked example 1)",
			monthlyNetTaxable:    mad(8_191.20), // annual=98,294.40; income tax=(98,294.40×30%)−18,000=11,488.32; monthly=957.36
			wantMonthlyIncomeTax: cents(95736),
		},
		{
			name:                 "37% bracket — annual taxable 212,942.40 (worked example 2)",
			monthlyNetTaxable:    mad(17_745.20), // annual=212,942.40; income tax=(212,942.40×37%)−27,400=51,388.69; monthly=4,282.39
			wantMonthlyIncomeTax: cents(428239),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := calculateIncomeTax(tt.monthlyNetTaxable)
			require.NoError(t, err)
			assert.Equal(t, tt.wantMonthlyIncomeTax, got)
		})
	}
}

// ==========================================================================
// Unit tests: roundToNearestDirham
// ==========================================================================

func TestRoundToNearestDirham(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input int64
		want  int64
	}{
		{100_00, 100_00},     // already whole
		{100_49, 100_00},     // round down
		{100_50, 101_00},     // round up
		{100_51, 101_00},     // round up
		{99_84, 100_00},      // round up
		{9696_93, 9697_00},   // worked example 1 rounding
		{15962_81, 15963_00}, // worked example 2 rounding
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			got := roundToNearestDirham(cents(tt.input))
			assert.Equal(t, tt.want, got.Cents(), "input=%d", tt.input)
		})
	}
}

// ==========================================================================
// Integration tests: full worked examples
// ==========================================================================

func newTestPeriod(year, month int) *domain.PayrollPeriod {
	return &domain.PayrollPeriod{
		ID:    uuid.New(),
		OrgID: uuid.New(),
		Year:  year,
		Month: month,
	}
}

func newTestEmployee(hireDate time.Time, numDependents int) *domain.Employee {
	return &domain.Employee{
		ID:            uuid.New(),
		HireDate:      hireDate,
		NumDependents: numDependents,
	}
}

func newTestPackage(baseSalaryMAD float64) *domain.EmployeeCompensationPackage {
	return &domain.EmployeeCompensationPackage{
		ID:         uuid.New(),
		BaseSalary: mad(baseSalaryMAD),
		Currency:   money.MAD,
	}
}

// TestCalculate_WorkedExample1 validates the full calculation against the first
// worked example in DOMAIN.md:
// Base: 10,000 MAD, seniority 6 years, 2 dependents.
func TestCalculate_WorkedExample1(t *testing.T) {
	t.Parallel()

	// Period: January 2026
	period := newTestPeriod(2026, 1)

	// Employee hired 6 years before period start
	hireDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	emp := newTestEmployee(hireDate, 2)

	pkg := newTestPackage(10_000)

	calc := New()
	result, err := calc.Calculate(context.Background(), period, emp, pkg)
	require.NoError(t, err)

	// Salary components
	assert.Equal(t, mad(10_000), result.BaseSalary, "base salary")
	assert.Equal(t, mad(1_000), result.SeniorityBonus, "seniority bonus: 10,000 × 10%")
	assert.Equal(t, mad(11_000), result.GrossSalary, "gross salary")

	// CNSS employee
	assert.Equal(t, mad(268.80), result.SocialAllowanceEmployeeContrib, "social allowance employee")
	assert.Equal(t, mad(11.40), result.JobLossCompensationEmployeeContrib, "job loss compensation employee")
	assert.Equal(t, mad(280.20), result.TotalCNSSEmployeeContrib, "total cnss employee")

	// AMO employee
	assert.Equal(t, mad(248.60), result.AMOEmployeeContrib, "amo employee")

	// Income Tax inputs
	assert.Equal(t, mad(2_280), result.TotalExemptions, "total exemptions: 2,200 prof + 80 family")
	assert.Equal(t, mad(8_191.20), result.TaxableNetSalary, "net taxable salary")

	// Income Tax
	assert.Equal(t, cents(95736), result.IncomeTax, "monthly income tax")

	// Net to pay
	assert.Equal(t, int64(16), result.RoundingAmount.Cents(), "rounding amount: +16 cents")
	assert.Equal(t, mad(9_514), result.NetToPay, "net to pay")

	// CNSS employer
	assert.Equal(t, mad(704), result.FamilyBenefitsEmployerContrib, "family benefits employer")
	assert.Equal(t, mad(538.80), result.SocialAllowanceEmployerContrib, "social allowance employer")
	assert.Equal(t, mad(22.80), result.JobLossCompensationEmployerContrib, "job loss compensation employer")
	assert.Equal(t, mad(176), result.TrainingTaxEmployerContrib, "training tax employer")
	assert.Equal(t, mad(1_441.60), result.TotalCNSSEmployerContrib, "total cnss employer")

	// AMO employer
	assert.Equal(t, mad(452.10), result.AMOEmployerContrib, "amo employer")
}

// TestCalculate_WorkedExample2 validates the full calculation against the second
// worked example in DOMAIN.md:
// Base: 20,000 MAD, seniority 3 years, 0 dependents.
func TestCalculate_WorkedExample2(t *testing.T) {
	t.Parallel()

	// Period: January 2026
	period := newTestPeriod(2026, 1)

	// Employee hired 3 years before period start
	hireDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	emp := newTestEmployee(hireDate, 0)

	pkg := newTestPackage(20_000)

	calc := New()
	result, err := calc.Calculate(context.Background(), period, emp, pkg)
	require.NoError(t, err)

	// Salary components
	assert.Equal(t, mad(20_000), result.BaseSalary, "base salary")
	assert.Equal(t, mad(1_000), result.SeniorityBonus, "seniority bonus: 20,000 × 5%")
	assert.Equal(t, mad(21_000), result.GrossSalary, "gross salary")

	// CNSS employee (capped at 6,000)
	assert.Equal(t, mad(268.80), result.SocialAllowanceEmployeeContrib, "social allowance employee")
	assert.Equal(t, mad(11.40), result.JobLossCompensationEmployeeContrib, "job loss compensation employee")
	assert.Equal(t, mad(280.20), result.TotalCNSSEmployeeContrib, "total cnss employee")

	// AMO employee
	assert.Equal(t, mad(474.60), result.AMOEmployeeContrib, "amo employee: 21,000 × 2.26%")

	// Income Tax inputs — professional expense hits cap at 2,500
	assert.Equal(t, mad(2_500), result.TotalExemptions, "total exemptions: 2,500 prof + 0 family")
	assert.Equal(t, mad(17_745.20), result.TaxableNetSalary, "net taxable salary")

	// Income Tax — top bracket 37%
	assert.Equal(t, cents(428239), result.IncomeTax, "monthly income tax")

	// Net to pay
	assert.Equal(t, int64(19), result.RoundingAmount.Cents(), "rounding amount: +19 cents")
	assert.Equal(t, mad(15_963), result.NetToPay, "net to pay")

	// CNSS employer
	assert.Equal(t, mad(1_344), result.FamilyBenefitsEmployerContrib, "family benefits employer: 21,000 × 6.40%")
	assert.Equal(t, mad(538.80), result.SocialAllowanceEmployerContrib, "social allowance employer: 6,000 × 8.98%")
	assert.Equal(t, mad(22.80), result.JobLossCompensationEmployerContrib, "job loss compensation employer: 6,000 × 0.38%")
	assert.Equal(t, mad(336), result.TrainingTaxEmployerContrib, "training tax employer: 21,000 × 1.60%")
	assert.Equal(t, mad(2_241.60), result.TotalCNSSEmployerContrib, "total cnss employer")

	// AMO employer
	assert.Equal(t, mad(863.10), result.AMOEmployerContrib, "amo employer: 21,000 × 4.11%")
}
