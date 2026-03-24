package calculator

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/iamoeg/bootdev-capstone/internal/domain"
	"github.com/iamoeg/bootdev-capstone/pkg/money"
)

// ============================================================================
// Year-specific rate tables
// ============================================================================
// yearRates holds all legislation-specific values for a calendar year.
// Add a new entry to ratesByYear when a new fiscal year is supported.
// Structural constants (seniority tier thresholds, rounding rules) live below.

type yearRates struct {
	// CNSS employee contributions
	CnssSocialAllowanceEmployee float64
	CnssJobLossCompEmployee     float64

	// CNSS employer contributions
	CnssSocialAllowanceEmployer float64
	CnssJobLossCompEmployer     float64
	CnssTrainingTaxEmployer     float64
	CnssFamilyBenefitsEmployer  float64

	// CNSS monthly base ceiling for capped components (cents)
	CnssMonthlyBaseCeiling int64

	// AMO
	AmoEmployee float64
	AmoEmployer float64

	// Professional expense deduction
	ProfExpenseRateAbove       float64
	ProfExpenseRateBelow       float64
	ProfExpenseAnnualThreshold int64 // cents; annual gross threshold
	ProfExpenseMonthlyCap      int64 // cents

	// Family charge deduction (IR)
	FamilyChargePerDependent  int64 // cents per dependent
	FamilyChargeMaxDependents int

	// Family allowance (CNSS cash payment, tax-exempt)
	FamilyAllowanceLowTierPerChild  int64 // cents, children 1–FamilyAllowanceLowTierLimit
	FamilyAllowanceHighTierPerChild int64 // cents, remaining children up to max
	FamilyAllowanceLowTierLimit     int
	FamilyAllowanceMaxChildren      int

	// Legal minimum wage (SMIG)
	SmigMonthly int64 // cents

	// Progressive income tax (IR) brackets
	IncomeTaxBrackets []incomeTaxBracket
}

// ratesByYear maps a calendar year to its legislation-specific rate table.
// Add a new entry here when a new fiscal year is supported.
var ratesByYear = map[int]yearRates{
	2026: {
		// CNSS employee
		CnssSocialAllowanceEmployee: 0.0448,
		CnssJobLossCompEmployee:     0.0019,

		// CNSS employer
		CnssSocialAllowanceEmployer: 0.0898,
		CnssJobLossCompEmployer:     0.0038,
		CnssTrainingTaxEmployer:     0.016,
		CnssFamilyBenefitsEmployer:  0.064,

		// CNSS ceiling
		CnssMonthlyBaseCeiling: 6_000_00,

		// AMO
		AmoEmployee: 0.0226,
		AmoEmployer: 0.0411,

		// Professional expense
		ProfExpenseRateAbove:       0.20,
		ProfExpenseRateBelow:       0.35,
		ProfExpenseAnnualThreshold: 78_000_00,
		ProfExpenseMonthlyCap:      2_500_00,

		// Family charge deduction
		FamilyChargePerDependent:  40_00,
		FamilyChargeMaxDependents: 6,

		// Family allowance
		FamilyAllowanceLowTierPerChild:  300_00,
		FamilyAllowanceHighTierPerChild: 36_00,
		FamilyAllowanceLowTierLimit:     3,
		FamilyAllowanceMaxChildren:      6,

		// SMIG
		SmigMonthly: 3_422_00,

		// IR brackets
		IncomeTaxBrackets: []incomeTaxBracket{
			{upperBound: 40_000_00, rate: 0.00, deduction: 0},
			{upperBound: 60_000_00, rate: 0.10, deduction: 4_000_00},
			{upperBound: 80_000_00, rate: 0.20, deduction: 10_000_00},
			{upperBound: 100_000_00, rate: 0.30, deduction: 18_000_00},
			{upperBound: 180_000_00, rate: 0.34, deduction: 22_000_00},
			{upperBound: -1, rate: 0.37, deduction: 27_400_00},
		},
	},
}

// ============================================================================
// Errors
// ============================================================================

// ErrUnsupportedPayrollYear is returned when no rate table exists for period.Year.
var ErrUnsupportedPayrollYear = errors.New("calculator: no rate table for this payroll year")

