package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sqlite "github.com/iamoeg/fman/internal/adapter/sqlite"
	"github.com/iamoeg/fman/internal/application"
	"github.com/iamoeg/fman/internal/domain"
	"github.com/iamoeg/fman/pkg/money"
)

// ===============================================================================
// MOCK REPOSITORIES
// ===============================================================================

// mockPayrollPeriodRepository implements payrollPeriodRepository interface for testing
type mockPayrollPeriodRepository struct {
	createFunc                             func(context.Context, *domain.PayrollPeriod) error
	updateFunc                             func(context.Context, *domain.PayrollPeriod) error
	deleteFunc                             func(context.Context, uuid.UUID) error
	restoreFunc                            func(context.Context, uuid.UUID) error
	hardDeleteFunc                         func(context.Context, uuid.UUID) error
	finalizeFunc                           func(context.Context, uuid.UUID) error
	unfinalizeFunc                         func(context.Context, uuid.UUID) error
	findByIDFunc                           func(context.Context, uuid.UUID) (*domain.PayrollPeriod, error)
	findByIDIncludingDeletedFunc           func(context.Context, uuid.UUID) (*domain.PayrollPeriod, error)
	findByOrgYearMonthFunc                 func(context.Context, uuid.UUID, int, int) (*domain.PayrollPeriod, error)
	findByOrganizationFunc                 func(context.Context, uuid.UUID) ([]*domain.PayrollPeriod, error)
	findByOrganizationIncludingDeletedFunc func(context.Context, uuid.UUID) ([]*domain.PayrollPeriod, error)
	findAllDraftFunc                       func(context.Context) ([]*domain.PayrollPeriod, error)
	findAllFunc                            func(context.Context) ([]*domain.PayrollPeriod, error)
	findAllIncludingDeletedFunc            func(context.Context) ([]*domain.PayrollPeriod, error)
}

func (m *mockPayrollPeriodRepository) Create(ctx context.Context, period *domain.PayrollPeriod) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, period)
	}
	return nil
}

func (m *mockPayrollPeriodRepository) Update(ctx context.Context, period *domain.PayrollPeriod) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, period)
	}
	return nil
}

func (m *mockPayrollPeriodRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockPayrollPeriodRepository) Restore(ctx context.Context, id uuid.UUID) error {
	if m.restoreFunc != nil {
		return m.restoreFunc(ctx, id)
	}
	return nil
}

func (m *mockPayrollPeriodRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	if m.hardDeleteFunc != nil {
		return m.hardDeleteFunc(ctx, id)
	}
	return nil
}

func (m *mockPayrollPeriodRepository) Finalize(ctx context.Context, id uuid.UUID) error {
	if m.finalizeFunc != nil {
		return m.finalizeFunc(ctx, id)
	}
	return nil
}

func (m *mockPayrollPeriodRepository) Unfinalize(ctx context.Context, id uuid.UUID) error {
	if m.unfinalizeFunc != nil {
		return m.unfinalizeFunc(ctx, id)
	}
	return nil
}

func (m *mockPayrollPeriodRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, sqlite.ErrRecordNotFound
}

func (m *mockPayrollPeriodRepository) FindByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
	if m.findByIDIncludingDeletedFunc != nil {
		return m.findByIDIncludingDeletedFunc(ctx, id)
	}
	return nil, sqlite.ErrRecordNotFound
}

func (m *mockPayrollPeriodRepository) FindByOrgYearMonth(ctx context.Context, orgID uuid.UUID, year, month int) (*domain.PayrollPeriod, error) {
	if m.findByOrgYearMonthFunc != nil {
		return m.findByOrgYearMonthFunc(ctx, orgID, year, month)
	}
	return nil, sqlite.ErrRecordNotFound
}

func (m *mockPayrollPeriodRepository) FindByOrganization(ctx context.Context, orgID uuid.UUID) ([]*domain.PayrollPeriod, error) {
	if m.findByOrganizationFunc != nil {
		return m.findByOrganizationFunc(ctx, orgID)
	}
	return nil, nil
}

func (m *mockPayrollPeriodRepository) FindByOrganizationIncludingDeleted(ctx context.Context, orgID uuid.UUID) ([]*domain.PayrollPeriod, error) {
	if m.findByOrganizationIncludingDeletedFunc != nil {
		return m.findByOrganizationIncludingDeletedFunc(ctx, orgID)
	}
	return nil, nil
}

func (m *mockPayrollPeriodRepository) FindAllDraft(ctx context.Context) ([]*domain.PayrollPeriod, error) {
	if m.findAllDraftFunc != nil {
		return m.findAllDraftFunc(ctx)
	}
	return nil, nil
}

func (m *mockPayrollPeriodRepository) FindAll(ctx context.Context) ([]*domain.PayrollPeriod, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc(ctx)
	}
	return nil, nil
}

func (m *mockPayrollPeriodRepository) FindAllIncludingDeleted(ctx context.Context) ([]*domain.PayrollPeriod, error) {
	if m.findAllIncludingDeletedFunc != nil {
		return m.findAllIncludingDeletedFunc(ctx)
	}
	return nil, nil
}

