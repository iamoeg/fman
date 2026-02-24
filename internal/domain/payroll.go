package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/iamoeg/bootdev-capstone/pkg/money"
	"github.com/iamoeg/bootdev-capstone/pkg/util"
)

// ============================================================================
// PayrollPeriod Entity
// ============================================================================

// PayrollPeriod represents a monthly payroll cycle for an organization.
// It serves as a container for all payroll results for employees during
// a specific month.
//
// Workflow:
//  1. Create period with Status = DRAFT
//  2. Generate PayrollResults for all employees
//  3. Review and adjust results as needed
//  4. Finalize: set Status = FINALIZED, set FinalizedAt timestamp
//  5. Once finalized, the period and all its results become immutable
//
// Business Rules:
//   - One period per organization per month
//   - Status and FinalizedAt must be consistent:
//     └─ DRAFT: FinalizedAt must be nil
//     └─ FINALIZED: FinalizedAt must not be nil
//   - FinalizedAt must be >= CreatedAt
//   - FinalizedAt cannot be in the future
type PayrollPeriod struct {
	// Identity
	ID    uuid.UUID
	OrgID uuid.UUID

	// Period Definition
	Year  int // Year of the payroll period (2020-2050)
	Month int // Month of the payroll period (1-12)

	// Status
	Status      PayrollPeriodStatusEnum
	FinalizedAt *time.Time // When the period was finalized (nil if DRAFT)

	// Metadata
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time // Soft delete timestamp
}

// Validate performs comprehensive validation of the payroll period.
func (pp *PayrollPeriod) Validate() error {
	if err := pp.ValidateID(); err != nil {
		return err
	}

	if err := pp.ValidateOrgID(); err != nil {
		return err
	}

	if err := pp.ValidateYear(); err != nil {
		return err
	}

	if err := pp.ValidateMonth(); err != nil {
		return err
	}

	if err := pp.ValidateStatus(); err != nil {
		return err
	}

	if err := pp.ValidateStateConsistency(); err != nil {
		return err
	}

	return nil
}

// ============================================================================
// PayrollPeriod Validation Methods
// ============================================================================

// ValidateID ensures the payroll period has a valid UUID.
func (pp *PayrollPeriod) ValidateID() error {
	if pp.ID == uuid.Nil {
		return ErrPayrollPeriodIDRequired
	}
	return nil
}

// ValidateOrgID ensures the payroll period belongs to an organization.
func (pp *PayrollPeriod) ValidateOrgID() error {
	if pp.OrgID == uuid.Nil {
		return ErrPayrollPeriodOrgIDRequired
	}
	return nil
}

// ValidateYear ensures the year is within the supported range.
func (pp *PayrollPeriod) ValidateYear() error {
	if pp.Year < PayrollPeriodMinYear || pp.Year > PayrollPeriodMaxYear {
		return fmt.Errorf(
			"%w: must be between %v and %v inclusive",
			ErrInvalidPayrollPeriodYear,
			PayrollPeriodMinYear,
			PayrollPeriodMaxYear,
		)
	}
	return nil
}

// ValidateMonth ensures the month is valid (1-12).
func (pp *PayrollPeriod) ValidateMonth() error {
	if pp.Month < 1 || pp.Month > 12 {
		return fmt.Errorf(
			"%w: must be between 1 and 12 inclusive",
			ErrInvalidPayrollPeriodMonth,
		)
	}
	return nil
}

// ValidateStatus ensures the status is one of the supported values.
func (pp *PayrollPeriod) ValidateStatus() error {
	if !pp.Status.IsSupported() {
		return fmt.Errorf(
			"%w: must be one of %v",
			ErrInvalidPayrollPeriodStatus,
			SupportedPayrollPeriodStatusesStr,
		)
	}
	return nil
}