// ErrGrossSalaryBelowSMIG is returned when the computed gross salary falls
// below the legal minimum wage (SMIG). This should not happen if domain
// validation is enforced, but the calculator checks it as a second line of
// defense against bypassed validation or stale compensation packages.
var ErrGrossSalaryBelowSMIG = errors.New("calculator: gross salary is below the legal minimum wage (SMIG)")

// ============================================================================
// Structural types and constants
// ============================================================================
// These are algorithm-structural and do not vary by year.

// Income tax bracket type.
type incomeTaxBracket struct {
	upperBound int64 // inclusive upper bound in cents; -1 means no upper bound
	rate       float64
	deduction  int64 // fixed deduction in cents
}

// Seniority bonus tiers. Lower bound inclusive, upper bound exclusive.
type seniorityTier struct {
	minYears int
	maxYears int // -1 means no upper bound
	rate     float64
}

var seniorityTiers = []seniorityTier{
	{minYears: 0, maxYears: 2, rate: 0.00},
	{minYears: 2, maxYears: 5, rate: 0.05},
	{minYears: 5, maxYears: 12, rate: 0.10},
	{minYears: 12, maxYears: 20, rate: 0.15},
	{minYears: 20, maxYears: 25, rate: 0.20},
	{minYears: 25, maxYears: -1, rate: 0.25},
}

// ============================================================================
// Public API
// ============================================================================

// Calculator implements Moroccan payroll calculations.
// Supported years are defined in ratesByYear.
// It satisfies the payrollCalculator interface in application/payroll_service.go.
type Calculator struct{}

// New returns a new Calculator.
func New() *Calculator {
	return &Calculator{}
}

