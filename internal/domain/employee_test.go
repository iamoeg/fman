package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iamoeg/bootdev-capstone/internal/domain"
	"github.com/iamoeg/bootdev-capstone/pkg/money"
)

// ============================================================================
// Employee Validation Tests
// ============================================================================

func TestEmployee_Validate(t *testing.T) {
	t.Parallel()

	// Valid base employee for tests
	validEmployee := func() *domain.Employee {
		now := time.Now().UTC()
		birthDate := now.AddDate(-30, 0, 0) // 30 years old
		hireDate := now.AddDate(0, -6, 0)   // Hired 6 months ago

		return &domain.Employee{
			ID:                    uuid.New(),
			OrgID:                 uuid.New(),
			SerialNum:             1,
			FullName:              "Ahmed Ben Ali",
			DisplayName:           "Ahmed",
			Address:               "123 Rue Mohammed V, Casablanca",
			PhoneNumber:           "+212-6-12-34-56-78",
			BirthDate:             birthDate,
			Gender:                domain.GenderMale,
			MaritalStatus:         domain.MaritalStatusSingle,
			NumDependents:         0,
			NumChildren:           0,
			CINNum:                "AB123456",
			CNSSNum:               "1234567890",
			HireDate:              hireDate,
			Position:              "Software Developer",
			CompensationPackageID: uuid.New(),
			BankRIB:               "012345678901234567890123",
			CreatedAt:             now,
			UpdatedAt:             now,
		}
	}

	tests := []struct {
		name    string
		emp     *domain.Employee
		wantErr error
	}{
		// ====================================================================
		// Valid Cases
		// ====================================================================
		{
			name:    "valid employee with all fields",
			emp:     validEmployee(),
			wantErr: nil,
		},
		{
			name: "valid employee - minimal fields",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.DisplayName = ""
				emp.Address = ""
				emp.PhoneNumber = ""
				emp.CNSSNum = "" // Optional for first job
				emp.BankRIB = ""
				return emp
			}(),
			wantErr: nil,
		},
		{
			name: "valid employee - female married with dependents",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.Gender = domain.GenderFemale
				emp.MaritalStatus = domain.MaritalStatusMarried
				emp.NumDependents = 2
				emp.NumChildren = 3
				return emp
			}(),
			wantErr: nil,
		},
		{
			name: "valid employee - minimum age (16 years old)",
			emp: func() *domain.Employee {
				emp := validEmployee()
				now := time.Now().UTC()
				// Employee is 17 years old (born 17 years ago)
				emp.BirthDate = now.AddDate(-17, 0, 0)
				// Hired 1 year ago (when they were 16)
				emp.HireDate = now.AddDate(-1, 0, 0)
				return emp
			}(),
			wantErr: nil,
		},
		{
			name: "valid employee - maximum age (79 years old)",
			emp: func() *domain.Employee {
				emp := validEmployee()
				now := time.Now().UTC()
				// Employee is 79 years old (born 79 years ago)
				emp.BirthDate = now.AddDate(-79, 0, 0)
				// Hired last year (within MaxHireYearsInPast)
				emp.HireDate = now.AddDate(-1, 0, 0)
				return emp
			}(),
			wantErr: nil,
		},
		{
			name: "valid employee - hired in previous year",
			emp: func() *domain.Employee {
				emp := validEmployee()
				now := time.Now().UTC()
				emp.HireDate = time.Date(now.Year()-1, 12, 31, 0, 0, 0, 0, time.UTC)
				return emp
			}(),
			wantErr: nil,
		},

		// ====================================================================
		// ID Validation Errors
		// ====================================================================
		{
			name: "missing employee ID",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.ID = uuid.Nil
				return emp
			}(),
			wantErr: domain.ErrEmployeeIDRequired,
		},
		{
			name: "missing org ID",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.OrgID = uuid.Nil
				return emp
			}(),
			wantErr: domain.ErrEmployeeOrgIDRequired,
		},
		{
			name: "missing compensation package ID",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.CompensationPackageID = uuid.Nil
				return emp
			}(),
			wantErr: domain.ErrEmployeeCompensationPackageIDRequired,
		},

		// ====================================================================
		// Serial Number Validation Errors
		// ====================================================================
		{
			name: "serial number zero",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.SerialNum = 0
				return emp
			}(),
			wantErr: domain.ErrInvalidEmployeeSerialNum,
		},
		{
			name: "serial number negative",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.SerialNum = -1
				return emp
			}(),
			wantErr: domain.ErrInvalidEmployeeSerialNum,
		},

		// ====================================================================
		// Name Validation Errors
		// ====================================================================
		{
			name: "empty full name",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.FullName = ""
				return emp
			}(),
			wantErr: domain.ErrEmployeeFullNameRequired,
		},
		{
			name: "full name with only whitespace",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.FullName = "   "
				return emp
			}(),
			wantErr: domain.ErrEmployeeFullNameRequired,
		},
		{
			name: "full name with tabs and newlines",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.FullName = "\t\n  "
				return emp
			}(),
			wantErr: domain.ErrEmployeeFullNameRequired,
		},

		// ====================================================================
		// CIN Validation Errors
		// ====================================================================
		{
			name: "empty CIN number",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.CINNum = ""
				return emp
			}(),
			wantErr: domain.ErrEmployeeCINNumRequired,
		},
		{
			name: "CIN with only whitespace",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.CINNum = "   "
				return emp
			}(),
			wantErr: domain.ErrEmployeeCINNumRequired,
		},

		// ====================================================================
		// Position Validation Errors
		// ====================================================================
		{
			name: "empty position",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.Position = ""
				return emp
			}(),
			wantErr: domain.ErrEmployeePositionRequired,
		},
		{
			name: "position with only whitespace",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.Position = "   "
				return emp
			}(),
			wantErr: domain.ErrEmployeePositionRequired,
		},

		// ====================================================================
		// Gender Validation Errors
		// ====================================================================
		{
			name: "empty gender",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.Gender = ""
				return emp
			}(),
			wantErr: domain.ErrGenderNotSupported,
		},
		{
			name: "invalid gender - lowercase",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.Gender = "male"
				return emp
			}(),
			wantErr: domain.ErrGenderNotSupported,
		},
		{
			name: "invalid gender - random value",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.Gender = "OTHER"
				return emp
			}(),
			wantErr: domain.ErrGenderNotSupported,
		},

		// ====================================================================
		// Marital Status Validation Errors
		// ====================================================================
		{
			name: "empty marital status",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.MaritalStatus = ""
				return emp
			}(),
			wantErr: domain.ErrMaritalStatusNotSupported,
		},
		{
			name: "invalid marital status - lowercase",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.MaritalStatus = "single"
				return emp
			}(),
			wantErr: domain.ErrMaritalStatusNotSupported,
		},
		{
			name: "invalid marital status - random value",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.MaritalStatus = "UNKNOWN"
				return emp
			}(),
			wantErr: domain.ErrMaritalStatusNotSupported,
		},

		// ====================================================================
		// NumDependents Validation Errors
		// ====================================================================
		{
			name: "negative number of dependents",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.NumDependents = -1
				return emp
			}(),
			wantErr: domain.ErrInvalidEmployeeNumDependents,
		},

		// ====================================================================
		// NumChildren Validation Errors
		// ====================================================================
		{
			name: "negative number of children",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.NumChildren = -1
				return emp
			}(),
			wantErr: domain.ErrInvalidEmployeeNumChildren,
		},

		// ====================================================================
		// Birth Date Validation Errors
		// ====================================================================
		{
			name: "birth date too recent (age < 16)",
			emp: func() *domain.Employee {
				emp := validEmployee()
				now := time.Now().UTC()
				emp.BirthDate = now.AddDate(-15, 0, 0) // 15 years old
				return emp
			}(),
			wantErr: domain.ErrInvalidEmployeeBirthDate,
		},
		{
			name: "birth date too old (age > 80)",
			emp: func() *domain.Employee {
				emp := validEmployee()
				now := time.Now().UTC()
				emp.BirthDate = now.AddDate(-81, 0, 0) // 81 years old
				return emp
			}(),
			wantErr: domain.ErrInvalidEmployeeBirthDate,
		},
		{
			name: "birth date in the future",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.BirthDate = time.Now().UTC().AddDate(0, 0, 1)
				return emp
			}(),
			wantErr: domain.ErrInvalidEmployeeBirthDate,
		},

		// ====================================================================
		// Hire Date Validation Errors
		// ====================================================================
		{
			name: "hire date in the future",
			emp: func() *domain.Employee {
				emp := validEmployee()
				emp.HireDate = time.Now().UTC().AddDate(0, 0, 1)
				return emp
			}(),
			wantErr: domain.ErrInvalidEmployeeHireDate,
		},
		{
			name: "hire date too far in the past (> MaxHireYearsInPast)",
			emp: func() *domain.Employee {
				emp := validEmployee()
				now := time.Now().UTC()
				emp.HireDate = time.Date(now.Year()-domain.MaxHireYearsInPast-1, 1, 1, 0, 0, 0, 0, time.UTC)
				return emp
			}(),
			wantErr: domain.ErrInvalidEmployeeHireDate,
		},
		{
			name: "hired before minimum working age",
			emp: func() *domain.Employee {
				emp := validEmployee()
				now := time.Now().UTC()
				emp.BirthDate = now.AddDate(-20, 0, 0)                               // 20 years old
				emp.HireDate = emp.BirthDate.AddDate(domain.MinWorkLegalAge-1, 0, 0) // Hired at 15
				return emp
			}(),
			wantErr: domain.ErrInvalidEmployeeHireDate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.emp.Validate()

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

func TestEmployee_ValidateID(t *testing.T) {
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
			wantErr: domain.ErrEmployeeIDRequired,
		},
		{
			name:    "zero UUID",
			id:      uuid.UUID{},
			wantErr: domain.ErrEmployeeIDRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			emp := &domain.Employee{ID: tt.id}
			err := emp.ValidateID()

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateID() error = %v, wantErr nil", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateID() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestEmployee_ValidateSerialNum(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		serialNum int
		wantErr   error
	}{
		{
			name:      "valid serial number (1)",
			serialNum: 1,
			wantErr:   nil,
		},
		{
			name:      "valid serial number (100)",
			serialNum: 100,
			wantErr:   nil,
		},
		{
			name:      "valid serial number (large)",
			serialNum: 999999,
			wantErr:   nil,
		},
		{
			name:      "zero serial number",
			serialNum: 0,
			wantErr:   domain.ErrInvalidEmployeeSerialNum,
		},
		{
			name:      "negative serial number",
			serialNum: -1,
			wantErr:   domain.ErrInvalidEmployeeSerialNum,
		},
		{
			name:      "large negative serial number",
			serialNum: -999,
			wantErr:   domain.ErrInvalidEmployeeSerialNum,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			emp := &domain.Employee{SerialNum: tt.serialNum}
			err := emp.ValidateSerialNum()

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateSerialNum() error = %v, wantErr nil", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateSerialNum() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestEmployee_ValidateFullName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fullName string
		wantErr  error
	}{
		// Valid names
		{
			name:     "normal name",
			fullName: "Ahmed Ben Ali",
			wantErr:  nil,
		},
		{
			name:     "single word name",
			fullName: "Ahmed",
			wantErr:  nil,
		},
		{
			name:     "name with hyphens",
			fullName: "Jean-Pierre Dubois",
			wantErr:  nil,
		},
		{
			name:     "name with apostrophes",
			fullName: "O'Connor",
			wantErr:  nil,
		},
		{
			name:     "Arabic name",
			fullName: "أحمد بن علي",
			wantErr:  nil,
		},
		{
			name:     "name with accents",
			fullName: "François Lefèvre",
			wantErr:  nil,
		},
		{
			name:     "very long name",
			fullName: "Pablo Diego José Francisco de Paula Juan Nepomuceno María de los Remedios",
			wantErr:  nil,
		},
		{
			name:     "name with leading/trailing spaces",
			fullName: "  Ahmed Ali  ",
			wantErr:  nil, // TrimSpace should handle this
		},

		// Invalid names
		{
			name:     "empty name",
			fullName: "",
			wantErr:  domain.ErrEmployeeFullNameRequired,
		},
		{
			name:     "only spaces",
			fullName: "     ",
			wantErr:  domain.ErrEmployeeFullNameRequired,
		},
		{
			name:     "only tabs",
			fullName: "\t\t",
			wantErr:  domain.ErrEmployeeFullNameRequired,
		},
		{
			name:     "mixed whitespace",
			fullName: " \t \n ",
			wantErr:  domain.ErrEmployeeFullNameRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			emp := &domain.Employee{FullName: tt.fullName}
			err := emp.ValidateFullName()

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateFullName() error = %v, wantErr nil", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateFullName() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestEmployee_ValidateCINNum(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cinNum  string
		wantErr error
	}{
		// Valid CIN numbers (format validation not yet implemented)
		{
			name:    "normal CIN",
			cinNum:  "AB123456",
			wantErr: nil,
		},
		{
			name:    "CIN with different format",
			cinNum:  "X1234567",
			wantErr: nil,
		},
		{
			name:    "numeric CIN",
			cinNum:  "12345678",
			wantErr: nil,
		},
		{
			name:    "CIN with spaces (not trimmed yet)",
			cinNum:  "  AB123456  ",
			wantErr: nil, // TrimSpace should handle this
		},

		// Invalid CIN numbers
		{
			name:    "empty CIN",
			cinNum:  "",
			wantErr: domain.ErrEmployeeCINNumRequired,
		},
		{
			name:    "only whitespace",
			cinNum:  "   ",
			wantErr: domain.ErrEmployeeCINNumRequired,
		},
		{
			name:    "only tabs",
			cinNum:  "\t\t",
			wantErr: domain.ErrEmployeeCINNumRequired,
		}, {
			name:    "mixed whitespace",
			cinNum:  "\t \n\t ",
			wantErr: domain.ErrEmployeeCINNumRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			emp := &domain.Employee{CINNum: tt.cinNum}
			err := emp.ValidateCINNum()

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateCINNum() error = %v, wantErr nil", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateCINNum() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestEmployee_ValidateNumDependents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		numDependents int
		wantErr       error
	}{
		{
			name:          "zero dependents",
			numDependents: 0,
			wantErr:       nil,
		},
		{
			name:          "one dependent",
			numDependents: 1,
			wantErr:       nil,
		},
		{
			name:          "multiple dependents",
			numDependents: 5,
			wantErr:       nil,
		},
		{
			name:          "large number of dependents",
			numDependents: 20,
			wantErr:       nil,
		},
		{
			name:          "negative dependents",
			numDependents: -1,
			wantErr:       domain.ErrInvalidEmployeeNumDependents,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			emp := &domain.Employee{NumDependents: tt.numDependents}
			err := emp.ValidateNumDependents()

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateNumDependents() error = %v, wantErr nil", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateNumDependents() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestEmployee_ValidateNumChildren(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		numChildren int
		wantErr     error
	}{
		{
			name:        "zero children",
			numChildren: 0,
			wantErr:     nil,
		},
		{
			name:        "one kid",
			numChildren: 1,
			wantErr:     nil,
		},
		{
			name:        "multiple children",
			numChildren: 6,
			wantErr:     nil,
		},
		{
			name:        "large number of children",
			numChildren: 15,
			wantErr:     nil,
		},
		{
			name:        "negative children",
			numChildren: -1,
			wantErr:     domain.ErrInvalidEmployeeNumChildren,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			emp := &domain.Employee{NumChildren: tt.numChildren}
			err := emp.ValidateNumChildren()

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateNumChildren() error = %v, wantErr nil", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateNumChildren() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestEmployee_ValidateBirthDate(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name      string
		birthDate time.Time
		wantErr   error
	}{
		// Valid birth dates
		{
			name:      "exactly minimum age (16 years)",
			birthDate: now.AddDate(-domain.MinWorkLegalAge, 0, 0),
			wantErr:   nil,
		},
		{
			name:      "middle age (30 years)",
			birthDate: now.AddDate(-30, 0, 0),
			wantErr:   nil,
		},
		{
			name:      "just under maximum age (79 years)",
			birthDate: now.AddDate(-79, 0, 0),
			wantErr:   nil,
		},
		{
			name:      "exactly maximum age boundary",
			birthDate: now.AddDate(-domain.MaxWorkLegalAge, 0, 1), // Just under 80
			wantErr:   nil,
		},

		// Invalid birth dates
		{
			name:      "too young (15 years)",
			birthDate: now.AddDate(-15, 0, 0),
			wantErr:   domain.ErrInvalidEmployeeBirthDate,
		},
		{
			name:      "too old (81 years)",
			birthDate: now.AddDate(-81, 0, 0),
			wantErr:   domain.ErrInvalidEmployeeBirthDate,
		},
		{
			name:      "in the future (tomorrow)",
			birthDate: now.AddDate(0, 0, 1),
			wantErr:   domain.ErrInvalidEmployeeBirthDate,
		},
		{
			name:      "in the future (1 year)",
			birthDate: now.AddDate(1, 0, 0),
			wantErr:   domain.ErrInvalidEmployeeBirthDate,
		},
		{
			name:      "way too old (100 years)",
			birthDate: now.AddDate(-100, 0, 0),
			wantErr:   domain.ErrInvalidEmployeeBirthDate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			emp := &domain.Employee{BirthDate: tt.birthDate}
			err := emp.ValidateBirthDate()

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateBirthDate() error = %v, wantErr nil", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateBirthDate() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestEmployee_ValidateHireDate(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	currentYear := now.Year()

	tests := []struct {
		name     string
		hireDate time.Time
		wantErr  error
	}{
		// Valid hire dates
		{
			name:     "hired today",
			hireDate: now,
			wantErr:  nil,
		},
		{
			name:     "hired last month",
			hireDate: now.AddDate(0, -1, 0),
			wantErr:  nil,
		},
		{
			name:     "hired at start of current year",
			hireDate: time.Date(currentYear, 1, 1, 0, 0, 0, 0, time.UTC),
			wantErr:  nil,
		},
		{
			name:     "hired last year (within MaxHireYearsInPast)",
			hireDate: time.Date(currentYear-1, 12, 31, 0, 0, 0, 0, time.UTC),
			wantErr:  nil,
		},

		// Invalid hire dates
		{
			name:     "in the future (tomorrow)",
			hireDate: now.AddDate(0, 0, 1),
			wantErr:  domain.ErrInvalidEmployeeHireDate,
		},
		{
			name:     "in the future (next month)",
			hireDate: now.AddDate(0, 1, 0),
			wantErr:  domain.ErrInvalidEmployeeHireDate,
		},
		{
			name:     "too far in past (beyond MaxHireYearsInPast)",
			hireDate: time.Date(currentYear-domain.MaxHireYearsInPast-1, 12, 31, 0, 0, 0, 0, time.UTC),
			wantErr:  domain.ErrInvalidEmployeeHireDate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			emp := &domain.Employee{HireDate: tt.hireDate}
			err := emp.ValidateHireDate()

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateHireDate() error = %v, wantErr nil", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateHireDate() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestEmployee_ValidateMinHireDate(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name      string
		birthDate time.Time
		hireDate  time.Time
		wantErr   error
	}{
		// Valid cases
		{
			name:      "hired exactly at minimum age (16 years after birth)",
			birthDate: now.AddDate(-20, 0, 0),
			hireDate:  now.AddDate(-20+domain.MinWorkLegalAge, 0, 0),
			wantErr:   nil,
		},
		{
			name:      "hired well after minimum age",
			birthDate: now.AddDate(-30, 0, 0),
			hireDate:  now.AddDate(-10, 0, 0), // Hired at 20
			wantErr:   nil,
		},
		{
			name:      "hired recently (current age > 16)",
			birthDate: now.AddDate(-25, 0, 0),
			hireDate:  now.AddDate(0, -1, 0), // Hired last month
			wantErr:   nil,
		},

		// Invalid cases
		{
			name:      "hired before minimum age (15 years old)",
			birthDate: now.AddDate(-20, 0, 0),
			hireDate:  now.AddDate(-20+domain.MinWorkLegalAge-1, 0, 0),
			wantErr:   domain.ErrInvalidEmployeeHireDate,
		},
		{
			name:      "hired way before minimum age (10 years old)",
			birthDate: now.AddDate(-25, 0, 0),
			hireDate:  now.AddDate(-15, 0, 0), // Hired at 10
			wantErr:   domain.ErrInvalidEmployeeHireDate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			emp := &domain.Employee{
				BirthDate: tt.birthDate,
				HireDate:  tt.hireDate,
			}
			err := emp.ValidateMinHireDate()

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateMinHireDate() error = %v, wantErr nil", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateMinHireDate() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

// ============================================================================
// Enum Tests
// ============================================================================

func TestGenderEnum_IsSupported(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		gender domain.GenderEnum
		want   bool
	}{
		{
			name:   "MALE is supported",
			gender: domain.GenderMale,
			want:   true,
		},
		{
			name:   "FEMALE is supported",
			gender: domain.GenderFemale,
			want:   true,
		},
		{
			name:   "empty string not supported",
			gender: "",
			want:   false,
		},
		{
			name:   "lowercase not supported",
			gender: "male",
			want:   false,
		},
		{
			name:   "OTHER not supported",
			gender: "OTHER",
			want:   false,
		},
		{
			name:   "random value not supported",
			gender: "INVALID",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.gender.IsSupported()
			if got != tt.want {
				t.Errorf("IsSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaritalStatusEnum_IsSupported(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		maritalStatus domain.MaritalStatusEnum
		want          bool
	}{
		{
			name:          "SINGLE is supported",
			maritalStatus: domain.MaritalStatusSingle,
			want:          true,
		},
		{
			name:          "MARRIED is supported",
			maritalStatus: domain.MaritalStatusMarried,
			want:          true,
		},
		{
			name:          "SEPARATED is supported",
			maritalStatus: domain.MaritalStatusSeparated,
			want:          true,
		},
		{
			name:          "DIVORCED is supported",
			maritalStatus: domain.MaritalStatusDivorced,
			want:          true,
		},
		{
			name:          "WIDOWED is supported",
			maritalStatus: domain.MaritalStatusWidowed,
			want:          true,
		},
		{
			name:          "empty string not supported",
			maritalStatus: "",
			want:          false,
		},
		{
			name:          "lowercase not supported",
			maritalStatus: "single",
			want:          false,
		},
		{
			name:          "UNKNOWN not supported",
			maritalStatus: "UNKNOWN",
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.maritalStatus.IsSupported()
			if got != tt.want {
				t.Errorf("IsSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// EmployeeCompensationPackage Tests
// ============================================================================

func TestEmployeeCompensationPackage_Validate(t *testing.T) {
	t.Parallel()

	validPackage := func() *domain.EmployeeCompensationPackage {
		baseSalary, _ := money.FromMAD(5000.00)
		return &domain.EmployeeCompensationPackage{
			ID:         uuid.New(),
			OrgID:      uuid.New(),
			Name:       "Standard Package",
			Currency:   money.MAD,
			BaseSalary: baseSalary,
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}
	}

	tests := []struct {
		name    string
		pkg     *domain.EmployeeCompensationPackage
		wantErr error
	}{
		// Valid cases
		{
			name:    "valid package with normal salary",
			pkg:     validPackage(),
			wantErr: nil,
		},
		{
			name: "valid package with minimum wage (SMIG)",
			pkg: func() *domain.EmployeeCompensationPackage {
				pkg := validPackage()
				pkg.BaseSalary = domain.MinWageSMIG
				return pkg
			}(),
			wantErr: nil,
		},
		{
			name: "valid package with high salary",
			pkg: func() *domain.EmployeeCompensationPackage {
				pkg := validPackage()
				salary, _ := money.FromMAD(50000.00)
				pkg.BaseSalary = salary
				return pkg
			}(),
			wantErr: nil,
		},

		// ID errors
		{
			name: "missing ID",
			pkg: func() *domain.EmployeeCompensationPackage {
				pkg := validPackage()
				pkg.ID = uuid.Nil
				return pkg
			}(),
			wantErr: domain.ErrEmployeeCompensationPackageIDRequired,
		},

		// BaseSalary errors
		{
			name: "salary below SMIG",
			pkg: func() *domain.EmployeeCompensationPackage {
				pkg := validPackage()
				belowSMIG, _ := money.FromMAD(3000.00) // Below SMIG
				pkg.BaseSalary = belowSMIG
				return pkg
			}(),
			wantErr: domain.ErrInvalidEmployeeCompensationPackageBaseSalary,
		},
		{
			name: "zero salary",
			pkg: func() *domain.EmployeeCompensationPackage {
				pkg := validPackage()
				pkg.BaseSalary = money.FromCents(0)
				return pkg
			}(),
			wantErr: domain.ErrInvalidEmployeeCompensationPackageBaseSalary,
		},
		{
			name: "negative salary",
			pkg: func() *domain.EmployeeCompensationPackage {
				pkg := validPackage()
				pkg.BaseSalary = money.FromCents(-10000)
				return pkg
			}(),
			wantErr: domain.ErrInvalidEmployeeCompensationPackageBaseSalary,
		},

		// Currency errors
		{
			name: "unsupported currency",
			pkg: func() *domain.EmployeeCompensationPackage {
				pkg := validPackage()
				pkg.Currency = "USD"
				return pkg
			}(),
			wantErr: money.ErrCurrencyNotSupported,
		},
		{
			name: "empty currency",
			pkg: func() *domain.EmployeeCompensationPackage {
				pkg := validPackage()
				pkg.Currency = ""
				return pkg
			}(),
			wantErr: money.ErrCurrencyNotSupported,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.pkg.Validate()

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
// Benchmark Tests
// ============================================================================

func BenchmarkEmployee_Validate(b *testing.B) {
	now := time.Now().UTC()
	emp := &domain.Employee{
		ID:                    uuid.New(),
		OrgID:                 uuid.New(),
		SerialNum:             1,
		FullName:              "Ahmed Ben Ali",
		BirthDate:             now.AddDate(-30, 0, 0),
		Gender:                domain.GenderMale,
		MaritalStatus:         domain.MaritalStatusSingle,
		NumDependents:         0,
		NumChildren:           0,
		CINNum:                "AB123456",
		HireDate:              now.AddDate(0, -6, 0),
		Position:              "Developer",
		CompensationPackageID: uuid.New(),
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = emp.Validate()
	}
}

func BenchmarkEmployee_ValidateFullName(b *testing.B) {
	emp := &domain.Employee{FullName: "Ahmed Ben Ali"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = emp.ValidateFullName()
	}
}

func BenchmarkGenderEnum_IsSupported(b *testing.B) {
	gender := domain.GenderMale

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gender.IsSupported()
	}
}

func BenchmarkEmployeeCompensationPackage_Validate(b *testing.B) {
	salary, _ := money.FromMAD(5000.00)
	pkg := &domain.EmployeeCompensationPackage{
		ID:         uuid.New(),
		Currency:   money.MAD,
		BaseSalary: salary,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pkg.Validate()
	}
}