// ValidateStateConsistency ensures Status and FinalizedAt are consistent.
// Valid states:
//   - DRAFT with FinalizedAt = nil
//   - FINALIZED with FinalizedAt != nil
//
// Also validates that:
//   - FinalizedAt >= CreatedAt (if set)
//   - FinalizedAt <= now (if set)
func (pp *PayrollPeriod) ValidateStateConsistency() error {
	// Check Status/FinalizedAt consistency
	if !((pp.Status == PayrollPeriodStatusDraft && pp.FinalizedAt == nil) ||
		(pp.Status == PayrollPeriodStatusFinalized && pp.FinalizedAt != nil)) {
		return fmt.Errorf(
			"%w: .Status and .FinalizedAt are inconsistent. .Status is %v but .FinalizedAt is %v",
			ErrInvalidPayrollPeriodState,
			pp.Status,
			pp.FinalizedAt,
		)
	}

	// If finalized, validate the timestamp
	if pp.FinalizedAt != nil {
		// FinalizedAt must be after creation
		if pp.FinalizedAt.Before(pp.CreatedAt) {
			return fmt.Errorf(
				"%w: .FinalizedAt must be >= %v",
				ErrInvalidPayrollPeriodState,
				pp.CreatedAt,
			)
		}

		// FinalizedAt cannot be in the future
		now := time.Now()
		if pp.FinalizedAt.After(now) {
			return fmt.Errorf(
				"%w: .FinalizedAt must be past (< %v)",
				ErrInvalidPayrollPeriodState,
				now,
			)
		}
	}

	return nil
}

// ============================================================================
// PayrollPeriod Status Enum
// ============================================================================

// PayrollPeriodStatusEnum represents the processing status of a payroll period.
type PayrollPeriodStatusEnum string

const (
	// PayrollPeriodStatusDraft indicates the period is still being processed.
	// Results can be modified or regenerated.
	PayrollPeriodStatusDraft PayrollPeriodStatusEnum = "DRAFT"

	// PayrollPeriodStatusFinalized indicates the period is complete and locked.
	// No modifications are allowed - results are immutable historical records.
	PayrollPeriodStatusFinalized PayrollPeriodStatusEnum = "FINALIZED"
)

var supportedPayrollPeriodStatuses = map[PayrollPeriodStatusEnum]struct{}{
	PayrollPeriodStatusDraft:     {},
	PayrollPeriodStatusFinalized: {},
}

// SupportedPayrollPeriodStatusesStr is a comma-separated list of supported status values.
var SupportedPayrollPeriodStatusesStr = util.EnumMapToString(supportedPayrollPeriodStatuses)

// IsSupported returns true if the status is one of the supported values.
func (ps PayrollPeriodStatusEnum) IsSupported() bool {
	_, ok := supportedPayrollPeriodStatuses[ps]
	return ok
}

// ============================================================================
// PayrollPeriod Constants
// ============================================================================

const (
	// PayrollPeriodMinYear is the earliest supported year for payroll periods.
	PayrollPeriodMinYear = 2020

	// PayrollPeriodMaxYear is the latest supported year for payroll periods.
	PayrollPeriodMaxYear = 2050
)

// ============================================================================
// PayrollPeriod Errors
// ============================================================================

var (
	ErrPayrollPeriodIDRequired    = errors.New("domain: payroll: payroll period id (uuid) is required")
	ErrPayrollPeriodOrgIDRequired = errors.New("domain: payroll: payroll period org id (uuid) is required")
	ErrInvalidPayrollPeriodYear   = errors.New("domain: payroll: invalid payroll period year")
	ErrInvalidPayrollPeriodMonth  = errors.New("domain: payroll: invalid payroll period month")
	ErrInvalidPayrollPeriodStatus = errors.New("domain: payroll: invalid payroll period status")
	ErrInvalidPayrollPeriodState  = errors.New("domain: payroll: invalid payroll period state")
)

// ============================================================================
// PayrollResult Entity
// ============================================================================