// mockPayrollResultRepository implements payrollResultRepository interface for testing
type mockPayrollResultRepository struct {
	createFunc                         func(context.Context, *domain.PayrollResult) error
	deleteFunc                         func(context.Context, uuid.UUID) error
	restoreFunc                        func(context.Context, uuid.UUID) error
	hardDeleteFunc                     func(context.Context, uuid.UUID) error
	findByIDFunc                       func(context.Context, uuid.UUID) (*domain.PayrollResult, error)
	findByIDIncludingDeletedFunc       func(context.Context, uuid.UUID) (*domain.PayrollResult, error)
	findByPeriodFunc                   func(context.Context, uuid.UUID) ([]*domain.PayrollResult, error)
	findByPeriodIncludingDeletedFunc   func(context.Context, uuid.UUID) ([]*domain.PayrollResult, error)
	findByEmployeeFunc                 func(context.Context, uuid.UUID) ([]*domain.PayrollResult, error)
	findByEmployeeIncludingDeletedFunc func(context.Context, uuid.UUID) ([]*domain.PayrollResult, error)
	findAllFunc                        func(context.Context) ([]*domain.PayrollResult, error)
	findAllIncludingDeletedFunc        func(context.Context) ([]*domain.PayrollResult, error)
	replaceAllForPeriodFunc            func(context.Context, uuid.UUID, []*domain.PayrollResult) error
}

func (m *mockPayrollResultRepository) Create(ctx context.Context, result *domain.PayrollResult) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, result)
	}
	return nil
}

func (m *mockPayrollResultRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockPayrollResultRepository) Restore(ctx context.Context, id uuid.UUID) error {
	if m.restoreFunc != nil {
		return m.restoreFunc(ctx, id)
	}
	return nil
}

func (m *mockPayrollResultRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	if m.hardDeleteFunc != nil {
		return m.hardDeleteFunc(ctx, id)
	}
	return nil
}

func (m *mockPayrollResultRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.PayrollResult, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, sqlite.ErrRecordNotFound
}

func (m *mockPayrollResultRepository) FindByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*domain.PayrollResult, error) {
	if m.findByIDIncludingDeletedFunc != nil {
		return m.findByIDIncludingDeletedFunc(ctx, id)
	}
	return nil, sqlite.ErrRecordNotFound
}

func (m *mockPayrollResultRepository) FindByPeriod(ctx context.Context, periodID uuid.UUID) ([]*domain.PayrollResult, error) {
	if m.findByPeriodFunc != nil {
		return m.findByPeriodFunc(ctx, periodID)
	}
	return nil, nil
}

func (m *mockPayrollResultRepository) FindByPeriodIncludingDeleted(ctx context.Context, periodID uuid.UUID) ([]*domain.PayrollResult, error) {
	if m.findByPeriodIncludingDeletedFunc != nil {
		return m.findByPeriodIncludingDeletedFunc(ctx, periodID)
	}
	return nil, nil
}

func (m *mockPayrollResultRepository) FindByEmployee(ctx context.Context, employeeID uuid.UUID) ([]*domain.PayrollResult, error) {
	if m.findByEmployeeFunc != nil {
		return m.findByEmployeeFunc(ctx, employeeID)
	}
	return nil, nil
}

func (m *mockPayrollResultRepository) FindByEmployeeIncludingDeleted(ctx context.Context, employeeID uuid.UUID) ([]*domain.PayrollResult, error) {
	if m.findByEmployeeIncludingDeletedFunc != nil {
		return m.findByEmployeeIncludingDeletedFunc(ctx, employeeID)
	}
	return nil, nil
}

func (m *mockPayrollResultRepository) FindAll(ctx context.Context) ([]*domain.PayrollResult, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc(ctx)
	}
	return nil, nil
}

func (m *mockPayrollResultRepository) FindAllIncludingDeleted(ctx context.Context) ([]*domain.PayrollResult, error) {
	if m.findAllIncludingDeletedFunc != nil {
		return m.findAllIncludingDeletedFunc(ctx)
	}
	return nil, nil
}

func (m *mockPayrollResultRepository) ReplaceAllForPeriod(ctx context.Context, periodID uuid.UUID, results []*domain.PayrollResult) error {
	if m.replaceAllForPeriodFunc != nil {
		return m.replaceAllForPeriodFunc(ctx, periodID, results)
	}
	return nil
}

// mockPayrollCalculator implements payrollCalculator interface for testing
type mockPayrollCalculator struct {
	calculateFunc func(context.Context, *domain.PayrollPeriod, *domain.Employee, *domain.EmployeeCompensationPackage) (*domain.PayrollResult, error)
}

func (m *mockPayrollCalculator) Calculate(
	ctx context.Context,
	period *domain.PayrollPeriod,
	emp *domain.Employee,
	pkg *domain.EmployeeCompensationPackage,
) (*domain.PayrollResult, error) {
	if m.calculateFunc != nil {
		return m.calculateFunc(ctx, period, emp, pkg)
	}
	// Default: return a minimal valid result
	now := time.Now().UTC()
	return &domain.PayrollResult{
		ID:                    uuid.New(),
		PayrollPeriodID:       period.ID,
		EmployeeID:            emp.ID,
		CompensationPackageID: pkg.ID,
		Currency:              pkg.Currency,
		CreatedAt:             now,
		UpdatedAt:             now,
	}, nil
}

// ===============================================================================
// TEST HELPERS
// ===============================================================================

