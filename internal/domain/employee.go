package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/iamoeg/bootdev-capstone/pkg/money"
	"github.com/iamoeg/bootdev-capstone/pkg/util"
)

// ============================================================================
// Employee Entity
// ============================================================================

// Employee represents an employee within an organization.
// It contains demographic information, employment details, and references
// to the organization and compensation package.
//
// Business Rules:
//   - Must belong to an organization (OrgID)
//   - Must have a unique serial number within the organization
//   - Must be between MinWorkLegalAge and MaxWorkLegalAge years old
//   - Hire date must be within the last MaxHireYearsInPast years
//   - Must have a valid compensation package
type Employee struct {
	// Identity
	ID    uuid.UUID
	OrgID uuid.UUID

	// Employee Number
	SerialNum int // Unique within organization, starts at 1

	// Personal Information
	FullName     string
	DisplayName  string // Optional preferred name
	Address      string
	EmailAddress string
	PhoneNumber  string
	BirthDate    time.Time
	Gender       GenderEnum
	CINNum       string // Carte d'Identité Nationale (Moroccan National ID)
	CNSSNum      string // Optional - may not have one for first job

	// Employment Information
	HireDate              time.Time
	Position              string
	CompensationPackageID uuid.UUID

	// Tax-Relevant Information
	MaritalStatus MaritalStatusEnum
	NumDependents int // Number of dependents for tax calculations
	NumKids       int // Number of children for tax calculations

	// Banking
	BankRIB string // RIB (Relevé d'Identité Bancaire) - bank account number

	// Metadata
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time // Soft delete timestamp
}

// Validate performs comprehensive validation of the employee.
// It checks all business rules including age requirements, hire date constraints,
// and required field presence.
func (e *Employee) Validate() error {
	if err := e.ValidateID(); err != nil {
		return err
	}

	if err := e.ValidateOrgID(); err != nil {
		return err
	}

	if err := e.ValidateSerialNum(); err != nil {
		return err
	}

	if err := e.ValidateFullName(); err != nil {
		return err
	}

	if err := e.ValidateBirthDate(); err != nil {
		return err
	}

	if err := e.ValidateGender(); err != nil {
		return err
	}

	if err := e.ValidateMaritalStatus(); err != nil {
		return err
	}

	if err := e.ValidateNumDependents(); err != nil {
		return err
	}

	if err := e.ValidateNumKids(); err != nil {
		return err
	}

	if err := e.ValidateCINNum(); err != nil {
		return err
	}

	if err := e.ValidateHireDate(); err != nil {
		return err
	}

	if err := e.ValidateMinHireDate(); err != nil {
		return err
	}

	if err := e.ValidatePosition(); err != nil {
		return err
	}

	if err := e.ValidateCompensationPackageID(); err != nil {
		return err
	}

	return nil
}

// ============================================================================
// Employee Validation Methods
// ============================================================================

// ValidateID ensures the employee has a valid UUID.
func (e *Employee) ValidateID() error {
	if e.ID == uuid.Nil {
		return ErrEmployeeIDRequired
	}
	return nil
}

// ValidateOrgID ensures the employee belongs to an organization.
func (e *Employee) ValidateOrgID() error {
	if e.OrgID == uuid.Nil {
		return ErrEmployeeOrgIDRequired
	}
	return nil
}

// ValidateSerialNum ensures the employee number is valid.
// Employee serial numbers are unique within an organization.
func (e *Employee) ValidateSerialNum() error {
	if e.SerialNum < MinEmployeeSerialNum {
		return fmt.Errorf(
			"%w: must be >= 1",
			ErrInvalidEmployeeSerialNum,
		)
	}
	if e.SerialNum > MaxEmployeeSerialNum {
		return fmt.Errorf(
			"%w: serial number %d exceeds maximum safe value %d",
			ErrInvalidEmployeeSerialNum,
			e.SerialNum,
			MaxEmployeeSerialNum,
		)
	}
	return nil
}