// Calculate computes a complete PayrollResult for the given employee and
// compensation package within the given period. All intermediate values are
// stored in the result for auditability.
func (c *Calculator) Calculate(
	_ context.Context,
	period *domain.PayrollPeriod,
	emp *domain.Employee,
	pkg *domain.EmployeeCompensationPackage,
) (*domain.PayrollResult, error) {
	rates, ok := ratesByYear[period.Year]
	if !ok {
		return nil, fmt.Errorf("%w: %d", ErrUnsupportedPayrollYear, period.Year)
	}

	// ── Step 1: Base salary ───────────────────────────────────────────────
	baseSalary := pkg.BaseSalary

	// ── Step 2: Seniority bonus ───────────────────────────────────────────
	yearsOfService := completedYears(emp.HireDate, periodDate(period))
	seniorityBonus, err := calculateSeniorityBonus(baseSalary, yearsOfService)
	if err != nil {
		return nil, fmt.Errorf("seniority bonus: %w", err)
	}

	// ── Step 3: Gross salary ──────────────────────────────────────────────
	grossSalary, err := baseSalary.Add(seniorityBonus)
	if err != nil {
		return nil, fmt.Errorf("gross salary: %w", err)
	}
	if grossSalary.Cents() < rates.SmigMonthly {
		return nil, fmt.Errorf("%w: got %v, minimum is %v",
			ErrGrossSalaryBelowSMIG, grossSalary, money.FromCents(rates.SmigMonthly))
	}

	// ── Step 4: CNSS employee ─────────────────────────────────────────────
	cnssEmp, err := calculateCNSSEmployee(grossSalary, rates)
	if err != nil {
		return nil, fmt.Errorf("cnss employee: %w", err)
	}

	// ── Step 5: AMO employee ──────────────────────────────────────────────
	amoEmp, err := calculateAMOEmployee(grossSalary, rates)
	if err != nil {
		return nil, fmt.Errorf("amo employee: %w", err)
	}

	// ── Step 6: Professional expense deduction ────────────────────────────
	profExpense, err := calculateProfessionalExpenseDeduction(grossSalary, rates)
	if err != nil {
		return nil, fmt.Errorf("professional expense deduction: %w", err)
	}

	// ── Step 7: Family charge deduction ───────────────────────────────────
	familyCharge, err := calculateFamilyChargeDeduction(emp.NumDependents, rates)
	if err != nil {
		return nil, fmt.Errorf("family charge deduction: %w", err)
	}

	// ── Step 8: Net taxable salary ────────────────────────────────────────
	netTaxable, err := calculateNetTaxableSalary(grossSalary, cnssEmp.total, amoEmp, profExpense, familyCharge)
	if err != nil {
		return nil, fmt.Errorf("net taxable salary: %w", err)
	}

	// ── Step 9: IR ────────────────────────────────────────────────────────
	ir, err := calculateIncomeTax(netTaxable, rates)
	if err != nil {
		return nil, fmt.Errorf("ir: %w", err)
	}

	// ── Step 10: Family allowance (Allocations Familiales) ────────────────
	// Tax-exempt cash payment to employee; does not affect CNSS or IR base.
	familyAllowance := calculateFamilyAllowance(emp.NumChildren, rates)

	// ── Step 11: Net to pay ───────────────────────────────────────────────
	netToPay, roundingAmount, err := calculateNetToPay(grossSalary, cnssEmp.total, amoEmp, ir, familyAllowance)
	if err != nil {
		return nil, fmt.Errorf("net to pay: %w", err)
	}

	// ── Step 12: Employer contributions ──────────────────────────────────
	cnssEmr, err := calculateCNSSEmployer(grossSalary, rates)
	if err != nil {
		return nil, fmt.Errorf("cnss employer: %w", err)
	}
	amoEmr, err := calculateAMOEmployer(grossSalary, rates)
	if err != nil {
		return nil, fmt.Errorf("amo employer: %w", err)
	}

	// ── Totals ────────────────────────────────────────────────────────────
	totalCNSSEmp, err := cnssEmp.socialAllowance.Add(cnssEmp.jobLossCompensation)
	if err != nil {
		return nil, fmt.Errorf("total cnss employee: %w", err)
	}

	totalCNSSEmr, err := cnssEmr.familyBenefits.Add(cnssEmr.socialAllowance)
	if err != nil {
		return nil, fmt.Errorf("total cnss employer step 1: %w", err)
	}
	totalCNSSEmr, err = totalCNSSEmr.Add(cnssEmr.jobLossCompensation)
	if err != nil {
		return nil, fmt.Errorf("total cnss employer step 2: %w", err)
	}
	totalCNSSEmr, err = totalCNSSEmr.Add(cnssEmr.trainingTax)
	if err != nil {
		return nil, fmt.Errorf("total cnss employer step 3: %w", err)
	}

	// Total exemptions = professional expenses + family charges
	totalExemptions, err := profExpense.Add(familyCharge)
	if err != nil {
		return nil, fmt.Errorf("total exemptions: %w", err)
	}

	// Taxable gross = gross − exemptions
	taxableGross, err := grossSalary.Subtract(totalExemptions)
	if err != nil {
		return nil, fmt.Errorf("taxable gross: %w", err)
	}

	// Grand total = gross salary + any other bonuses
	totalOtherBonus := money.FromCents(0)
	grossSalaryGrandTotal, err := grossSalary.Add(totalOtherBonus)
	if err != nil {
		return nil, fmt.Errorf("gross salary grand total: %w", err)
	}

	// ── Assemble result ───────────────────────────────────────────────────
	now := time.Now().UTC()
	result := &domain.PayrollResult{
		ID:                    uuid.New(),
		PayrollPeriodID:       period.ID,
		EmployeeID:            emp.ID,
		CompensationPackageID: pkg.ID,
		Currency:              pkg.Currency,

		// Salary components
		BaseSalary:            baseSalary,
		SeniorityBonus:        seniorityBonus,
		GrossSalary:           grossSalary,
		TotalOtherBonus:       totalOtherBonus,
		GrossSalaryGrandTotal: grossSalaryGrandTotal,

		// CNSS employee
		SocialAllowanceEmployeeContrib:     cnssEmp.socialAllowance,
		JobLossCompensationEmployeeContrib: cnssEmp.jobLossCompensation,
		TotalCNSSEmployeeContrib:           totalCNSSEmp,
		AMOEmployeeContrib:                 amoEmp,

		// CNSS employer
		SocialAllowanceEmployerContrib:     cnssEmr.socialAllowance,
		JobLossCompensationEmployerContrib: cnssEmr.jobLossCompensation,
		TrainingTaxEmployerContrib:         cnssEmr.trainingTax,
		FamilyBenefitsEmployerContrib:      cnssEmr.familyBenefits,
		TotalCNSSEmployerContrib:           totalCNSSEmr,
		AMOEmployerContrib:                 amoEmr,

		// Family allowance (income to employee, tax-exempt)
		FamilyAllowance: familyAllowance,

		// Tax
		TotalExemptions:    totalExemptions,
		TaxableGrossSalary: taxableGross,
		TaxableNetSalary:   netTaxable,
		IncomeTax:          ir,

		// Final
		RoundingAmount: roundingAmount,
		NetToPay:       netToPay,

		CreatedAt: now,
		UpdatedAt: now,
	}

	return result, nil
}