func createTestPayrollPeriod(orgID uuid.UUID, year, month int) *domain.PayrollPeriod {
	now := time.Now().UTC()
	return &domain.PayrollPeriod{
		ID:        uuid.New(),
		OrgID:     orgID,
		Year:      year,
		Month:     month,
		Status:    domain.PayrollPeriodStatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func createTestPayrollResult(periodID, employeeID, compPackID uuid.UUID) *domain.PayrollResult {
	now := time.Now().UTC()
	return &domain.PayrollResult{
		ID:                    uuid.New(),
		PayrollPeriodID:       periodID,
		EmployeeID:            employeeID,
		CompensationPackageID: compPackID,
		Currency:              money.MAD,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
}

// ===============================================================================
// PAYROLL PERIOD - CREATE TESTS
// ===============================================================================

func TestPayrollService_CreatePayrollPeriod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMocks func(*mockPayrollPeriodRepository)
		period     *domain.PayrollPeriod
		wantErr    error
		validateFn func(*testing.T, *domain.PayrollPeriod)
	}{
		{
			name: "creates period successfully",
			setupMocks: func(m *mockPayrollPeriodRepository) {
				m.createFunc = func(ctx context.Context, p *domain.PayrollPeriod) error {
					return nil
				}
			},
			period: &domain.PayrollPeriod{
				OrgID: uuid.New(),
				Year:  2025,
				Month: 1,
			},
			wantErr: nil,
			validateFn: func(t *testing.T, p *domain.PayrollPeriod) {
				assert.NotEqual(t, uuid.Nil, p.ID, "should generate UUID")
				assert.Equal(t, domain.PayrollPeriodStatusDraft, p.Status, "should set status to DRAFT")
				assert.Nil(t, p.FinalizedAt, "should not set finalized_at for DRAFT")
				assert.False(t, p.CreatedAt.IsZero(), "should set created_at")
				assert.False(t, p.UpdatedAt.IsZero(), "should set updated_at")
			},
		},
		{
			name: "preserves provided UUID",
			setupMocks: func(m *mockPayrollPeriodRepository) {
				m.createFunc = func(ctx context.Context, p *domain.PayrollPeriod) error {
					return nil
				}
			},
			period: &domain.PayrollPeriod{
				ID:    uuid.New(),
				OrgID: uuid.New(),
				Year:  2025,
				Month: 1,
			},
			wantErr: nil,
			validateFn: func(t *testing.T, p *domain.PayrollPeriod) {
				assert.NotEqual(t, uuid.Nil, p.ID, "should preserve UUID")
			},
		},
		{
			name: "returns error if period already exists",
			setupMocks: func(m *mockPayrollPeriodRepository) {
				m.createFunc = func(ctx context.Context, p *domain.PayrollPeriod) error {
					return sqlite.ErrDuplicateRecord
				}
			},
			period: &domain.PayrollPeriod{
				OrgID: uuid.New(),
				Year:  2025,
				Month: 1,
			},
			wantErr: application.ErrPayrollPeriodExists,
		},
		{
			name: "returns error for invalid period",
			setupMocks: func(m *mockPayrollPeriodRepository) {
				m.createFunc = func(ctx context.Context, p *domain.PayrollPeriod) error {
					return nil
				}
			},
			period: &domain.PayrollPeriod{
				OrgID: uuid.New(),
				Year:  1999, // Invalid year
				Month: 1,
			},
			wantErr: domain.ErrInvalidPayrollPeriodYear,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockPeriods := &mockPayrollPeriodRepository{}
			if tt.setupMocks != nil {
				tt.setupMocks(mockPeriods)
			}

			service := application.NewPayrollService(
				mockPeriods,
				&mockPayrollResultRepository{},
				&mockEmployeeRepository{},
				&mockCompensationPackageRepository{},
				&mockPayrollCalculator{},
			)

			err := service.CreatePayrollPeriod(context.Background(), tt.period)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				if tt.validateFn != nil {
					tt.validateFn(t, tt.period)
				}
			}
		})
	}
}

// ===============================================================================
// PAYROLL PERIOD - DELETE TESTS
// ===============================================================================