// PayrollResult represents the complete calculated payroll for one employee
// for one payroll period. It is an immutable historical record once the
// parent PayrollPeriod is finalized.
//
// Design Rationale:
// All calculated fields are stored, not recomputed on demand. This ensures:
//   - Historical accuracy: shows what was actually calculated/paid
//   - Legal compliance: payroll is a legal document that must be preserved
//   - Performance: no need to recalculate for reports
//   - Audit trail: permanent record of calculations
//
// Moroccan Payroll Components:
//
// Gross Salary Calculation:
//   - GrossSalary = BaseSalary + SeniorityBonus
//   - GrossSalaryGrandTotal = GrossSalary + TotalOtherBonus
//
// Social Contributions:
//   - CNSS (Caisse Nationale de Sécurité Sociale):
//     └─ Social Allowance (employee + employer)
//     └─ Job Loss Compensation (IPE - employee + employer)
//     └─ Training Tax (employer only)
//     └─ Family Benefits (employer only)
//   - AMO (Assurance Maladie Obligatoire - Health Insurance):
//     └─ Separate from CNSS but collected by CNSS in practice
//     └─ Employee + employer contributions
//
// Tax Calculation:
//   - TaxableGrossSalary = GrossSalaryGrandTotal - TotalExemptions
//   - TaxableNetSalary = TaxableGrossSalary - TotalCNSSEmployeeContrib - AMOEmployeeContrib
//   - IncomeTax = Progressive tax on TaxableNetSalary (IR - Impôt sur le Revenu)
//
// Final Payment:
//   - NetToPay = TaxableNetSalary - IncomeTax + RoundingAmount
//
// Business Rules:
//   - All monetary amounts must be >= 0 (except RoundingAmount)
//   - RoundingAmount must be between -1 and +1 MAD (-100 to +100 cents)
//   - Mathematical consistency: all formulas must be internally consistent
//   - Currency must be supported
type PayrollResult struct {
	// Identity
	ID                    uuid.UUID
	PayrollPeriodID       uuid.UUID
	EmployeeID            uuid.UUID
	CompensationPackageID uuid.UUID

	// Currency
	Currency money.Currency

	// Salary Components
	BaseSalary            money.Money // Monthly base salary
	SeniorityBonus        money.Money // Seniority bonus (ancienneté)
	GrossSalary           money.Money // Base + Seniority
	TotalOtherBonus       money.Money // Other bonuses (overtime, etc.)
	GrossSalaryGrandTotal money.Money // Total gross including all bonuses

	// Tax Calculations
	TotalExemptions    money.Money // Professional expenses, etc.
	TaxableGrossSalary money.Money // Gross after exemptions
	TaxableNetSalary   money.Money // After CNSS and AMO deductions
	IncomeTax          money.Money // IR (Impôt sur le Revenu)

	// CNSS Contributions (Social Security)
	// Note: CNSS does NOT include AMO - AMO is separate
	SocialAllowanceEmployeeContrib     money.Money // Prestations Sociales (employee part)
	SocialAllowanceEmployerContrib     money.Money // Prestations Sociales (employer part)
	JobLossCompensationEmployeeContrib money.Money // IPE - Indemnité pour Perte d'Emploi (employee)
	JobLossCompensationEmployerContrib money.Money // IPE (employer part)
	TrainingTaxEmployerContrib         money.Money // Taxe de Formation Professionnelle (employer only)
	FamilyBenefitsEmployerContrib      money.Money // Allocations Familiales (employer only)
	TotalCNSSEmployeeContrib           money.Money // Total CNSS employee (excludes AMO)
	TotalCNSSEmployerContrib           money.Money // Total CNSS employer (excludes AMO)

	// AMO Contributions (Health Insurance)
	// AMO is separate from CNSS but collected by CNSS in practice
	AMOEmployeeContrib money.Money // Health insurance (employee part)
	AMOEmployerContrib money.Money // Health insurance (employer part)

	// Final Payment
	RoundingAmount money.Money // Rounding adjustment (-1 to +1 MAD)
	NetToPay       money.Money // Final amount paid to employee

	// Metadata
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time // Soft delete timestamp
}

// Validate performs comprehensive validation of the payroll result.
// It validates:
//   - Required fields (IDs, currency)
//   - All monetary amounts are non-negative (except RoundingAmount)
//   - RoundingAmount is within bounds
//   - Mathematical consistency of all calculations
func (pr *PayrollResult) Validate() error {
	if err := pr.ValidateID(); err != nil {
		return err
	}

	if err := pr.ValidateEmployeeID(); err != nil {
		return err
	}

	if err := pr.ValidatePayrollPeriodID(); err != nil {
		return err
	}

	if err := pr.ValidateCompensationPackageID(); err != nil {
		return err
	}

	if err := pr.ValidateCurrency(); err != nil {
		return err
	}

	if err := pr.ValidatePositiveMoneyValues(); err != nil {
		return err
	}

	if err := pr.ValidateRoundingAmount(); err != nil {
		return err
	}

	if err := pr.ValidateMathConsistency(); err != nil {
		return err
	}

	return nil
}

// ============================================================================
// PayrollResult Validation Methods
// ============================================================================

// ValidateID ensures the payroll result has a valid UUID.
func (pr *PayrollResult) ValidateID() error {
	if pr.ID == uuid.Nil {
		return ErrPayrollResultIDRequired
	}
	return nil
}

// ValidateEmployeeID ensures the payroll result is linked to an employee.
func (pr *PayrollResult) ValidateEmployeeID() error {
	if pr.EmployeeID == uuid.Nil {
		return ErrPayrollResultEmployeeIDRequired
	}
	return nil
}