// ValidateFullName ensures the employee has a non-empty name.
func (e *Employee) ValidateFullName() error {
	fullNameTrimmed := strings.TrimSpace(e.FullName)
	if fullNameTrimmed == "" {
		return ErrEmployeeFullNameRequired
	}
	return nil
}

// ValidateCINNum ensures the employee has a CIN number.
// CIN (Carte d'Identité Nationale) is the Moroccan National ID.
func (e *Employee) ValidateCINNum() error {
	CINNumTrimmed := strings.TrimSpace(e.CINNum)
	if CINNumTrimmed == "" {
		return ErrEmployeeCINNumRequired
	}
	return nil
}

// ValidateBirthDate ensures the employee's age is within legal working age.
// Employees must be between MinWorkLegalAge (16) and MaxWorkLegalAge (80).
func (e *Employee) ValidateBirthDate() error {
	now := time.Now().UTC()
	minBirthDate := now.AddDate(-MaxWorkLegalAge, 0, 0)
	maxBirthDate := now.AddDate(-MinWorkLegalAge, 0, 0)
	if e.BirthDate.Before(minBirthDate) || e.BirthDate.After(maxBirthDate) {
		return fmt.Errorf(
			"%w: employee's age must be between %v and %v years",
			ErrInvalidEmployeeBirthDate,
			MinWorkLegalAge,
			MaxWorkLegalAge,
		)
	}
	return nil
}

// ValidateGender ensures the gender is one of the supported values.
func (e *Employee) ValidateGender() error {
	if !e.Gender.IsSupported() {
		return fmt.Errorf(
			"%w: must be one of %v",
			ErrGenderNotSupported,
			SupportedGendersStr,
		)
	}
	return nil
}

// ValidateMaritalStatus ensures the marital status is one of the supported values.
func (e *Employee) ValidateMaritalStatus() error {
	if !e.MaritalStatus.IsSupported() {
		return fmt.Errorf(
			"%w: must be one of %v",
			ErrMaritalStatusNotSupported,
			SupportedMaritalStatusesStr,
		)
	}
	return nil
}

// ValidateNumDependents ensures the number of dependents is non-negative.
func (e *Employee) ValidateNumDependents() error {
	if e.NumDependents < MinEmployeeNumDependents {
		return fmt.Errorf(
			"%w: must be >= %v",
			ErrInvalidEmployeeNumDependents,
			MinEmployeeNumDependents,
		)
	}
	return nil
}

// ValidateNumKids ensures the number of children is non-negative.
func (e *Employee) ValidateNumKids() error {
	if e.NumKids < MinEmployeeNumKids {
		return fmt.Errorf(
			"%w: must be >= %v",
			ErrInvalidEmployeeNumKids,
			MinEmployeeNumKids,
		)
	}
	return nil
}

// ValidateHireDate ensures the hire date is within supported range.
// Hire dates must be:
//   - Within the last MaxHireYearsInPast years (payroll calculation limitation)
//   - Not in the future
func (e *Employee) ValidateHireDate() error {
	now := time.Now()
	currentYear := now.Year()
	minAllowedYear := currentYear - MaxHireYearsInPast

	if e.HireDate.Year() < minAllowedYear {
		return fmt.Errorf(
			"%w: payroll calculations only supported for %d and later",
			ErrInvalidEmployeeHireDate,
			minAllowedYear,
		)
	}

	if e.HireDate.After(now) {
		return fmt.Errorf(
			"%w: cannot be in the future",
			ErrInvalidEmployeeHireDate,
		)
	}

	return nil
}

// ValidateMinHireDate ensures the employee was at least MinWorkLegalAge when hired.
// This is a cross-field validation between BirthDate and HireDate.
func (e *Employee) ValidateMinHireDate() error {
	minHireDate := e.BirthDate.AddDate(MinWorkLegalAge, 0, 0)
	if e.HireDate.Before(minHireDate) {
		return fmt.Errorf(
			"%w: hire date must be at least %v years after birth date",
			ErrInvalidEmployeeHireDate,
			MinWorkLegalAge,
		)
	}
	return nil
}