func TestPayrollService_DeletePayrollPeriod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMocks func(*mockPayrollPeriodRepository)
		periodID   uuid.UUID
		wantErr    error
	}{
		{
			name: "deletes draft period successfully",
			setupMocks: func(m *mockPayrollPeriodRepository) {
				m.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
					return &domain.PayrollPeriod{
						ID:     id,
						OrgID:  uuid.New(),
						Year:   2025,
						Month:  1,
						Status: domain.PayrollPeriodStatusDraft,
					}, nil
				}
				m.deleteFunc = func(ctx context.Context, id uuid.UUID) error {
					return nil
				}
			},
			periodID: uuid.New(),
			wantErr:  nil,
		},
		{
			name: "cannot delete finalized period",
			setupMocks: func(m *mockPayrollPeriodRepository) {
				m.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
					now := time.Now().UTC()
					return &domain.PayrollPeriod{
						ID:          id,
						OrgID:       uuid.New(),
						Year:        2025,
						Month:       1,
						Status:      domain.PayrollPeriodStatusFinalized,
						FinalizedAt: &now,
					}, nil
				}
			},
			periodID: uuid.New(),
			wantErr:  application.ErrPayrollPeriodAlreadyFinalized,
		},
		{
			name: "returns error if period not found",
			setupMocks: func(m *mockPayrollPeriodRepository) {
				m.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
					return nil, sqlite.ErrRecordNotFound
				}
			},
			periodID: uuid.New(),
			wantErr:  application.ErrPayrollPeriodNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockPeriods := &mockPayrollPeriodRepository{}
			if tt.setupMocks != nil {
				tt.setupMocks(mockPeriods)
			}

			service := application.NewPayrollService(
				mockPeriods,
				&mockPayrollResultRepository{},
				&mockEmployeeRepository{},
				&mockCompensationPackageRepository{},
				&mockPayrollCalculator{},
			)

			err := service.DeletePayrollPeriod(context.Background(), tt.periodID)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPayrollService_RestorePayrollPeriod(t *testing.T) {
	t.Parallel()

	t.Run("restores period successfully", func(t *testing.T) {
		t.Parallel()

		mockPeriods := &mockPayrollPeriodRepository{
			restoreFunc: func(ctx context.Context, id uuid.UUID) error {
				return nil
			},
		}

		service := application.NewPayrollService(
			mockPeriods,
			&mockPayrollResultRepository{},
			&mockEmployeeRepository{},
			&mockCompensationPackageRepository{},
			&mockPayrollCalculator{},
		)

		err := service.RestorePayrollPeriod(context.Background(), uuid.New())
		require.NoError(t, err)
	})

	t.Run("returns error if period not found", func(t *testing.T) {
		t.Parallel()

		mockPeriods := &mockPayrollPeriodRepository{
			restoreFunc: func(ctx context.Context, id uuid.UUID) error {
				return sqlite.ErrRecordNotFound
			},
		}

		service := application.NewPayrollService(
			mockPeriods,
			&mockPayrollResultRepository{},
			&mockEmployeeRepository{},
			&mockCompensationPackageRepository{},
			&mockPayrollCalculator{},
		)

		err := service.RestorePayrollPeriod(context.Background(), uuid.New())
		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrPayrollPeriodNotFound)
	})
}

func TestPayrollService_HardDeletePayrollPeriod(t *testing.T) {
	t.Parallel()

	t.Run("hard deletes period successfully", func(t *testing.T) {
		t.Parallel()

		mockPeriods := &mockPayrollPeriodRepository{
			hardDeleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return nil
			},
		}

		service := application.NewPayrollService(
			mockPeriods,
			&mockPayrollResultRepository{},
			&mockEmployeeRepository{},
			&mockCompensationPackageRepository{},
			&mockPayrollCalculator{},
		)

		err := service.HardDeletePayrollPeriod(context.Background(), uuid.New())
		require.NoError(t, err)
	})

	t.Run("returns error if period not found", func(t *testing.T) {
		t.Parallel()

		mockPeriods := &mockPayrollPeriodRepository{
			hardDeleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return sqlite.ErrRecordNotFound
			},
		}

		service := application.NewPayrollService(
			mockPeriods,
			&mockPayrollResultRepository{},
			&mockEmployeeRepository{},
			&mockCompensationPackageRepository{},
			&mockPayrollCalculator{},
		)

		err := service.HardDeletePayrollPeriod(context.Background(), uuid.New())
		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrPayrollPeriodNotFound)
	})
}

// ===============================================================================
// PAYROLL PERIOD - WORKFLOW TESTS
// ===============================================================================

