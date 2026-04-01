package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iamoeg/fman/internal/domain"
	"github.com/iamoeg/fman/pkg/money"
)

// ============================================================================
// PayrollPeriod Validation Tests
// ============================================================================

func TestPayrollPeriod_Validate(t *testing.T) {
	t.Parallel()

	// Valid base payroll period for tests
	validPeriod := func() *domain.PayrollPeriod {
		now := time.Now().UTC()
		return &domain.PayrollPeriod{
			ID:        uuid.New(),
			OrgID:     uuid.New(),
			Year:      2026,
			Month:     1,
			Status:    domain.PayrollPeriodStatusDraft,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	tests := []struct {
		name    string
		period  *domain.PayrollPeriod
		wantErr error
	}{
		// ====================================================================
		// Valid Cases
		// ====================================================================
		{
			name:    "valid draft period",
			period:  validPeriod(),
			wantErr: nil,
		},
		{
			name: "valid finalized period",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.Status = domain.PayrollPeriodStatusFinalized
				finalizedAt := time.Now().UTC().Add(-1 * time.Nanosecond)
				p.FinalizedAt = &finalizedAt
				return p
			}(),
			wantErr: nil,
		},
		{
			name: "valid period - January (month 1)",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.Month = 1
				return p
			}(),
			wantErr: nil,
		},
		{
			name: "valid period - December (month 12)",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.Month = 12
				return p
			}(),
			wantErr: nil,
		},
		{
			name: "valid period - minimum year (2020)",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.Year = domain.PayrollPeriodMinYear
				return p
			}(),
			wantErr: nil,
		},
		{
			name: "valid period - maximum year (2050)",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.Year = domain.PayrollPeriodMaxYear
				return p
			}(),
			wantErr: nil,
		},

		// ====================================================================
		// ID Validation Errors
		// ====================================================================
		{
			name: "missing period ID",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.ID = uuid.Nil
				return p
			}(),
			wantErr: domain.ErrPayrollPeriodIDRequired,
		},
		{
			name: "missing org ID",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.OrgID = uuid.Nil
				return p
			}(),
			wantErr: domain.ErrPayrollPeriodOrgIDRequired,
		},

		// ====================================================================
		// Year Validation Errors
		// ====================================================================
		{
			name: "year too low (below PayrollPeriodMinYear)",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.Year = domain.PayrollPeriodMinYear - 1
				return p
			}(),
			wantErr: domain.ErrInvalidPayrollPeriodYear,
		},
		{
			name: "year too high (above PayrollPeriodMaxYear)",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.Year = domain.PayrollPeriodMaxYear + 1
				return p
			}(),
			wantErr: domain.ErrInvalidPayrollPeriodYear,
		},
		{
			name: "year zero",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.Year = 0
				return p
			}(),
			wantErr: domain.ErrInvalidPayrollPeriodYear,
		},
		{
			name: "negative year",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.Year = -2026
				return p
			}(),
			wantErr: domain.ErrInvalidPayrollPeriodYear,
		},

		// ====================================================================
		// Month Validation Errors
		// ====================================================================
		{
			name: "month zero",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.Month = 0
				return p
			}(),
			wantErr: domain.ErrInvalidPayrollPeriodMonth,
		},
		{
			name: "month too high (13)",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.Month = 13
				return p
			}(),
			wantErr: domain.ErrInvalidPayrollPeriodMonth,
		},
		{
			name: "negative month",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.Month = -1
				return p
			}(),
			wantErr: domain.ErrInvalidPayrollPeriodMonth,
		},

		// ====================================================================
		// Status Validation Errors
		// ====================================================================
		{
			name: "empty status",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.Status = ""
				return p
			}(),
			wantErr: domain.ErrInvalidPayrollPeriodStatus,
		},
		{
			name: "invalid status - lowercase",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.Status = "draft"
				return p
			}(),
			wantErr: domain.ErrInvalidPayrollPeriodStatus,
		},
		{
			name: "invalid status - random value",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.Status = "RANDOM_VALUE"
				return p
			}(),
			wantErr: domain.ErrInvalidPayrollPeriodStatus,
		},

		// ====================================================================
		// State Consistency Errors
		// ====================================================================
		{
			name: "draft with finalized_at set",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.Status = domain.PayrollPeriodStatusDraft
				finalizedAt := time.Now().UTC()
				p.FinalizedAt = &finalizedAt
				return p
			}(),
			wantErr: domain.ErrInvalidPayrollPeriodState,
		},
		{
			name: "finalized without finalized_at",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.Status = domain.PayrollPeriodStatusFinalized
				p.FinalizedAt = nil
				return p
			}(),
			wantErr: domain.ErrInvalidPayrollPeriodState,
		},
		{
			name: "finalized_at before created_at",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.Status = domain.PayrollPeriodStatusFinalized
				p.CreatedAt = time.Now().UTC()
				finalizedAt := p.CreatedAt.Add(-1 * time.Hour) // 1 hour before creation
				p.FinalizedAt = &finalizedAt
				return p
			}(),
			wantErr: domain.ErrInvalidPayrollPeriodState,
		},
		{
			name: "finalized_at in the future",
			period: func() *domain.PayrollPeriod {
				p := validPeriod()
				p.Status = domain.PayrollPeriodStatusFinalized
				finalizedAt := time.Now().UTC().Add(1 * time.Hour) // 1 hour in future
				p.FinalizedAt = &finalizedAt
				return p
			}(),
			wantErr: domain.ErrInvalidPayrollPeriodState,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.period.Validate()

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
// PayrollPeriod Status Enum Tests
// ============================================================================

func TestPayrollPeriodStatusEnum_IsSupported(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status domain.PayrollPeriodStatusEnum
		want   bool
	}{
		{
			name:   "DRAFT is supported",
			status: domain.PayrollPeriodStatusDraft,
			want:   true,
		},
		{
			name:   "FINALIZED is supported",
			status: domain.PayrollPeriodStatusFinalized,
			want:   true,
		},
		{
			name:   "empty string not supported",
			status: "",
			want:   false,
		},
		{
			name:   "lowercase not supported",
			status: "draft",
			want:   false,
		},
		{
			name:   "PENDING not supported",
			status: "PENDING",
			want:   false,
		},
		{
			name:   "random value not supported",
			status: "RANDOM_VALUE",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.status.IsSupported()
			if got != tt.want {
				t.Errorf("IsSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// PayrollResult Validation Tests
// ============================================================================

func TestPayrollResult_Validate(t *testing.T) {
	t.Parallel()

	// Valid base payroll result for tests
	validResult := func() *domain.PayrollResult {
		now := time.Now().UTC()

		// Create mathematically consistent payroll result
		baseSalary := money.FromCents(500000)            // 5,000.00 MAD
		seniorityBonus := money.FromCents(50000)         // 500.00 MAD
		grossSalary, _ := baseSalary.Add(seniorityBonus) // 5,500.00 MAD

		totalOtherBonus := money.FromCents(0)
		grossSalaryGrandTotal, _ := grossSalary.Add(totalOtherBonus) // 5,500.00 MAD

		totalExemptions := money.FromCents(110000)                               // 1,100.00 MAD (20% of gross)
		taxableGrossSalary, _ := grossSalaryGrandTotal.Subtract(totalExemptions) // 4,400.00 MAD

		// CNSS employee contributions (no AMO here)
		socialAllowanceEmp := money.FromCents(20000)          // 200.00 MAD
		jobLossEmp := money.FromCents(10000)                  // 100.00 MAD
		totalCNSSEmp, _ := socialAllowanceEmp.Add(jobLossEmp) // 300.00 MAD

		// CNSS employer contributions (no AMO here)
		socialAllowanceEmployer := money.FromCents(50000) // 500.00 MAD
		jobLossEmployer := money.FromCents(25000)         // 250.00 MAD
		trainingTax := money.FromCents(30000)             // 300.00 MAD
		familyBenefits := money.FromCents(40000)          // 400.00 MAD
		totalCNSSEmployer, _ := socialAllowanceEmployer.Add(jobLossEmployer)
		totalCNSSEmployer, _ = totalCNSSEmployer.Add(trainingTax)
		totalCNSSEmployer, _ = totalCNSSEmployer.Add(familyBenefits) // 1,450.00 MAD

		// AMO (separate from CNSS)
		amoEmployee := money.FromCents(11000) // 110.00 MAD
		amoEmployer := money.FromCents(22000) // 220.00 MAD

		// Taxable net salary
		taxableNetSalary, _ := taxableGrossSalary.Subtract(totalCNSSEmp)
		taxableNetSalary, _ = taxableNetSalary.Subtract(amoEmployee) // 3,990.00 MAD

		// Income tax
		incomeTax := money.FromCents(39900) // 399.00 MAD (10% of taxable net)

		// Net to pay
		netToPay, _ := taxableNetSalary.Subtract(incomeTax)
		roundingAmount := money.FromCents(0)       // no rounding
		netToPay, _ = netToPay.Add(roundingAmount) // 3,591.00 MAD

		return &domain.PayrollResult{
			ID:                    uuid.New(),
			PayrollPeriodID:       uuid.New(),
			EmployeeID:            uuid.New(),
			CompensationPackageID: uuid.New(),
			Currency:              money.MAD,

			BaseSalary:            baseSalary,
			SeniorityBonus:        seniorityBonus,
			GrossSalary:           grossSalary,
			TotalOtherBonus:       totalOtherBonus,
			GrossSalaryGrandTotal: grossSalaryGrandTotal,

			TotalExemptions:    totalExemptions,
			TaxableGrossSalary: taxableGrossSalary,

			SocialAllowanceEmployeeContrib:     socialAllowanceEmp,
			SocialAllowanceEmployerContrib:     socialAllowanceEmployer,
			JobLossCompensationEmployeeContrib: jobLossEmp,
			JobLossCompensationEmployerContrib: jobLossEmployer,
			TrainingTaxEmployerContrib:         trainingTax,
			FamilyBenefitsEmployerContrib:      familyBenefits,
			TotalCNSSEmployeeContrib:           totalCNSSEmp,
			TotalCNSSEmployerContrib:           totalCNSSEmployer,

			AMOEmployeeContrib: amoEmployee,
			AMOEmployerContrib: amoEmployer,

			TaxableNetSalary: taxableNetSalary,
			IncomeTax:        incomeTax,
			RoundingAmount:   roundingAmount,
			NetToPay:         netToPay,

			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	tests := []struct {
		name    string
		result  *domain.PayrollResult
		wantErr error
	}{
		// ====================================================================
		// Valid Cases
		// ====================================================================
		{
			name:    "valid payroll result with all fields",
			result:  validResult(),
			wantErr: nil,
		},
		{
			name: "valid result with zero bonuses",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.SeniorityBonus = money.FromCents(0)
				r.TotalOtherBonus = money.FromCents(0)
				// Recalculate dependent fields
				r.GrossSalary = r.BaseSalary
				r.GrossSalaryGrandTotal = r.BaseSalary
				// Continue recalculating chain...
				r.TaxableGrossSalary, _ = r.GrossSalaryGrandTotal.Subtract(r.TotalExemptions)
				r.TaxableNetSalary, _ = r.TaxableGrossSalary.Subtract(r.TotalCNSSEmployeeContrib)
				r.TaxableNetSalary, _ = r.TaxableNetSalary.Subtract(r.AMOEmployeeContrib)
				r.NetToPay, _ = r.TaxableNetSalary.Subtract(r.IncomeTax)
				r.NetToPay, _ = r.NetToPay.Add(r.RoundingAmount)
				return r
			}(),
			wantErr: nil,
		},
		{
			name: "valid result with negative rounding",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.RoundingAmount = money.FromCents(-50) // -0.50 MAD
				r.NetToPay, _ = r.TaxableNetSalary.Subtract(r.IncomeTax)
				r.NetToPay, _ = r.NetToPay.Add(r.RoundingAmount)
				return r
			}(),
			wantErr: nil,
		},
		{
			name: "valid result with maximum rounding (+100 cents)",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.RoundingAmount = money.FromCents(100) // +1.00 MAD
				r.NetToPay, _ = r.TaxableNetSalary.Subtract(r.IncomeTax)
				r.NetToPay, _ = r.NetToPay.Add(r.RoundingAmount)
				return r
			}(),
			wantErr: nil,
		},
		{
			name: "valid result with minimum rounding (-100 cents)",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.RoundingAmount = money.FromCents(-100) // -1.00 MAD
				r.NetToPay, _ = r.TaxableNetSalary.Subtract(r.IncomeTax)
				r.NetToPay, _ = r.NetToPay.Add(r.RoundingAmount)
				return r
			}(),
			wantErr: nil,
		},

		// ====================================================================
		// ID Validation Errors
		// ====================================================================
		{
			name: "missing result ID",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.ID = uuid.Nil
				return r
			}(),
			wantErr: domain.ErrPayrollResultIDRequired,
		},
		{
			name: "missing employee ID",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.EmployeeID = uuid.Nil
				return r
			}(),
			wantErr: domain.ErrPayrollResultEmployeeIDRequired,
		},
		{
			name: "missing payroll period ID",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.PayrollPeriodID = uuid.Nil
				return r
			}(),
			wantErr: domain.ErrPayrollPeriodIDRequired,
		},
		{
			name: "missing compensation package ID",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.CompensationPackageID = uuid.Nil
				return r
			}(),
			wantErr: domain.ErrPayrollResultCompensationPackageIDRequired,
		},

		// ====================================================================
		// Currency Validation Errors
		// ====================================================================
		{
			name: "unsupported currency",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.Currency = "USD"
				return r
			}(),
			wantErr: money.ErrCurrencyNotSupported,
		},
		{
			name: "empty currency",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.Currency = ""
				return r
			}(),
			wantErr: money.ErrCurrencyNotSupported,
		},

		// ====================================================================
		// Positive Money Value Errors
		// ====================================================================
		{
			name: "negative base salary",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.BaseSalary = money.FromCents(-100000)
				return r
			}(),
			wantErr: domain.ErrInvalidPayrollResultMoneyValue,
		},
		{
			name: "negative gross salary",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.GrossSalary = money.FromCents(-50000)
				return r
			}(),
			wantErr: domain.ErrInvalidPayrollResultMoneyValue,
		},
		{
			name: "negative net to pay",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.NetToPay = money.FromCents(-10000)
				return r
			}(),
			wantErr: domain.ErrInvalidPayrollResultMoneyValue,
		},
		{
			name: "negative CNSS employee contribution",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.TotalCNSSEmployeeContrib = money.FromCents(-5000)
				return r
			}(),
			wantErr: domain.ErrInvalidPayrollResultMoneyValue,
		},

		// ====================================================================
		// Rounding Amount Errors
		// ====================================================================
		{
			name: "rounding amount too high (>100 cents)",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.RoundingAmount = money.FromCents(101)
				return r
			}(),
			wantErr: domain.ErrInvalidPayrollResultRoundingAmount,
		},
		{
			name: "rounding amount too low (<-100 cents)",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.RoundingAmount = money.FromCents(-101)
				return r
			}(),
			wantErr: domain.ErrInvalidPayrollResultRoundingAmount,
		},
		{
			name: "rounding amount way too high (500 cents)",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.RoundingAmount = money.FromCents(500)
				return r
			}(),
			wantErr: domain.ErrInvalidPayrollResultRoundingAmount,
		},

		// ====================================================================
		// Mathematical Consistency Errors
		// ====================================================================
		{
			name: "gross salary inconsistent with base + seniority",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.GrossSalary = money.FromCents(999999) // Wrong value
				return r
			}(),
			wantErr: domain.ErrInconsistentPayrollResultCalculation,
		},
		{
			name: "gross salary grand total inconsistent",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.GrossSalaryGrandTotal = money.FromCents(999999) // Wrong value
				return r
			}(),
			wantErr: domain.ErrInconsistentPayrollResultCalculation,
		},
		{
			name: "total CNSS employee contribution inconsistent",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.TotalCNSSEmployeeContrib = money.FromCents(999999) // Wrong value
				return r
			}(),
			wantErr: domain.ErrInconsistentPayrollResultCalculation,
		},
		{
			name: "total CNSS employer contribution inconsistent",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.TotalCNSSEmployerContrib = money.FromCents(999999) // Wrong value
				return r
			}(),
			wantErr: domain.ErrInconsistentPayrollResultCalculation,
		},
		{
			name: "taxable gross salary inconsistent",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.TaxableGrossSalary = money.FromCents(999999) // Wrong value
				return r
			}(),
			wantErr: domain.ErrInconsistentPayrollResultCalculation,
		},
		{
			name: "taxable net salary inconsistent",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.TaxableNetSalary = money.FromCents(999999) // Wrong value
				return r
			}(),
			wantErr: domain.ErrInconsistentPayrollResultCalculation,
		},
		{
			name: "net to pay inconsistent",
			result: func() *domain.PayrollResult {
				r := validResult()
				r.NetToPay = money.FromCents(999999) // Wrong value
				return r
			}(),
			wantErr: domain.ErrInconsistentPayrollResultCalculation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.result.Validate()

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
// PayrollResult Helper Method Tests
// ============================================================================

func TestPayrollResult_TotalDueToCNSS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		cnssEmployee  money.Money
		cnssEmployer  money.Money
		amoEmployee   money.Money
		amoEmployer   money.Money
		expectedTotal int64 // in cents
		wantErr       bool
	}{
		{
			name:          "normal case",
			cnssEmployee:  money.FromCents(30000),  // 300.00 MAD
			cnssEmployer:  money.FromCents(147500), // 1,475.00 MAD
			amoEmployee:   money.FromCents(11000),  // 110.00 MAD
			amoEmployer:   money.FromCents(22000),  // 220.00 MAD
			expectedTotal: 210500,                  // 2,105.00 MAD total
			wantErr:       false,
		},
		{
			name:          "zero contributions",
			cnssEmployee:  money.FromCents(0),
			cnssEmployer:  money.FromCents(0),
			amoEmployee:   money.FromCents(0),
			amoEmployer:   money.FromCents(0),
			expectedTotal: 0,
			wantErr:       false,
		},
		{
			name:          "only CNSS contributions",
			cnssEmployee:  money.FromCents(50000),
			cnssEmployer:  money.FromCents(100000),
			amoEmployee:   money.FromCents(0),
			amoEmployer:   money.FromCents(0),
			expectedTotal: 150000,
			wantErr:       false,
		},
		{
			name:          "only AMO contributions",
			cnssEmployee:  money.FromCents(0),
			cnssEmployer:  money.FromCents(0),
			amoEmployee:   money.FromCents(25000),
			amoEmployer:   money.FromCents(50000),
			expectedTotal: 75000,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := &domain.PayrollResult{
				TotalCNSSEmployeeContrib: tt.cnssEmployee,
				TotalCNSSEmployerContrib: tt.cnssEmployer,
				AMOEmployeeContrib:       tt.amoEmployee,
				AMOEmployerContrib:       tt.amoEmployer,
			}

			total, err := result.TotalDueToCNSS()

			if tt.wantErr {
				if err == nil {
					t.Error("TotalDueToCNSS() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("TotalDueToCNSS() unexpected error: %v", err)
					return
				}
				if total.Cents() != tt.expectedTotal {
					t.Errorf("TotalDueToCNSS() = %d cents, want %d cents", total.Cents(), tt.expectedTotal)
				}
			}
		})
	}
}