// ValidatePayrollPeriodID ensures the payroll result is linked to a period.
func (pr *PayrollResult) ValidatePayrollPeriodID() error {
	if pr.PayrollPeriodID == uuid.Nil {
		return ErrPayrollPeriodIDRequired
	}
	return nil
}

// ValidateCompensationPackageID ensures the result references a compensation package.
func (pr *PayrollResult) ValidateCompensationPackageID() error {
	if pr.CompensationPackageID == uuid.Nil {
		return ErrPayrollResultCompensationPackageIDRequired
	}
	return nil
}

// ValidateCurrency ensures the currency is supported.
func (pr *PayrollResult) ValidateCurrency() error {
	if !pr.Currency.IsSupported() {
		return fmt.Errorf(
			"%w: must be one of %v",
			money.ErrCurrencyNotSupported,
			money.SupportedCurrenciesStr,
		)
	}
	return nil
}

// ValidatePositiveMoneyValues ensures all monetary amounts are non-negative.
// The only exception is RoundingAmount, which can be negative.
func (pr *PayrollResult) ValidatePositiveMoneyValues() error {
	if pr.BaseSalary.IsNegative() ||
		pr.SeniorityBonus.IsNegative() ||
		pr.GrossSalary.IsNegative() ||
		pr.TotalOtherBonus.IsNegative() ||
		pr.GrossSalaryGrandTotal.IsNegative() ||
		pr.TotalExemptions.IsNegative() ||
		pr.TaxableGrossSalary.IsNegative() ||
		pr.SocialAllowanceEmployeeContrib.IsNegative() ||
		pr.SocialAllowanceEmployerContrib.IsNegative() ||
		pr.JobLossCompensationEmployeeContrib.IsNegative() ||
		pr.JobLossCompensationEmployerContrib.IsNegative() ||
		pr.TrainingTaxEmployerContrib.IsNegative() ||
		pr.FamilyBenefitsEmployerContrib.IsNegative() ||
		pr.TotalCNSSEmployeeContrib.IsNegative() ||
		pr.TotalCNSSEmployerContrib.IsNegative() ||
		pr.AMOEmployeeContrib.IsNegative() ||
		pr.AMOEmployerContrib.IsNegative() ||
		pr.TaxableNetSalary.IsNegative() ||
		pr.IncomeTax.IsNegative() ||
		pr.NetToPay.IsNegative() {
		return fmt.Errorf(
			"%w: all money values except .RoundingAmount must be >= 0",
			ErrInvalidPayrollResultMoneyValue,
		)
	}
	return nil
}

// ValidateRoundingAmount ensures the rounding adjustment is within bounds.
// Moroccan payroll practice rounds to the nearest dirham (1 MAD = 100 cents).
// Therefore, rounding should never exceed ±1 MAD (±100 cents).
func (pr *PayrollResult) ValidateRoundingAmount() error {
	if pr.RoundingAmount.LessThan(MinPayrollResultRoundingAmount) ||
		pr.RoundingAmount.GreaterThan(MaxPayrollResultRoundingAmount) {
		return fmt.Errorf(
			"%w: must be between %v and %v inclusive",
			ErrInvalidPayrollResultRoundingAmount,
			MinPayrollResultRoundingAmount,
			MaxPayrollResultRoundingAmount,
		)
	}
	return nil
}