func TestPayrollService_FinalizePayrollPeriod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMocks func(*mockPayrollPeriodRepository, *mockPayrollResultRepository)
		periodID   uuid.UUID
		wantErr    error
	}{
		{
			name: "finalizes period successfully",
			setupMocks: func(mp *mockPayrollPeriodRepository, mr *mockPayrollResultRepository) {
				mp.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
					return &domain.PayrollPeriod{
						ID:     id,
						OrgID:  uuid.New(),
						Year:   2025,
						Month:  1,
						Status: domain.PayrollPeriodStatusDraft,
					}, nil
				}
				mr.findByPeriodFunc = func(ctx context.Context, periodID uuid.UUID) ([]*domain.PayrollResult, error) {
					return []*domain.PayrollResult{
						{ID: uuid.New()},
					}, nil
				}
				mp.finalizeFunc = func(ctx context.Context, id uuid.UUID) error {
					return nil
				}
			},
			periodID: uuid.New(),
			wantErr:  nil,
		},
		{
			name: "cannot finalize already finalized period",
			setupMocks: func(mp *mockPayrollPeriodRepository, mr *mockPayrollResultRepository) {
				now := time.Now().UTC()
				mp.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
					return &domain.PayrollPeriod{
						ID:          id,
						OrgID:       uuid.New(),
						Year:        2025,
						Month:       1,
						Status:      domain.PayrollPeriodStatusFinalized,
						FinalizedAt: &now,
					}, nil
				}
			},
			periodID: uuid.New(),
			wantErr:  application.ErrPayrollPeriodAlreadyFinalized,
		},
		{
			name: "cannot finalize empty period",
			setupMocks: func(mp *mockPayrollPeriodRepository, mr *mockPayrollResultRepository) {
				mp.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
					return &domain.PayrollPeriod{
						ID:     id,
						OrgID:  uuid.New(),
						Year:   2025,
						Month:  1,
						Status: domain.PayrollPeriodStatusDraft,
					}, nil
				}
				mr.findByPeriodFunc = func(ctx context.Context, periodID uuid.UUID) ([]*domain.PayrollResult, error) {
					return []*domain.PayrollResult{}, nil
				}
			},
			periodID: uuid.New(),
			wantErr:  application.ErrPayrollPeriodEmpty,
		},
		{
			name: "returns error if period not found",
			setupMocks: func(mp *mockPayrollPeriodRepository, mr *mockPayrollResultRepository) {
				mp.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
					return nil, sqlite.ErrRecordNotFound
				}
			},
			periodID: uuid.New(),
			wantErr:  application.ErrPayrollPeriodNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockPeriods := &mockPayrollPeriodRepository{}
			mockResults := &mockPayrollResultRepository{}
			if tt.setupMocks != nil {
				tt.setupMocks(mockPeriods, mockResults)
			}

			service := application.NewPayrollService(
				mockPeriods,
				mockResults,
				&mockEmployeeRepository{},
				&mockCompensationPackageRepository{},
				&mockPayrollCalculator{},
			)

			err := service.FinalizePayrollPeriod(context.Background(), tt.periodID)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPayrollService_UnfinalizePayrollPeriod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMocks func(*mockPayrollPeriodRepository)
		periodID   uuid.UUID
		wantErr    error
	}{
		{
			name: "unfinalizes period successfully",
			setupMocks: func(m *mockPayrollPeriodRepository) {
				now := time.Now().UTC()
				m.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
					return &domain.PayrollPeriod{
						ID:          id,
						OrgID:       uuid.New(),
						Year:        2025,
						Month:       1,
						Status:      domain.PayrollPeriodStatusFinalized,
						FinalizedAt: &now,
					}, nil
				}
				m.unfinalizeFunc = func(ctx context.Context, id uuid.UUID) error {
					return nil
				}
			},
			periodID: uuid.New(),
			wantErr:  nil,
		},
		{
			name: "cannot unfinalize draft period",
			setupMocks: func(m *mockPayrollPeriodRepository) {
				m.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
					return &domain.PayrollPeriod{
						ID:     id,
						OrgID:  uuid.New(),
						Year:   2025,
						Month:  1,
						Status: domain.PayrollPeriodStatusDraft,
					}, nil
				}
			},
			periodID: uuid.New(),
			wantErr:  application.ErrPayrollPeriodNotFinalized,
		},
		{
			name: "returns error if period not found",
			setupMocks: func(m *mockPayrollPeriodRepository) {
				m.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
					return nil, sqlite.ErrRecordNotFound
				}
			},
			periodID: uuid.New(),
			wantErr:  application.ErrPayrollPeriodNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockPeriods := &mockPayrollPeriodRepository{}
			if tt.setupMocks != nil {
				tt.setupMocks(mockPeriods)
			}

			service := application.NewPayrollService(
				mockPeriods,
				&mockPayrollResultRepository{},
				&mockEmployeeRepository{},
				&mockCompensationPackageRepository{},
				&mockPayrollCalculator{},
			)

			err := service.UnfinalizePayrollPeriod(context.Background(), tt.periodID)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ===============================================================================
// PAYROLL PERIOD - QUERY TESTS
// ===============================================================================

func TestPayrollService_GetPayrollPeriod(t *testing.T) {
	t.Parallel()

	t.Run("gets period successfully", func(t *testing.T) {
		t.Parallel()

		expectedPeriod := createTestPayrollPeriod(uuid.New(), 2025, 1)

		mockPeriods := &mockPayrollPeriodRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
				return expectedPeriod, nil
			},
		}

		service := application.NewPayrollService(
			mockPeriods,
			&mockPayrollResultRepository{},
			&mockEmployeeRepository{},
			&mockCompensationPackageRepository{},
			&mockPayrollCalculator{},
		)

		period, err := service.GetPayrollPeriod(context.Background(), expectedPeriod.ID)
		require.NoError(t, err)
		assert.Equal(t, expectedPeriod.ID, period.ID)
	})

	t.Run("returns error if period not found", func(t *testing.T) {
		t.Parallel()

		mockPeriods := &mockPayrollPeriodRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
				return nil, sqlite.ErrRecordNotFound
			},
		}

		service := application.NewPayrollService(
			mockPeriods,
			&mockPayrollResultRepository{},
			&mockEmployeeRepository{},
			&mockCompensationPackageRepository{},
			&mockPayrollCalculator{},
		)

		_, err := service.GetPayrollPeriod(context.Background(), uuid.New())
		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrPayrollPeriodNotFound)
	})
}

func TestPayrollService_GetPayrollPeriodIncludingDeleted(t *testing.T) {
	t.Parallel()

	t.Run("gets deleted period successfully", func(t *testing.T) {
		t.Parallel()

		now := time.Now().UTC()
		expectedPeriod := createTestPayrollPeriod(uuid.New(), 2025, 1)
		expectedPeriod.DeletedAt = &now

		mockPeriods := &mockPayrollPeriodRepository{
			findByIDIncludingDeletedFunc: func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
				return expectedPeriod, nil
			},
		}

		service := application.NewPayrollService(
			mockPeriods,
			&mockPayrollResultRepository{},
			&mockEmployeeRepository{},
			&mockCompensationPackageRepository{},
			&mockPayrollCalculator{},
		)

		period, err := service.GetPayrollPeriodIncludingDeleted(context.Background(), expectedPeriod.ID)
		require.NoError(t, err)
		assert.Equal(t, expectedPeriod.ID, period.ID)
		assert.NotNil(t, period.DeletedAt)
	})
}