// ============================================================================
// Private helpers
// ============================================================================

// cnssEmployeeResult holds the individual CNSS employee contribution components.
type cnssEmployeeResult struct {
	socialAllowance     money.Money
	jobLossCompensation money.Money
	total               money.Money
}

// cnssEmployerResult holds the individual CNSS employer contribution components.
type cnssEmployerResult struct {
	familyBenefits      money.Money
	socialAllowance     money.Money
	jobLossCompensation money.Money
	trainingTax         money.Money
}

// completedYears returns the number of fully completed years between from and to.
// Uses truncated arithmetic — 4 years and 11 months = 4.
func completedYears(from, to time.Time) int {
	years := to.Year() - from.Year()
	// Step back one year if the anniversary hasn't occurred yet this year
	if to.Month() < from.Month() || (to.Month() == from.Month() && to.Day() < from.Day()) {
		years--
	}
	if years < 0 {
		return 0
	}
	return years
}

// periodDate returns the first day of the payroll period, used as the
// reference date for seniority calculations.
func periodDate(period *domain.PayrollPeriod) time.Time {
	return time.Date(period.Year, time.Month(period.Month), 1, 0, 0, 0, 0, time.UTC)
}

// calculateSeniorityBonus returns the seniority bonus for the given base salary
// and completed years of service.
func calculateSeniorityBonus(baseSalary money.Money, yearsOfService int) (money.Money, error) {
	rate := seniorityRate(yearsOfService)
	return baseSalary.Multiply(rate)
}

// seniorityRate returns the applicable seniority bonus rate for the given
// completed years of service.
func seniorityRate(years int) float64 {
	for _, tier := range seniorityTiers {
		if tier.maxYears == -1 || years < tier.maxYears {
			if years >= tier.minYears {
				return tier.rate
			}
		}
	}
	return 0
}

// calculateCNSSEmployee returns the employee's CNSS contributions broken down
// by component. Both Prestations Sociales and IPE are applied to the capped base.
func calculateCNSSEmployee(gross money.Money, r yearRates) (cnssEmployeeResult, error) {
	cappedBase := capAt(gross, r.CnssMonthlyBaseCeiling)

	socialAllowance, err := cappedBase.Multiply(r.CnssSocialAllowanceEmployee)
	if err != nil {
		return cnssEmployeeResult{}, fmt.Errorf("social allowance (employee): %w", err)
	}

	jobLossComp, err := cappedBase.Multiply(r.CnssJobLossCompEmployee)
	if err != nil {
		return cnssEmployeeResult{}, fmt.Errorf("job loss compensation (employee): %w", err)
	}

	total, err := socialAllowance.Add(jobLossComp)
	if err != nil {
		return cnssEmployeeResult{}, fmt.Errorf("total cnss (employee): %w", err)
	}

	return cnssEmployeeResult{
		socialAllowance:     socialAllowance,
		jobLossCompensation: jobLossComp,
		total:               total,
	}, nil
}