// ValidatePosition ensures the employee has a position/title.
func (e *Employee) ValidatePosition() error {
	positionTrimmed := strings.TrimSpace(e.Position)
	if positionTrimmed == "" {
		return ErrEmployeePositionRequired
	}
	return nil
}

// ValidateCompensationPackageID ensures the employee has a compensation package.
func (e *Employee) ValidateCompensationPackageID() error {
	if e.CompensationPackageID == uuid.Nil {
		return ErrEmployeeCompensationPackageIDRequired
	}
	return nil
}

// ============================================================================
// Gender Enum
// ============================================================================

// GenderEnum represents the employee's gender.
// Limited to MALE/FEMALE as these are the only options recognized
// in Moroccan official documentation.
type GenderEnum string

const (
	GenderMale   GenderEnum = "MALE"
	GenderFemale GenderEnum = "FEMALE"
)

var supportedGenders = map[GenderEnum]struct{}{
	GenderMale:   {},
	GenderFemale: {},
}

// SupportedGendersStr is a comma-separated list of supported gender values.
var SupportedGendersStr = util.EnumMapToString(supportedGenders)

// IsSupported returns true if the gender is one of the supported values.
func (g GenderEnum) IsSupported() bool {
	_, ok := supportedGenders[g]
	return ok
}

// ============================================================================
// Marital Status Enum
// ============================================================================

// MaritalStatusEnum represents the employee's marital status.
// Used for tax calculations in Moroccan payroll.
type MaritalStatusEnum string

const (
	MaritalStatusSingle    MaritalStatusEnum = "SINGLE"
	MaritalStatusMarried   MaritalStatusEnum = "MARRIED"
	MaritalStatusSeparated MaritalStatusEnum = "SEPARATED"
	MaritalStatusDivorced  MaritalStatusEnum = "DIVORCED"
	MaritalStatusWidowed   MaritalStatusEnum = "WIDOWED"
)

var supportedMaritalStatuses = map[MaritalStatusEnum]struct{}{
	MaritalStatusSingle:    {},
	MaritalStatusMarried:   {},
	MaritalStatusSeparated: {},
	MaritalStatusDivorced:  {},
	MaritalStatusWidowed:   {},
}

// SupportedMaritalStatusesStr is a comma-separated list of supported marital status values.
var SupportedMaritalStatusesStr = util.EnumMapToString(supportedMaritalStatuses)

// IsSupported returns true if the marital status is one of the supported values.
func (ms MaritalStatusEnum) IsSupported() bool {
	_, ok := supportedMaritalStatuses[ms]
	return ok
}

// ============================================================================
// Employee Constants
// ============================================================================

const (
	// MinEmployeeSerialNum is the minimum valid employee serial number.
	MinEmployeeSerialNum = 1

	// MaxEmployeeSerialNum is the maximum valid employee serial number (max safe int32 value).
	MaxEmployeeSerialNum = 2_147_483_647

	// MinEmployeeNumDependents is the minimum number of dependents (0).
	MinEmployeeNumDependents = 0

	// MinEmployeeNumKids is the minimum number of children (0).
	MinEmployeeNumKids = 0

	// MinWorkLegalAge is the minimum legal working age in Morocco (16 years).
	MinWorkLegalAge = 16

	// MaxWorkLegalAge is the maximum reasonable working age (80 years).
	MaxWorkLegalAge = 80

	// MaxHireYearsInPast limits how far back hire dates can be.
	// This is due to payroll calculation rules changing over time.
	// Currently set to 1, meaning only current year and previous year are supported.
	MaxHireYearsInPast = 1
)

// ============================================================================
// Employee Errors
// ============================================================================