func TestPayrollService_GetPayrollPeriodByOrgYearMonth(t *testing.T) {
	t.Parallel()

	t.Run("gets period by org/year/month successfully", func(t *testing.T) {
		t.Parallel()

		orgID := uuid.New()
		expectedPeriod := createTestPayrollPeriod(orgID, 2025, 1)

		mockPeriods := &mockPayrollPeriodRepository{
			findByOrgYearMonthFunc: func(ctx context.Context, id uuid.UUID, year, month int) (*domain.PayrollPeriod, error) {
				if id == orgID && year == 2025 && month == 1 {
					return expectedPeriod, nil
				}
				return nil, sqlite.ErrRecordNotFound
			},
		}

		service := application.NewPayrollService(
			mockPeriods,
			&mockPayrollResultRepository{},
			&mockEmployeeRepository{},
			&mockCompensationPackageRepository{},
			&mockPayrollCalculator{},
		)

		period, err := service.GetPayrollPeriodByOrgYearMonth(context.Background(), orgID, 2025, 1)
		require.NoError(t, err)
		assert.Equal(t, expectedPeriod.ID, period.ID)
		assert.Equal(t, 2025, period.Year)
		assert.Equal(t, 1, period.Month)
	})

	t.Run("returns error if period not found", func(t *testing.T) {
		t.Parallel()

		mockPeriods := &mockPayrollPeriodRepository{
			findByOrgYearMonthFunc: func(ctx context.Context, id uuid.UUID, year, month int) (*domain.PayrollPeriod, error) {
				return nil, sqlite.ErrRecordNotFound
			},
		}

		service := application.NewPayrollService(
			mockPeriods,
			&mockPayrollResultRepository{},
			&mockEmployeeRepository{},
			&mockCompensationPackageRepository{},
			&mockPayrollCalculator{},
		)

		_, err := service.GetPayrollPeriodByOrgYearMonth(context.Background(), uuid.New(), 2025, 1)
		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrPayrollPeriodNotFound)
	})
}

func TestPayrollService_ListPayrollPeriodsByOrganization(t *testing.T) {
	t.Parallel()

	t.Run("lists periods successfully", func(t *testing.T) {
		t.Parallel()

		orgID := uuid.New()
		expectedPeriods := []*domain.PayrollPeriod{
			createTestPayrollPeriod(orgID, 2025, 1),
			createTestPayrollPeriod(orgID, 2025, 2),
		}

		mockPeriods := &mockPayrollPeriodRepository{
			findByOrganizationFunc: func(ctx context.Context, id uuid.UUID) ([]*domain.PayrollPeriod, error) {
				if id == orgID {
					return expectedPeriods, nil
				}
				return nil, nil
			},
		}

		service := application.NewPayrollService(
			mockPeriods,
			&mockPayrollResultRepository{},
			&mockEmployeeRepository{},
			&mockCompensationPackageRepository{},
			&mockPayrollCalculator{},
		)

		periods, err := service.ListPayrollPeriodsByOrganization(context.Background(), orgID)
		require.NoError(t, err)
		assert.Len(t, periods, 2)
	})

	t.Run("returns empty slice if no periods found", func(t *testing.T) {
		t.Parallel()

		mockPeriods := &mockPayrollPeriodRepository{
			findByOrganizationFunc: func(ctx context.Context, id uuid.UUID) ([]*domain.PayrollPeriod, error) {
				return []*domain.PayrollPeriod{}, nil
			},
		}

		service := application.NewPayrollService(
			mockPeriods,
			&mockPayrollResultRepository{},
			&mockEmployeeRepository{},
			&mockCompensationPackageRepository{},
			&mockPayrollCalculator{},
		)

		periods, err := service.ListPayrollPeriodsByOrganization(context.Background(), uuid.New())
		require.NoError(t, err)
		assert.Empty(t, periods)
	})
}

func TestPayrollService_ListDraftPayrollPeriods(t *testing.T) {
	t.Parallel()

	t.Run("lists draft periods successfully", func(t *testing.T) {
		t.Parallel()

		expectedPeriods := []*domain.PayrollPeriod{
			createTestPayrollPeriod(uuid.New(), 2025, 1),
			createTestPayrollPeriod(uuid.New(), 2025, 2),
		}

		mockPeriods := &mockPayrollPeriodRepository{
			findAllDraftFunc: func(ctx context.Context) ([]*domain.PayrollPeriod, error) {
				return expectedPeriods, nil
			},
		}

		service := application.NewPayrollService(
			mockPeriods,
			&mockPayrollResultRepository{},
			&mockEmployeeRepository{},
			&mockCompensationPackageRepository{},
			&mockPayrollCalculator{},
		)

		periods, err := service.ListDraftPayrollPeriods(context.Background())
		require.NoError(t, err)
		assert.Len(t, periods, 2)
		for _, p := range periods {
			assert.Equal(t, domain.PayrollPeriodStatusDraft, p.Status)
		}
	})
}

// ===============================================================================
// PAYROLL RESULT - GENERATION TESTS
// ===============================================================================