// calculateCNSSEmployer returns the employer's CNSS contributions broken down
// by component.
func calculateCNSSEmployer(gross money.Money, r yearRates) (cnssEmployerResult, error) {
	cappedBase := capAt(gross, r.CnssMonthlyBaseCeiling)

	familyBenefits, err := gross.Multiply(r.CnssFamilyBenefitsEmployer)
	if err != nil {
		return cnssEmployerResult{}, fmt.Errorf("family benefits (employer): %w", err)
	}

	socialAllowance, err := cappedBase.Multiply(r.CnssSocialAllowanceEmployer)
	if err != nil {
		return cnssEmployerResult{}, fmt.Errorf("social allowance (employer): %w", err)
	}

	jobLossComp, err := cappedBase.Multiply(r.CnssJobLossCompEmployer)
	if err != nil {
		return cnssEmployerResult{}, fmt.Errorf("job loss compensation (employer): %w", err)
	}

	trainingTax, err := gross.Multiply(r.CnssTrainingTaxEmployer)
	if err != nil {
		return cnssEmployerResult{}, fmt.Errorf("training tax employer: %w", err)
	}

	return cnssEmployerResult{
		familyBenefits:      familyBenefits,
		socialAllowance:     socialAllowance,
		jobLossCompensation: jobLossComp,
		trainingTax:         trainingTax,
	}, nil
}

// calculateAMOEmployee returns the employee's AMO contribution.
func calculateAMOEmployee(gross money.Money, r yearRates) (money.Money, error) {
	return gross.Multiply(r.AmoEmployee)
}

// calculateAMOEmployer returns the employer's AMO contribution.
func calculateAMOEmployer(gross money.Money, r yearRates) (money.Money, error) {
	return gross.Multiply(r.AmoEmployer)
}

// calculateProfessionalExpenseDeduction returns the professional expense
// deduction. The rate depends on whether annualized gross exceeds the threshold.
// Evaluated monthly using gross × 12 as the annual proxy.
func calculateProfessionalExpenseDeduction(gross money.Money, r yearRates) (money.Money, error) {
	annualGross, err := gross.Multiply(12)
	if err != nil {
		return money.Money{}, fmt.Errorf("annualized gross: %w", err)
	}

	rate := r.ProfExpenseRateAbove
	if annualGross.Cents() <= r.ProfExpenseAnnualThreshold {
		rate = r.ProfExpenseRateBelow
	}

	deduction, err := gross.Multiply(rate)
	if err != nil {
		return money.Money{}, fmt.Errorf("professional expense deduction: %w", err)
	}

	return capAt(deduction, r.ProfExpenseMonthlyCap), nil
}

// calculateFamilyChargeDeduction returns the family charge deduction based on
// the number of dependents, capped at r.FamilyChargeMaxDependents.
func calculateFamilyChargeDeduction(numDependents int, r yearRates) (money.Money, error) {
	capped := numDependents
	if capped > r.FamilyChargeMaxDependents {
		capped = r.FamilyChargeMaxDependents
	}
	return money.FromCents(int64(capped) * r.FamilyChargePerDependent), nil
}

// calculateFamilyAllowance returns the monthly CNSS family allowance (allocations familiales)
// paid to the employee based on their number of qualifying children.
// This amount is tax-exempt and increases net pay directly.
func calculateFamilyAllowance(numChildren int, r yearRates) money.Money {
	if numChildren <= 0 {
		return money.FromCents(0)
	}
	capped := numChildren
	if capped > r.FamilyAllowanceMaxChildren {
		capped = r.FamilyAllowanceMaxChildren
	}
	lowTier := capped
	if lowTier > r.FamilyAllowanceLowTierLimit {
		lowTier = r.FamilyAllowanceLowTierLimit
	}
	highTier := capped - lowTier
	total := int64(lowTier)*r.FamilyAllowanceLowTierPerChild + int64(highTier)*r.FamilyAllowanceHighTierPerChild
	return money.FromCents(total)
}

// calculateNetTaxableSalary computes the net taxable salary used as the IR base.
func calculateNetTaxableSalary(
	gross, cnssEmployee, amoEmployee, profExpense, familyCharge money.Money,
) (money.Money, error) {
	result, err := gross.Subtract(cnssEmployee)
	if err != nil {
		return money.Money{}, err
	}
	result, err = result.Subtract(amoEmployee)
	if err != nil {
		return money.Money{}, err
	}
	result, err = result.Subtract(profExpense)
	if err != nil {
		return money.Money{}, err
	}
	result, err = result.Subtract(familyCharge)
	if err != nil {
		return money.Money{}, err
	}
	return result, nil
}