var (
	ErrEmployeeIDRequired                    = errors.New("domain: employee: id (uuid) is required")
	ErrEmployeeOrgIDRequired                 = errors.New("domain: employee: org id (uuid) required")
	ErrEmployeeFullNameRequired              = errors.New("domain: employee: full name is required")
	ErrEmployeeCINNumRequired                = errors.New("domain: employee: CIN number is required")
	ErrEmployeePositionRequired              = errors.New("domain: employee: position is required")
	ErrEmployeeCompensationPackageIDRequired = errors.New("domain: employee: compensation package id (uuid) required")
	ErrInvalidEmployeeSerialNum              = errors.New("domain: employee: invalid serial number")
	ErrInvalidEmployeeNumDependents          = errors.New("domain: employee: invalid number of dependents")
	ErrInvalidEmployeeNumKids                = errors.New("domain: employee: invalid number of kids")
	ErrInvalidEmployeeBirthDate              = errors.New("domain: employee: invalid birth date")
	ErrInvalidEmployeeHireDate               = errors.New("domain: employee: invalid hire date")
	ErrGenderNotSupported                    = errors.New("domain: employee: gender not supported")
	ErrMaritalStatusNotSupported             = errors.New("domain: employee: marital status not supported")
)

// ============================================================================
// EmployeeCompensationPackage Entity
// ============================================================================

// EmployeeCompensationPackage represents an immutable compensation record.
// Once created, compensation packages should not be modified - instead,
// create a new package and update the employee's reference.
//
// This design preserves historical accuracy: when looking at old payroll
// results, we can see exactly what compensation package was used.
//
// Business Rules:
//   - Base salary must be >= SMIG (Moroccan minimum wage)
//   - Currency must be supported (currently only MAD)
type EmployeeCompensationPackage struct {
	ID         uuid.UUID
	Currency   money.Currency
	BaseSalary money.Money

	// Metadata
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time // Soft delete - historical packages should never be hard-deleted
}

// Validate performs comprehensive validation of the compensation package.
func (cp *EmployeeCompensationPackage) Validate() error {
	if err := cp.ValidateID(); err != nil {
		return err
	}

	if err := cp.ValidateBaseSalary(); err != nil {
		return err
	}

	if err := cp.ValidateCurrency(); err != nil {
		return err
	}

	return nil
}

// ValidateID ensures the compensation package has a valid UUID.
func (cp *EmployeeCompensationPackage) ValidateID() error {
	if cp.ID == uuid.Nil {
		return ErrEmployeeCompensationPackageIDRequired
	}
	return nil
}

// ValidateBaseSalary ensures the base salary meets minimum wage requirements.
// In Morocco, the SMIG (Salaire Minimum Interprofessionnel Garanti) is the
// legal minimum wage.
func (cp *EmployeeCompensationPackage) ValidateBaseSalary() error {
	if cp.BaseSalary.LessThan(MinWageSMIG) {
		return fmt.Errorf(
			"%w: must be >= %v",
			ErrInvalidEmployeeCompensationPackageBaseSalary,
			MinWageSMIG,
		)
	}
	return nil
}

// ValidateCurrency ensures the currency is supported.
func (cp *EmployeeCompensationPackage) ValidateCurrency() error {
	if !cp.Currency.IsSupported() {
		return fmt.Errorf(
			"%w: must be one of %v",
			money.ErrCurrencyNotSupported,
			money.SupportedCurrenciesStr,
		)
	}
	return nil
}

// ============================================================================
// Compensation Package Constants
// ============================================================================

const (
	// SMIG2026Cents is the Moroccan minimum wage (SMIG) for 2026 in cents.
	// SMIG = Salaire Minimum Interprofessionnel Garanti
	// Value: 3,422.00 MAD = 342,200 cents
	SMIG2026Cents int64 = 3422 * 100
)

// MinWageSMIG is the minimum wage in Morocco (SMIG 2026).
var MinWageSMIG = money.FromCents(SMIG2026Cents)

// ============================================================================
// Compensation Package Errors
// ============================================================================

var (
	ErrInvalidEmployeeCompensationPackageBaseSalary = errors.New("domain: employee: compensation package: invalid base salary")
)
