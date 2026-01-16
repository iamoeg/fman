package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/iamoeg/bootdev-capstone/pkg/util"
)

// ============================================================================
// Organization Entity
// ============================================================================

// Organization represents a company or business entity in the system.
// This is the root entity for multi-tenant support - each organization
// has its own employees, payroll periods, and data.
//
// Business Rules:
//   - Name is required
//   - Legal form must be supported
//   - Moroccan business identifiers (ICE, IF, RC, CNSS) should be unique
type Organization struct {
	// Identity
	ID uuid.UUID

	// Basic Information
	Name     string
	Address  string
	Activity string // Business activity description

	// Legal Structure
	LegalForm OrgLegalFormEnum

	// Moroccan Business Identifiers
	ICENum  string // Identifiant Commun de l'Entreprise (Common Enterprise Identifier)
	IFNum   string // Identifiant Fiscal (Tax ID)
	RCNum   string // Registre de Commerce (Commerce Registry Number)
	CNSSNum string // Caisse Nationale de Sécurité Sociale (Social Security Number)

	// Banking
	BankRIB string // Relevé d'Identité Bancaire (Bank Account Number)

	// Metadata
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time // Soft delete timestamp
}

// Validate performs comprehensive validation of the organization.
func (o *Organization) Validate() error {
	if err := o.ValidateID(); err != nil {
		return err
	}

	if err := o.ValidateName(); err != nil {
		return err
	}

	if err := o.ValidateLegalForm(); err != nil {
		return err
	}

	return nil
}

// ============================================================================
// Organization Validation Methods
// ============================================================================

// ValidateID ensures the organization has a valid UUID.
func (o *Organization) ValidateID() error {
	if o.ID == uuid.Nil {
		return ErrOrgIDRequired
	}
	return nil
}

// ValidateName ensures the organization has a non-empty name.
func (o *Organization) ValidateName() error {
	nameTrimmed := strings.TrimSpace(o.Name)
	if nameTrimmed == "" {
		return ErrOrgNameRequired
	}
	return nil
}

// ValidateLegalForm ensures the legal form is one of the supported values.
func (o *Organization) ValidateLegalForm() error {
	if !o.LegalForm.IsSupported() {
		return fmt.Errorf(
			"%w: must be one of %v",
			ErrOrgLegalFormNotSupported,
			SupportedOrgLegalFormsStr,
		)
	}
	return nil
}

// ============================================================================
// Legal Form Enum
// ============================================================================

// OrgLegalFormEnum represents the legal structure of the organization.
// Currently only SARL is supported, but can be extended to support
// other Moroccan legal forms (SA, SAS, etc.).
type OrgLegalFormEnum string

const (
	// LegalFormSARL represents "Société à Responsabilité Limitée"
	// (Limited Liability Company) - the most common legal form in Morocco.
	LegalFormSARL OrgLegalFormEnum = "SARL"
)

var supportedOrgLegalForms = map[OrgLegalFormEnum]struct{}{
	LegalFormSARL: {},
}

// SupportedOrgLegalFormsStr is a comma-separated list of supported legal forms.
var SupportedOrgLegalFormsStr = util.EnumMapToString(supportedOrgLegalForms)

// IsSupported returns true if the legal form is one of the supported values.
func (lf OrgLegalFormEnum) IsSupported() bool {
	_, ok := supportedOrgLegalForms[lf]
	return ok
}

// ============================================================================
// Organization Errors
// ============================================================================

var (
	ErrOrgIDRequired            = errors.New("domain: organization: id (uuid) is required")
	ErrOrgNameRequired          = errors.New("domain: organization: name is required")
	ErrOrgLegalFormNotSupported = errors.New("domain: organization: legal form not supported")
)
