package morocco

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/iamoeg/bootdev-capstone/internal/domain"
	"github.com/iamoeg/bootdev-capstone/pkg/money"
)

// ============================================================================
// Constants
// ============================================================================
// All rates and thresholds from DOMAIN.md (2026 legislation).
// Update this section when rates change — never hardcode values in logic.

const (
	// CNSS — Social Allowance (Prestations Sociales), capped at cnssMonthlyBaseCeiling
	cnssSocialAllowanceEmployeeRate = 0.0448
	cnssSocialAllowanceEmployerRate = 0.0898

	// CNSS — Job Loss Compensation (Indemnité de Perte d'Emploi — IPE), capped at cnssMonthlyBaseCeiling
	cnssJobLossCompEmployeeRate = 0.0019
	cnssJobLossCompEmployerRate = 0.0038

	// CNSS — Training Tax (Taxe de Formation Professionnelle), no ceiling, employer only
	cnssTrainingTaxEmployerRate = 0.016

	// CNSS — Family Benefits (Prestations Familiales), no ceiling, employer only
	cnssFamilyBenefitsEmployerRate = 0.064

	// Monthly ceiling for capped CNSS components (Prestations Sociales + IPE)
	cnssMonthlyBaseCeilingMAD = 6_000_00 // 6,000.00 MAD in cents

	// AMO (no ceiling)
	amoEmployeeRate = 0.0226
	amoEmployerRate = 0.0411

	// Professional expense deduction
	profExpenseRateHigh        = 0.20      // annual gross > 78,000 MAD
	profExpenseRateLow         = 0.35      // annual gross <= 78,000 MAD
	profExpenseAnnualThreshold = 78_000_00 // 78,000.00 MAD in cents
	profExpenseMonthlyCap      = 2_500_00  // 2,500.00 MAD in cents

	// Family charge deduction
	familyChargePerDependentMAD = 40_00 // 40.00 MAD in cents
	familyChargeMaxDependents   = 6

	// SMIG
	smigMonthlyMAD = 3_422_00 // 3,422.00 MAD in cents
)

// Income tax brackets (2026). Applied to annualized net taxable salary.
// Formula: annual_ir = (annual_taxable × rate) − deduction
type incomeTaxBracket struct {
	upperBound int64 // exclusive upper bound in cents; -1 means no upper bound
	rate       float64
	deduction  int64 // fixed deduction in cents
}