func TestPayrollService_GeneratePayrollResults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMocks func(*mockPayrollPeriodRepository, *mockPayrollResultRepository, *mockEmployeeRepository, *mockCompensationPackageRepository)
		periodID   uuid.UUID
		wantErr    error
	}{
		{
			name: "generates results for all employees successfully",
			setupMocks: func(mp *mockPayrollPeriodRepository, mr *mockPayrollResultRepository, me *mockEmployeeRepository, mc *mockCompensationPackageRepository) {
				orgID := uuid.New()
				compPackID := uuid.New()

				mp.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
					return &domain.PayrollPeriod{
						ID:     id,
						OrgID:  orgID,
						Year:   2025,
						Month:  1,
						Status: domain.PayrollPeriodStatusDraft,
					}, nil
				}

				me.findByOrganizationFunc = func(ctx context.Context, id uuid.UUID) ([]*domain.Employee, error) {
					return []*domain.Employee{
						createTestEmployee(orgID, compPackID),
						createTestEmployee(orgID, compPackID),
					}, nil
				}

				mc.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.EmployeeCompensationPackage, error) {
					return createTestCompensationPackage(), nil
				}

				mr.replaceAllForPeriodFunc = func(ctx context.Context, periodID uuid.UUID, results []*domain.PayrollResult) error {
					assert.Len(t, results, 2, "should pass all calculated results")
					return nil
				}
			},
			periodID: uuid.New(),
			wantErr:  nil,
		},
		{
			name: "cannot generate for finalized period",
			setupMocks: func(mp *mockPayrollPeriodRepository, mr *mockPayrollResultRepository, me *mockEmployeeRepository, mc *mockCompensationPackageRepository) {
				now := time.Now().UTC()
				mp.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
					return &domain.PayrollPeriod{
						ID:          id,
						OrgID:       uuid.New(),
						Year:        2025,
						Month:       1,
						Status:      domain.PayrollPeriodStatusFinalized,
						FinalizedAt: &now,
					}, nil
				}
			},
			periodID: uuid.New(),
			wantErr:  application.ErrPayrollPeriodAlreadyFinalized,
		},
		{
			name: "replaces existing results atomically when regenerating",
			setupMocks: func(mp *mockPayrollPeriodRepository, mr *mockPayrollResultRepository, me *mockEmployeeRepository, mc *mockCompensationPackageRepository) {
				orgID := uuid.New()
				compPackID := uuid.New()

				mp.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
					return &domain.PayrollPeriod{
						ID:     id,
						OrgID:  orgID,
						Year:   2025,
						Month:  1,
						Status: domain.PayrollPeriodStatusDraft,
					}, nil
				}

				me.findByOrganizationFunc = func(ctx context.Context, id uuid.UUID) ([]*domain.Employee, error) {
					return []*domain.Employee{
						createTestEmployee(orgID, compPackID),
					}, nil
				}

				mc.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.EmployeeCompensationPackage, error) {
					return createTestCompensationPackage(), nil
				}

				mr.replaceAllForPeriodFunc = func(ctx context.Context, periodID uuid.UUID, results []*domain.PayrollResult) error {
					// The repository handles delete+create atomically; the service
					// must pass the full set of calculated results in one call.
					assert.Len(t, results, 1, "should pass one result per employee")
					return nil
				}
			},
			periodID: uuid.New(),
			wantErr:  nil,
		},
		{
			name: "returns error if period not found",
			setupMocks: func(mp *mockPayrollPeriodRepository, mr *mockPayrollResultRepository, me *mockEmployeeRepository, mc *mockCompensationPackageRepository) {
				mp.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
					return nil, sqlite.ErrRecordNotFound
				}
			},
			periodID: uuid.New(),
			wantErr:  application.ErrPayrollPeriodNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockPeriods := &mockPayrollPeriodRepository{}
			mockResults := &mockPayrollResultRepository{}
			mockEmployees := &mockEmployeeRepository{}
			mockCompPackages := &mockCompensationPackageRepository{}

			if tt.setupMocks != nil {
				tt.setupMocks(mockPeriods, mockResults, mockEmployees, mockCompPackages)
			}

			service := application.NewPayrollService(
				mockPeriods,
				mockResults,
				mockEmployees,
				mockCompPackages,
				&mockPayrollCalculator{},
			)

			err := service.GeneratePayrollResults(context.Background(), tt.periodID)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPayrollService_DeletePayrollResults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMocks func(*mockPayrollPeriodRepository, *mockPayrollResultRepository)
		periodID   uuid.UUID
		wantErr    error
	}{
		{
			name: "deletes all results for period successfully",
			setupMocks: func(mp *mockPayrollPeriodRepository, mr *mockPayrollResultRepository) {
				mp.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
					return &domain.PayrollPeriod{
						ID:     id,
						OrgID:  uuid.New(),
						Year:   2025,
						Month:  1,
						Status: domain.PayrollPeriodStatusDraft,
					}, nil
				}

				mr.findByPeriodFunc = func(ctx context.Context, periodID uuid.UUID) ([]*domain.PayrollResult, error) {
					return []*domain.PayrollResult{
						{ID: uuid.New()},
						{ID: uuid.New()},
					}, nil
				}

				mr.deleteFunc = func(ctx context.Context, id uuid.UUID) error {
					return nil
				}
			},
			periodID: uuid.New(),
			wantErr:  nil,
		},
		{
			name: "cannot delete results from finalized period",
			setupMocks: func(mp *mockPayrollPeriodRepository, mr *mockPayrollResultRepository) {
				now := time.Now().UTC()
				mp.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
					return &domain.PayrollPeriod{
						ID:          id,
						OrgID:       uuid.New(),
						Year:        2025,
						Month:       1,
						Status:      domain.PayrollPeriodStatusFinalized,
						FinalizedAt: &now,
					}, nil
				}
			},
			periodID: uuid.New(),
			wantErr:  application.ErrPayrollPeriodAlreadyFinalized,
		},
		{
			name: "returns error if period not found",
			setupMocks: func(mp *mockPayrollPeriodRepository, mr *mockPayrollResultRepository) {
				mp.findByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
					return nil, sqlite.ErrRecordNotFound
				}
			},
			periodID: uuid.New(),
			wantErr:  application.ErrPayrollPeriodNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockPeriods := &mockPayrollPeriodRepository{}
			mockResults := &mockPayrollResultRepository{}

			if tt.setupMocks != nil {
				tt.setupMocks(mockPeriods, mockResults)
			}

			service := application.NewPayrollService(
				mockPeriods,
				mockResults,
				&mockEmployeeRepository{},
				&mockCompensationPackageRepository{},
				&mockPayrollCalculator{},
			)

			err := service.DeletePayrollResults(context.Background(), tt.periodID)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ===============================================================================