// ValidateMathConsistency ensures all calculated fields are mathematically consistent.
// This validates the internal consistency of the payroll calculation, not the
// calculation rules themselves (which are the responsibility of the payroll calculator).
//
// Validated formulas:
//   - GrossSalary = BaseSalary + SeniorityBonus
//   - GrossSalaryGrandTotal = GrossSalary + TotalOtherBonus
//   - TotalCNSSEmployeeContrib = SocialAllowance + JobLossCompensation (NO AMO)
//   - TotalCNSSEmployerContrib = SocialAllowance + JobLoss + Training + Family (NO AMO)
//   - TaxableGrossSalary = GrossSalaryGrandTotal - TotalExemptions
//   - TaxableNetSalary = TaxableGrossSalary - TotalCNSSEmployee - AMOEmployee
//   - NetToPay = TaxableNetSalary - IncomeTax + RoundingAmount
func (pr *PayrollResult) ValidateMathConsistency() error {
	// GrossSalary = BaseSalary + SeniorityBonus
	grossSalaryExpected, err := pr.BaseSalary.Add(pr.SeniorityBonus)
	if err != nil {
		return fmt.Errorf(
			"%w: could not calculate expected gross salary",
			err,
		)
	}
	if !pr.GrossSalary.Equals(grossSalaryExpected) {
		return fmt.Errorf(
			"%w: .GrossSalary must equal .BaseSalary + .SeniorityBonus",
			ErrInconsistentPayrollResultCalculation,
		)
	}

	// GrossSalaryGrandTotal = GrossSalary + TotalOtherBonus
	grossSalaryGrandTotalExpected, err := pr.GrossSalary.Add(pr.TotalOtherBonus)
	if err != nil {
		return fmt.Errorf(
			"%w: could not calculate expected gross salary grand total",
			err,
		)
	}
	if !pr.GrossSalaryGrandTotal.Equals(grossSalaryGrandTotalExpected) {
		return fmt.Errorf(
			"%w: .GrossSalaryGrandTotal must equal .GrossSalary + .TotalOtherBonus",
			ErrInconsistentPayrollResultCalculation,
		)
	}

	// TotalCNSSEmployeeContrib = SocialAllowance + JobLossCompensation (NO AMO)
	totalCNSSEmployeeContribExpected, err := pr.SocialAllowanceEmployeeContrib.Add(pr.JobLossCompensationEmployeeContrib)
	if err != nil {
		return fmt.Errorf(
			"%w: could not calculate expected total CNSS employee contribution",
			err,
		)
	}
	if !pr.TotalCNSSEmployeeContrib.Equals(totalCNSSEmployeeContribExpected) {
		return fmt.Errorf(
			"%w: .TotalCNSSEmployeeContrib must equal .SocialAllowanceEmployeeContrib + .JobLossCompensationEmployeeContrib (AMO excluded)",
			ErrInconsistentPayrollResultCalculation,
		)
	}

	// TotalCNSSEmployerContrib = SocialAllowance + JobLoss + Training + Family (NO AMO)
	totalCNSSEmployerContribExpected, err := pr.SocialAllowanceEmployerContrib.Add(pr.JobLossCompensationEmployerContrib)
	if err != nil {
		return fmt.Errorf(
			"%w: could not add .SocialAllowanceEmployerContrib and .JobLossCompensationEmployerContrib",
			err,
		)
	}
	totalCNSSEmployerContribExpected, err = totalCNSSEmployerContribExpected.Add(pr.FamilyBenefitsEmployerContrib)
	if err != nil {
		return fmt.Errorf(
			"%w: could not add .FamilyBenefitsEmployerContrib to expected total CNSS employer contribution",
			err,
		)
	}
	totalCNSSEmployerContribExpected, err = totalCNSSEmployerContribExpected.Add(pr.TrainingTaxEmployerContrib)
	if err != nil {
		return fmt.Errorf(
			"%w: could not add .TrainingTaxEmployerContrib to expected total CNSS employer contribution",
			err,
		)
	}
	if !pr.TotalCNSSEmployerContrib.Equals(totalCNSSEmployerContribExpected) {
		return fmt.Errorf(
			"%w: .TotalCNSSEmployerContrib must equal .SocialAllowanceEmployerContrib + .JobLossCompensationEmployerContrib + .FamilyBenefitsEmployerContrib + .TrainingTaxEmployerContrib (AMO excluded)",
			ErrInconsistentPayrollResultCalculation,
		)
	}

	// TaxableGrossSalary = GrossSalaryGrandTotal - TotalExemptions
	taxableGrossSalaryExpected, err := pr.GrossSalaryGrandTotal.Subtract(pr.TotalExemptions)
	if err != nil {
		return fmt.Errorf(
			"%w: could not calculate expected taxable gross salary",
			err,
		)
	}
	if !pr.TaxableGrossSalary.Equals(taxableGrossSalaryExpected) {
		return fmt.Errorf(
			"%w: .TaxableGrossSalary must equal .GrossSalaryGrandTotal - .TotalExemptions",
			ErrInconsistentPayrollResultCalculation,
		)
	}

	// TaxableNetSalary = TaxableGrossSalary - TotalCNSSEmployee - AMOEmployee
	taxableNetSalaryExpected, err := pr.TaxableGrossSalary.Subtract(pr.TotalCNSSEmployeeContrib)
	if err != nil {
		return fmt.Errorf(
			"%w: could not subtract .TotalCNSSEmployeeContrib from .TaxableGrossSalary",
			err,
		)
	}
	taxableNetSalaryExpected, err = taxableNetSalaryExpected.Subtract(pr.AMOEmployeeContrib)
	if err != nil {
		return fmt.Errorf(
			"%w: could not subtract .AMOEmployeeContrib from expected taxable net salary",
			err,
		)
	}
	if !pr.TaxableNetSalary.Equals(taxableNetSalaryExpected) {
		return fmt.Errorf(
			"%w: .TaxableNetSalary must equal .TaxableGrossSalary - .TotalCNSSEmployeeContrib - .AMOEmployeeContrib",
			ErrInconsistentPayrollResultCalculation,
		)
	}

	// NetToPay = TaxableNetSalary - IncomeTax + RoundingAmount
	netToPayExpected, err := pr.TaxableNetSalary.Subtract(pr.IncomeTax)
	if err != nil {
		return fmt.Errorf(
			"%w: could not subtract .IncomeTax from .TaxableNetSalary",
			err,
		)
	}

	netToPayExpected, err = netToPayExpected.Add(pr.RoundingAmount)
	if err != nil {
		return fmt.Errorf(
			"%w: could not add .RoundingAmount to expected net to pay",
			err,
		)
	}

	if !pr.NetToPay.Equals(netToPayExpected) {
		return fmt.Errorf(
			"%w: .NetToPay must equal .TaxableNetSalary - .IncomeTax + .RoundingAmount",
			ErrInconsistentPayrollResultCalculation,
		)
	}

	return nil
}