var incomeTaxBrackets = []incomeTaxBracket{
	{upperBound: 40_000_00, rate: 0.00, deduction: 0},
	{upperBound: 60_000_00, rate: 0.10, deduction: 4_000_00},
	{upperBound: 80_000_00, rate: 0.20, deduction: 10_000_00},
	{upperBound: 100_000_00, rate: 0.30, deduction: 18_000_00},
	{upperBound: 180_000_00, rate: 0.34, deduction: 22_000_00},
	{upperBound: -1, rate: 0.37, deduction: 27_400_00},
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

// Calculator implements Moroccan payroll calculations for 2026.
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

	// ── Step 4: CNSS employee ─────────────────────────────────────────────
	cnssEmp, err := calculateCNSSEmployee(grossSalary)
	if err != nil {
		return nil, fmt.Errorf("cnss employee: %w", err)
	}

	// ── Step 5: AMO employee ──────────────────────────────────────────────
	amoEmp, err := calculateAMOEmployee(grossSalary)
	if err != nil {
		return nil, fmt.Errorf("amo employee: %w", err)
	}

	// ── Step 6: Professional expense deduction ────────────────────────────
	profExpense, err := calculateProfessionalExpenseDeduction(grossSalary)
	if err != nil {
		return nil, fmt.Errorf("professional expense deduction: %w", err)
	}

	// ── Step 7: Family charge deduction ───────────────────────────────────
	familyCharge, err := calculateFamilyChargeDeduction(emp.NumDependents)
	if err != nil {
		return nil, fmt.Errorf("family charge deduction: %w", err)
	}

	// ── Step 8: Net taxable salary ────────────────────────────────────────
	netTaxable, err := calculateNetTaxableSalary(grossSalary, cnssEmp.total, amoEmp, profExpense, familyCharge)
	if err != nil {
		return nil, fmt.Errorf("net taxable salary: %w", err)
	}

	// ── Step 9: IR ────────────────────────────────────────────────────────
	ir, err := calculateIncomeTax(netTaxable)
	if err != nil {
		return nil, fmt.Errorf("ir: %w", err)
	}

	// ── Step 10: Net to pay ───────────────────────────────────────────────
	netToPay, roundingAmount, err := calculateNetToPay(grossSalary, cnssEmp.total, amoEmp, ir)
	if err != nil {
		return nil, fmt.Errorf("net to pay: %w", err)
	}

	// ── Step 11: Employer contributions ──────────────────────────────────
	cnssEmr, err := calculateCNSSEmployer(grossSalary)
	if err != nil {
		return nil, fmt.Errorf("cnss employer: %w", err)
	}
	amoEmr, err := calculateAMOEmployer(grossSalary)
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
		TotalOtherBonus:       money.FromCents(0),
		GrossSalaryGrandTotal: grossSalary, // no other bonuses for now

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
func calculateCNSSEmployee(gross money.Money) (cnssEmployeeResult, error) {
	cappedBase := capAt(gross, cnssMonthlyBaseCeilingMAD)

	socialAllowance, err := cappedBase.Multiply(cnssSocialAllowanceEmployeeRate)
	if err != nil {
		return cnssEmployeeResult{}, fmt.Errorf("social allowance (employee): %w", err)
	}

	jobLossComp, err := cappedBase.Multiply(cnssJobLossCompEmployeeRate)
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
func calculateCNSSEmployer(gross money.Money) (cnssEmployerResult, error) {
	cappedBase := capAt(gross, cnssMonthlyBaseCeilingMAD)

	familyBenefits, err := gross.Multiply(cnssFamilyBenefitsEmployerRate)
	if err != nil {
		return cnssEmployerResult{}, fmt.Errorf("family benefits (employer): %w", err)
	}

	socialAllowance, err := cappedBase.Multiply(cnssSocialAllowanceEmployerRate)
	if err != nil {
		return cnssEmployerResult{}, fmt.Errorf("social allowance (employer): %w", err)
	}

	jobLossComp, err := cappedBase.Multiply(cnssJobLossCompEmployerRate)
	if err != nil {
		return cnssEmployerResult{}, fmt.Errorf("job loss compensation (employer): %w", err)
	}

	trainingTax, err := gross.Multiply(cnssTrainingTaxEmployerRate)
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
func calculateAMOEmployee(gross money.Money) (money.Money, error) {
	return gross.Multiply(amoEmployeeRate)
}

// calculateAMOEmployer returns the employer's AMO contribution.
func calculateAMOEmployer(gross money.Money) (money.Money, error) {
	return gross.Multiply(amoEmployerRate)
}

// calculateProfessionalExpenseDeduction returns the professional expense
// deduction. The rate depends on whether annualised gross exceeds 78,000 MAD.
// Evaluated monthly using gross × 12 as the annual proxy.
func calculateProfessionalExpenseDeduction(gross money.Money) (money.Money, error) {
	annualGross, err := gross.Multiply(12)
	if err != nil {
		return money.Money{}, fmt.Errorf("annualize gross: %w", err)
	}

	rate := profExpenseRateHigh
	if annualGross.Cents() <= profExpenseAnnualThreshold {
		rate = profExpenseRateLow
	}

	deduction, err := gross.Multiply(rate)
	if err != nil {
		return money.Money{}, fmt.Errorf("professional expense deduction: %w", err)
	}

	return capAt(deduction, profExpenseMonthlyCap), nil
}

// calculateFamilyChargeDeduction returns the family charge deduction based on
// the number of dependents, capped at familyChargeMaxDependents.
func calculateFamilyChargeDeduction(numDependents int) (money.Money, error) {
	capped := numDependents
	if capped > familyChargeMaxDependents {
		capped = familyChargeMaxDependents
	}
	return money.FromCents(int64(capped) * familyChargePerDependentMAD), nil
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
// taxable salary, using annualised progressive brackets.
func calculateIncomeTax(monthlyNetTaxable money.Money) (money.Money, error) {
	annualTaxable, err := monthlyNetTaxable.Multiply(12)
	if err != nil {
		return money.Money{}, fmt.Errorf("annualize net taxable: %w", err)
	}

	bracket := findIncomeTaxBracket(annualTaxable.Cents())

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

	monthlyIncomeTax, err := annualIncomeTax.Divide(12)
	if err != nil {
		return money.Money{}, fmt.Errorf("monthly income tax: %w", err)
	}

	return monthlyIncomeTax, nil
}

// findIncomeTaxBracket returns the IR bracket applicable to the given annual taxable
// income in cents.
func findIncomeTaxBracket(annualTaxableCents int64) incomeTaxBracket {
	for _, b := range incomeTaxBrackets {
		if b.upperBound == -1 || annualTaxableCents <= b.upperBound {
			return b
		}
	}
	// Unreachable: last bracket has no upper bound
	return incomeTaxBrackets[len(incomeTaxBrackets)-1]
}

// calculateNetToPay returns the net to pay and the rounding adjustment applied.
func calculateNetToPay(
	gross, cnssEmployee, amoEmployee, incomeTax money.Money,
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