// PAYROLL RESULT - QUERY TESTS
// ===============================================================================

func TestPayrollService_GetPayrollResult(t *testing.T) {
	t.Parallel()

	t.Run("gets result successfully", func(t *testing.T) {
		t.Parallel()

		expectedResult := createTestPayrollResult(uuid.New(), uuid.New(), uuid.New())

		mockResults := &mockPayrollResultRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.PayrollResult, error) {
				return expectedResult, nil
			},
		}

		service := application.NewPayrollService(
			&mockPayrollPeriodRepository{},
			mockResults,
			&mockEmployeeRepository{},
			&mockCompensationPackageRepository{},
			&mockPayrollCalculator{},
		)

		result, err := service.GetPayrollResult(context.Background(), expectedResult.ID)
		require.NoError(t, err)
		assert.Equal(t, expectedResult.ID, result.ID)
	})

	t.Run("returns error if result not found", func(t *testing.T) {
		t.Parallel()

		mockResults := &mockPayrollResultRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.PayrollResult, error) {
				return nil, sqlite.ErrRecordNotFound
			},
		}

		service := application.NewPayrollService(
			&mockPayrollPeriodRepository{},
			mockResults,
			&mockEmployeeRepository{},
			&mockCompensationPackageRepository{},
			&mockPayrollCalculator{},
		)

		_, err := service.GetPayrollResult(context.Background(), uuid.New())
		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrPayrollResultNotFound)
	})
}

func TestPayrollService_ListPayrollResultsByPeriod(t *testing.T) {
	t.Parallel()

	t.Run("lists results by period successfully", func(t *testing.T) {
		t.Parallel()

		periodID := uuid.New()
		expectedResults := []*domain.PayrollResult{
			createTestPayrollResult(periodID, uuid.New(), uuid.New()),
			createTestPayrollResult(periodID, uuid.New(), uuid.New()),
		}

		mockResults := &mockPayrollResultRepository{
			findByPeriodFunc: func(ctx context.Context, id uuid.UUID) ([]*domain.PayrollResult, error) {
				if id == periodID {
					return expectedResults, nil
				}
				return nil, nil
			},
		}

		service := application.NewPayrollService(
			&mockPayrollPeriodRepository{},
			mockResults,
			&mockEmployeeRepository{},
			&mockCompensationPackageRepository{},
			&mockPayrollCalculator{},
		)

		results, err := service.ListPayrollResultsByPeriod(context.Background(), periodID)
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("returns empty slice if no results found", func(t *testing.T) {
		t.Parallel()

		mockResults := &mockPayrollResultRepository{
			findByPeriodFunc: func(ctx context.Context, id uuid.UUID) ([]*domain.PayrollResult, error) {
				return []*domain.PayrollResult{}, nil
			},
		}

		service := application.NewPayrollService(
			&mockPayrollPeriodRepository{},
			mockResults,
			&mockEmployeeRepository{},
			&mockCompensationPackageRepository{},
			&mockPayrollCalculator{},
		)

		results, err := service.ListPayrollResultsByPeriod(context.Background(), uuid.New())
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestPayrollService_ListPayrollResultsByEmployee(t *testing.T) {
	t.Parallel()

	t.Run("lists results by employee successfully", func(t *testing.T) {
		t.Parallel()

		employeeID := uuid.New()
		expectedResults := []*domain.PayrollResult{
			createTestPayrollResult(uuid.New(), employeeID, uuid.New()),
			createTestPayrollResult(uuid.New(), employeeID, uuid.New()),
		}

		mockResults := &mockPayrollResultRepository{
			findByEmployeeFunc: func(ctx context.Context, id uuid.UUID) ([]*domain.PayrollResult, error) {
				if id == employeeID {
					return expectedResults, nil
				}
				return nil, nil
			},
		}

		service := application.NewPayrollService(
			&mockPayrollPeriodRepository{},
			mockResults,
			&mockEmployeeRepository{},
			&mockCompensationPackageRepository{},
			&mockPayrollCalculator{},
		)

		results, err := service.ListPayrollResultsByEmployee(context.Background(), employeeID)
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("returns empty slice if employee has no results", func(t *testing.T) {
		t.Parallel()

		mockResults := &mockPayrollResultRepository{
			findByEmployeeFunc: func(ctx context.Context, id uuid.UUID) ([]*domain.PayrollResult, error) {
				return []*domain.PayrollResult{}, nil
			},
		}

		service := application.NewPayrollService(
			&mockPayrollPeriodRepository{},
			mockResults,
			&mockEmployeeRepository{},
			&mockCompensationPackageRepository{},
			&mockPayrollCalculator{},
		)

		results, err := service.ListPayrollResultsByEmployee(context.Background(), uuid.New())
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}