// ============================================================================
// PayrollResult Helper Methods
// ============================================================================

// TotalDueToCNSS returns the total amount due to CNSS (including AMO).
// In practice, AMO is collected by CNSS even though it's conceptually separate.
// This helper provides the total amount that needs to be paid to CNSS.
func (pr *PayrollResult) TotalDueToCNSS() (money.Money, error) {
	total, err := pr.TotalCNSSEmployeeContrib.Add(pr.TotalCNSSEmployerContrib)
	if err != nil {
		return money.Money{}, fmt.Errorf(
			"%w: could not add .TotalCNSSEmployeeContrib and .TotalCNSSEmployerContrib",
			err,
		)
	}

	total, err = total.Add(pr.AMOEmployeeContrib)
	if err != nil {
		return money.Money{}, fmt.Errorf(
			"%w: could not add .AMOEmployeeContrib to total",
			err,
		)
	}

	total, err = total.Add(pr.AMOEmployerContrib)
	if err != nil {
		return money.Money{}, fmt.Errorf(
			"%w: could not add .AMOEmployerContrib to total",
			err,
		)
	}

	return total, nil
}

// TotalEmployeeDeductions returns the total amount deducted from employee salary.
// This includes both CNSS and AMO contributions.
func (pr *PayrollResult) TotalEmployeeDeductions() (money.Money, error) {
	total, err := pr.TotalCNSSEmployeeContrib.Add(pr.AMOEmployeeContrib)
	if err != nil {
		return money.Money{}, fmt.Errorf(
			"%w: could not add .TotalCNSSEmployeeContrib and .AMOEmployeeContrib",
			err,
		)
	}
	return total, nil
}

// ============================================================================
// PayrollResult Constants
// ============================================================================

var (
	// MinPayrollResultRoundingAmount is the minimum rounding adjustment (-1 MAD).
	MinPayrollResultRoundingAmount = money.FromCents(-100)

	// MaxPayrollResultRoundingAmount is the maximum rounding adjustment (+1 MAD).
	MaxPayrollResultRoundingAmount = money.FromCents(100)
)

// ============================================================================
// PayrollResult Errors
// ============================================================================

var (
	ErrPayrollResultIDRequired                    = errors.New("domain: payroll: payroll result id (uuid) is required")
	ErrPayrollResultEmployeeIDRequired            = errors.New("domain: payroll: payroll result employee id (uuid) is required")
	ErrPayrollResultCompensationPackageIDRequired = errors.New("domain: payroll: payroll result compensation package id (uuid) is required")
	ErrInconsistentPayrollResultCalculation       = errors.New("domain: payroll: inconsistent payroll result calculation")
	ErrInvalidPayrollResultRoundingAmount         = errors.New("domain: payroll: invalid payroll result rounding amount")
	ErrInvalidPayrollResultMoneyValue             = errors.New("domain: payroll: invalid payroll result money value")
)