func TestPayrollResult_TotalEmployeeDeductions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		cnssEmployee  money.Money
		amoEmployee   money.Money
		expectedTotal int64 // in cents
		wantErr       bool
	}{
		{
			name:          "normal deductions",
			cnssEmployee:  money.FromCents(30000), // 300.00 MAD
			amoEmployee:   money.FromCents(11000), // 110.00 MAD
			expectedTotal: 41000,                  // 410.00 MAD total
			wantErr:       false,
		},
		{
			name:          "zero deductions",
			cnssEmployee:  money.FromCents(0),
			amoEmployee:   money.FromCents(0),
			expectedTotal: 0,
			wantErr:       false,
		},
		{
			name:          "only CNSS",
			cnssEmployee:  money.FromCents(50000),
			amoEmployee:   money.FromCents(0),
			expectedTotal: 50000,
			wantErr:       false,
		},
		{
			name:          "only AMO",
			cnssEmployee:  money.FromCents(0),
			amoEmployee:   money.FromCents(25000),
			expectedTotal: 25000,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := &domain.PayrollResult{
				TotalCNSSEmployeeContrib: tt.cnssEmployee,
				AMOEmployeeContrib:       tt.amoEmployee,
			}

			total, err := result.TotalEmployeeDeductions()

			if tt.wantErr {
				if err == nil {
					t.Error("TotalEmployeeDeductions() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("TotalEmployeeDeductions() unexpected error: %v", err)
					return
				}
				if total.Cents() != tt.expectedTotal {
					t.Errorf("TotalEmployeeDeductions() = %d cents, want %d cents", total.Cents(), tt.expectedTotal)
				}
			}
		})
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkPayrollPeriod_Validate(b *testing.B) {
	now := time.Now().UTC()
	period := &domain.PayrollPeriod{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		Year:      2026,
		Month:     1,
		Status:    domain.PayrollPeriodStatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = period.Validate()
	}
}

func BenchmarkPayrollResult_Validate(b *testing.B) {
	now := time.Now().UTC()
	baseSalary := money.FromCents(500000)
	seniorityBonus := money.FromCents(50000)
	grossSalary, _ := baseSalary.Add(seniorityBonus)

	result := &domain.PayrollResult{
		ID:                                 uuid.New(),
		PayrollPeriodID:                    uuid.New(),
		EmployeeID:                         uuid.New(),
		CompensationPackageID:              uuid.New(),
		Currency:                           money.MAD,
		BaseSalary:                         baseSalary,
		SeniorityBonus:                     seniorityBonus,
		GrossSalary:                        grossSalary,
		TotalOtherBonus:                    money.FromCents(0),
		GrossSalaryGrandTotal:              grossSalary,
		TotalCNSSEmployeeContrib:           money.FromCents(30000),
		TotalCNSSEmployerContrib:           money.FromCents(147500),
		AMOEmployeeContrib:                 money.FromCents(11000),
		AMOEmployerContrib:                 money.FromCents(22000),
		SocialAllowanceEmployeeContrib:     money.FromCents(20000),
		SocialAllowanceEmployerContrib:     money.FromCents(50000),
		JobLossCompensationEmployeeContrib: money.FromCents(10000),
		JobLossCompensationEmployerContrib: money.FromCents(25000),
		TrainingTaxEmployerContrib:         money.FromCents(30000),
		FamilyBenefitsEmployerContrib:      money.FromCents(40000),
		TotalExemptions:                    money.FromCents(110000),
		TaxableGrossSalary:                 money.FromCents(440000),
		TaxableNetSalary:                   money.FromCents(399000),
		IncomeTax:                          money.FromCents(39900),
		RoundingAmount:                     money.FromCents(10),
		NetToPay:                           money.FromCents(359110),
		CreatedAt:                          now,
		UpdatedAt:                          now,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = result.Validate()
	}
}

func BenchmarkPayrollResult_TotalDueToCNSS(b *testing.B) {
	result := &domain.PayrollResult{
		TotalCNSSEmployeeContrib: money.FromCents(30000),
		TotalCNSSEmployerContrib: money.FromCents(147500),
		AMOEmployeeContrib:       money.FromCents(11000),
		AMOEmployerContrib:       money.FromCents(22000),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = result.TotalDueToCNSS()
	}
}

func BenchmarkPayrollPeriodStatusEnum_IsSupported(b *testing.B) {
	status := domain.PayrollPeriodStatusDraft

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = status.IsSupported()
	}
}
