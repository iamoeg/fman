package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iamoeg/fman/internal/domain"
)

// ============================================================================
// Organization Validation Tests
// ============================================================================

func TestOrganization_Validate(t *testing.T) {
	t.Parallel()

	// Valid base organization for tests
	validOrg := func() *domain.Organization {
		return &domain.Organization{
			ID:        uuid.New(),
			Name:      "Test Company SARL",
			Address:   "123 Rue Mohammed V, Casablanca",
			Activity:  "Software Development",
			LegalForm: domain.LegalFormSARL,
			ICENum:    "001234567890123",
			IFNum:     "12345678",
			RCNum:     "123456",
			CNSSNum:   "1234567",
			BankRIB:   "012345678901234567890123",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
	}

	tests := []struct {
		name    string
		org     *domain.Organization
		wantErr error
	}{
		// ====================================================================
		// Valid Cases
		// ====================================================================
		{
			name:    "valid organization with all fields",
			org:     validOrg(),
			wantErr: nil,
		},
		{
			name: "valid organization with minimal required fields",
			org: &domain.Organization{
				ID:        uuid.New(),
				Name:      "Minimal SARL",
				LegalForm: domain.LegalFormSARL,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			wantErr: nil,
		},
		{
			name: "valid organization with only name and legal form",
			org: &domain.Organization{
				ID:        uuid.New(),
				Name:      "Simple Company",
				LegalForm: domain.LegalFormSARL,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			wantErr: nil,
		},

		// ====================================================================
		// ID Validation Errors
		// ====================================================================
		{
			name: "missing ID (uuid.Nil)",
			org: func() *domain.Organization {
				org := validOrg()
				org.ID = uuid.Nil
				return org
			}(),
			wantErr: domain.ErrOrgIDRequired,
		},
		{
			name: "zero UUID",
			org: func() *domain.Organization {
				org := validOrg()
				org.ID = uuid.UUID{}
				return org
			}(),
			wantErr: domain.ErrOrgIDRequired,
		},

		// ====================================================================
		// Name Validation Errors
		// ====================================================================
		{
			name: "empty name",
			org: func() *domain.Organization {
				org := validOrg()
				org.Name = ""
				return org
			}(),
			wantErr: domain.ErrOrgNameRequired,
		},
		{
			name: "name with only whitespace",
			org: func() *domain.Organization {
				org := validOrg()
				org.Name = "   "
				return org
			}(),
			wantErr: domain.ErrOrgNameRequired,
		},
		{
			name: "name with tabs and spaces",
			org: func() *domain.Organization {
				org := validOrg()
				org.Name = "\t\n  \t"
				return org
			}(),
			wantErr: domain.ErrOrgNameRequired,
		},
		{
			name: "name with leading/trailing spaces (should pass)",
			org: func() *domain.Organization {
				org := validOrg()
				org.Name = "  Valid Company  "
				return org
			}(),
			wantErr: nil, // TrimSpace should allow this
		},

		// ====================================================================
		// Legal Form Validation Errors
		// ====================================================================
		{
			name: "unsupported legal form - empty",
			org: func() *domain.Organization {
				org := validOrg()
				org.LegalForm = ""
				return org
			}(),
			wantErr: domain.ErrOrgLegalFormNotSupported,
		},
		{
			name: "unsupported legal form - SA",
			org: func() *domain.Organization {
				org := validOrg()
				org.LegalForm = "SA" // Not yet supported
				return org
			}(),
			wantErr: domain.ErrOrgLegalFormNotSupported,
		},
		{
			name: "unsupported legal form - SAS",
			org: func() *domain.Organization {
				org := validOrg()
				org.LegalForm = "SAS" // Not yet supported
				return org
			}(),
			wantErr: domain.ErrOrgLegalFormNotSupported,
		},
		{
			name: "unsupported legal form - lowercase",
			org: func() *domain.Organization {
				org := validOrg()
				org.LegalForm = "sarl" // Wrong case
				return org
			}(),
			wantErr: domain.ErrOrgLegalFormNotSupported,
		},
		{
			name: "unsupported legal form - random value",
			org: func() *domain.Organization {
				org := validOrg()
				org.LegalForm = "INVALID"
				return org
			}(),
			wantErr: domain.ErrOrgLegalFormNotSupported,
		},

		// ====================================================================
		// Edge Cases - Optional Fields
		// ====================================================================
		{
			name: "optional fields can be empty",
			org: &domain.Organization{
				ID:        uuid.New(),
				Name:      "Company Without Details",
				LegalForm: domain.LegalFormSARL,
				// All optional fields empty
				Address:   "",
				Activity:  "",
				ICENum:    "",
				IFNum:     "",
				RCNum:     "",
				CNSSNum:   "",
				BankRIB:   "",
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.org.Validate()

			// Check if error matches expectation
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() error = %v, wantErr nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("Validate() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

// ============================================================================
// Individual Validation Method Tests
// ============================================================================

func TestOrganization_ValidateID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr error
	}{
		{
			name:    "valid UUID",
			id:      uuid.New(),
			wantErr: nil,
		},
		{
			name:    "nil UUID",
			id:      uuid.Nil,
			wantErr: domain.ErrOrgIDRequired,
		},
		{
			name:    "zero UUID",
			id:      uuid.UUID{},
			wantErr: domain.ErrOrgIDRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			org := &domain.Organization{ID: tt.id}
			err := org.ValidateID()

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateID() error = %v, wantErr nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateID() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateID() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestOrganization_ValidateName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		orgName string
		wantErr error
	}{
		// Valid names
		{
			name:    "normal company name",
			orgName: "Test Company SARL",
			wantErr: nil,
		},
		{
			name:    "single character name",
			orgName: "A",
			wantErr: nil,
		},
		{
			name:    "name with numbers",
			orgName: "Company 123",
			wantErr: nil,
		},
		{
			name:    "name with special characters",
			orgName: "Company & Associates, Ltd.",
			wantErr: nil,
		},
		{
			name:    "very long name",
			orgName: "This is a very long company name that might be unusual but should still be valid",
			wantErr: nil,
		},
		{
			name:    "name with leading/trailing spaces",
			orgName: "  Valid Company  ",
			wantErr: nil, // TrimSpace should handle this
		},
		{
			name:    "name with Arabic characters",
			orgName: "شركة الاختبار",
			wantErr: nil,
		},
		{
			name:    "name with French accents",
			orgName: "Société Française",
			wantErr: nil,
		},

		// Invalid names
		{
			name:    "empty name",
			orgName: "",
			wantErr: domain.ErrOrgNameRequired,
		},
		{
			name:    "only spaces",
			orgName: "     ",
			wantErr: domain.ErrOrgNameRequired,
		},
		{
			name:    "only tabs",
			orgName: "\t\t\t",
			wantErr: domain.ErrOrgNameRequired,
		},
		{
			name:    "only newlines",
			orgName: "\n\n",
			wantErr: domain.ErrOrgNameRequired,
		},
		{
			name:    "mixed whitespace",
			orgName: " \t \n ",
			wantErr: domain.ErrOrgNameRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			org := &domain.Organization{Name: tt.orgName}
			err := org.ValidateName()

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateName() error = %v, wantErr nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateName() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateName() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestOrganization_ValidateLegalForm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		legalForm domain.OrgLegalFormEnum
		wantErr   error
	}{
		// Valid legal forms
		{
			name:      "SARL (supported)",
			legalForm: domain.LegalFormSARL,
			wantErr:   nil,
		},

		// Invalid legal forms
		{
			name:      "empty string",
			legalForm: "",
			wantErr:   domain.ErrOrgLegalFormNotSupported,
		},
		{
			name:      "SA (not yet supported)",
			legalForm: "SA",
			wantErr:   domain.ErrOrgLegalFormNotSupported,
		},
		{
			name:      "SAS (not yet supported)",
			legalForm: "SAS",
			wantErr:   domain.ErrOrgLegalFormNotSupported,
		},
		{
			name:      "SARL AU (not yet supported)",
			legalForm: "SARL AU",
			wantErr:   domain.ErrOrgLegalFormNotSupported,
		},
		{
			name:      "lowercase sarl",
			legalForm: "sarl",
			wantErr:   domain.ErrOrgLegalFormNotSupported,
		},
		{
			name:      "mixed case SaRl",
			legalForm: "SaRl",
			wantErr:   domain.ErrOrgLegalFormNotSupported,
		},
		{
			name:      "random string",
			legalForm: "INVALID",
			wantErr:   domain.ErrOrgLegalFormNotSupported,
		},
		{
			name:      "numeric value",
			legalForm: "123",
			wantErr:   domain.ErrOrgLegalFormNotSupported,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			org := &domain.Organization{LegalForm: tt.legalForm}
			err := org.ValidateLegalForm()

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateLegalForm() error = %v, wantErr nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateLegalForm() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateLegalForm() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

// ============================================================================
// Enum Tests
// ============================================================================

func TestOrgLegalFormEnum_IsSupported(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		legalForm domain.OrgLegalFormEnum
		want      bool
	}{
		{
			name:      "SARL is supported",
			legalForm: domain.LegalFormSARL,
			want:      true,
		},
		{
			name:      "empty string not supported",
			legalForm: "",
			want:      false,
		},
		{
			name:      "SA not supported",
			legalForm: "SA",
			want:      false,
		},
		{
			name:      "lowercase not supported",
			legalForm: "sarl",
			want:      false,
		},
		{
			name:      "random value not supported",
			legalForm: "INVALID",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.legalForm.IsSupported()
			if got != tt.want {
				t.Errorf("IsSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkOrganization_Validate(b *testing.B) {
	org := &domain.Organization{
		ID:        uuid.New(),
		Name:      "Test Company SARL",
		LegalForm: domain.LegalFormSARL,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = org.Validate()
	}
}

func BenchmarkOrganization_ValidateName(b *testing.B) {
	org := &domain.Organization{
		Name: "Test Company SARL",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = org.ValidateName()
	}
}

func BenchmarkOrgLegalFormEnum_IsSupported(b *testing.B) {
	legalForm := domain.LegalFormSARL

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = legalForm.IsSupported()
	}
}