// calculateIncomeTax returns the monthly IR (income tax) for the given monthly net
// taxable salary, using annualized progressive brackets from r.
func calculateIncomeTax(monthlyNetTaxable money.Money, r yearRates) (money.Money, error) {
	annualTaxable, err := monthlyNetTaxable.Multiply(12)
	if err != nil {
		return money.Money{}, fmt.Errorf("annualized net taxable: %w", err)
	}

	bracket := findIncomeTaxBracket(annualTaxable.Cents(), r.IncomeTaxBrackets)

	if bracket.rate == 0 {
		return money.FromCents(0), nil
	}

	annualIncomeTax, err := annualTaxable.Multiply(bracket.rate)
	if err != nil {
		return money.Money{}, fmt.Errorf("apply income tax rate: %w", err)
	}

	annualIncomeTax, err = annualIncomeTax.Subtract(money.FromCents(bracket.deduction))
	if err != nil {
		return money.Money{}, fmt.Errorf("apply income tax deduction: %w", err)
	}

	// The bracket structure guarantees non-negative tax for valid inputs, but
	// clamp defensively in case brackets are ever updated incorrectly.
	if annualIncomeTax.IsNegative() {
		annualIncomeTax = money.FromCents(0)
	}

	monthlyIncomeTax, err := annualIncomeTax.Divide(12)
	if err != nil {
		return money.Money{}, fmt.Errorf("monthly income tax: %w", err)
	}

	return monthlyIncomeTax, nil
}

// findIncomeTaxBracket returns the IR bracket applicable to the given annual taxable
// income in cents.
func findIncomeTaxBracket(annualTaxableCents int64, brackets []incomeTaxBracket) incomeTaxBracket {
	for _, b := range brackets {
		if b.upperBound == -1 || annualTaxableCents <= b.upperBound {
			return b
		}
	}
	// Unreachable: last bracket has no upper bound
	return brackets[len(brackets)-1]
}

// calculateNetToPay returns the net to pay and the rounding adjustment applied.
// familyAllowance (allocations familiales) is added after deductions — it is
// tax-exempt and not subject to CNSS contributions.
func calculateNetToPay(
	gross, cnssEmployee, amoEmployee, incomeTax, familyAllowance money.Money,
) (netToPay, roundingAmount money.Money, err error) {
	raw, err := gross.Subtract(cnssEmployee)
	if err != nil {
		return money.Money{}, money.Money{}, err
	}
	raw, err = raw.Subtract(amoEmployee)
	if err != nil {
		return money.Money{}, money.Money{}, err
	}
	raw, err = raw.Subtract(incomeTax)
	if err != nil {
		return money.Money{}, money.Money{}, err
	}
	raw, err = raw.Add(familyAllowance)
	if err != nil {
		return money.Money{}, money.Money{}, err
	}

	rounded := roundToNearestDirham(raw)

	rounding, err := rounded.Subtract(raw)
	if err != nil {
		return money.Money{}, money.Money{}, fmt.Errorf("rounding amount: %w", err)
	}

	return rounded, rounding, nil
}

// roundToNearestDirham rounds a Money value to the nearest whole dirham
// (nearest 100 cents).
func roundToNearestDirham(m money.Money) money.Money {
	cents := m.Cents()
	remainder := cents % 100
	if remainder == 0 {
		return m
	}
	if remainder >= 50 {
		return money.FromCents(cents - remainder + 100)
	}
	if remainder <= -50 {
		return money.FromCents(cents - remainder - 100)
	}
	return money.FromCents(cents - remainder)
}

// capAt returns the smaller of m and a ceiling expressed in cents.
func capAt(m money.Money, ceilingCents int64) money.Money {
	if m.Cents() > ceilingCents {
		return money.FromCents(ceilingCents)
	}
	return m
}
