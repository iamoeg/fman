package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/iamoeg/bootdev-capstone/internal/application"
	"github.com/iamoeg/bootdev-capstone/internal/domain"
	"github.com/iamoeg/bootdev-capstone/pkg/money"

	sqlite "github.com/iamoeg/bootdev-capstone/internal/adapter/sqlite"
)

// ===============================================================================
// MOCK REPOSITORY
// ===============================================================================

// mockCompensationPackageRepository is a mock implementation of
// compensationPackageRepository for testing.
type mockCompensationPackageRepository struct {
	// Function implementations - set these in tests
	createFunc                   func(context.Context, *domain.EmployeeCompensationPackage) error
	updateFunc                   func(context.Context, *domain.EmployeeCompensationPackage) error
	deleteFunc                   func(context.Context, uuid.UUID) error
	restoreFunc                  func(context.Context, uuid.UUID) error
	hardDeleteFunc               func(context.Context, uuid.UUID) error
	findByIDFunc                 func(context.Context, uuid.UUID) (*domain.EmployeeCompensationPackage, error)
	findByIDIncludingDeletedFunc func(context.Context, uuid.UUID) (*domain.EmployeeCompensationPackage, error)
	findAllFunc                  func(context.Context, uuid.UUID) ([]*domain.EmployeeCompensationPackage, error)
	findAllIncludingDeletedFunc  func(context.Context, uuid.UUID) ([]*domain.EmployeeCompensationPackage, error)
	countEmployeesUsingFunc      func(context.Context, uuid.UUID) (int64, error)
	countPayrollResultsUsingFunc func(context.Context, uuid.UUID) (int64, error)
}

func (m *mockCompensationPackageRepository) Create(ctx context.Context, pkg *domain.EmployeeCompensationPackage) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, pkg)
	}
	return nil
}

func (m *mockCompensationPackageRepository) Update(ctx context.Context, pkg *domain.EmployeeCompensationPackage) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, pkg)
	}
	return nil
}

func (m *mockCompensationPackageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockCompensationPackageRepository) Restore(ctx context.Context, id uuid.UUID) error {
	if m.restoreFunc != nil {
		return m.restoreFunc(ctx, id)
	}
	return nil
}

func (m *mockCompensationPackageRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	if m.hardDeleteFunc != nil {
		return m.hardDeleteFunc(ctx, id)
	}
	return nil
}

func (m *mockCompensationPackageRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.EmployeeCompensationPackage, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, sqlite.ErrRecordNotFound
}

func (m *mockCompensationPackageRepository) FindByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*domain.EmployeeCompensationPackage, error) {
	if m.findByIDIncludingDeletedFunc != nil {
		return m.findByIDIncludingDeletedFunc(ctx, id)
	}
	return nil, sqlite.ErrRecordNotFound
}

func (m *mockCompensationPackageRepository) FindAll(ctx context.Context, orgID uuid.UUID) ([]*domain.EmployeeCompensationPackage, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc(ctx, orgID)
	}
	return []*domain.EmployeeCompensationPackage{}, nil
}

func (m *mockCompensationPackageRepository) FindAllIncludingDeleted(ctx context.Context, orgID uuid.UUID) ([]*domain.EmployeeCompensationPackage, error) {
	if m.findAllIncludingDeletedFunc != nil {
		return m.findAllIncludingDeletedFunc(ctx, orgID)
	}
	return []*domain.EmployeeCompensationPackage{}, nil
}

func (m *mockCompensationPackageRepository) CountEmployeesUsing(ctx context.Context, pkgID uuid.UUID) (int64, error) {
	if m.countEmployeesUsingFunc != nil {
		return m.countEmployeesUsingFunc(ctx, pkgID)
	}
	return 0, nil
}

func (m *mockCompensationPackageRepository) CountPayrollResultsUsing(ctx context.Context, pkgID uuid.UUID) (int64, error) {
	if m.countPayrollResultsUsingFunc != nil {
		return m.countPayrollResultsUsingFunc(ctx, pkgID)
	}
	return 0, nil
}

// ===============================================================================
// TEST HELPERS
// ===============================================================================

// createTestCompensationPackage creates a valid compensation package for testing.
func createTestCompensationPackage() *domain.EmployeeCompensationPackage {
	baseSalary := money.FromCents(500000) // 5000.00 MAD
	now := time.Now().UTC()

	return &domain.EmployeeCompensationPackage{
		ID:         uuid.New(),
		OrgID:      uuid.New(),
		Name:       "Test Package",
		BaseSalary: baseSalary,
		Currency:   money.MAD,
		CreatedAt:  now,
		UpdatedAt:  now,
		DeletedAt:  nil,
	}
}

// ===============================================================================
// CREATE TESTS
// ===============================================================================

func TestCompensationPackageService_CreateCompensationPackage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("creates package successfully with generated UUID and timestamps", func(t *testing.T) {
		t.Parallel()

		mock := &mockCompensationPackageRepository{
			createFunc: func(ctx context.Context, pkg *domain.EmployeeCompensationPackage) error {
				// Verify UUID was generated
				assert.NotEqual(t, uuid.Nil, pkg.ID)
				// Verify timestamps were set
				assert.False(t, pkg.CreatedAt.IsZero())
				assert.False(t, pkg.UpdatedAt.IsZero())
				assert.Nil(t, pkg.DeletedAt)
				return nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		pkg := &domain.EmployeeCompensationPackage{
			OrgID:      uuid.New(),
			Name:       "Test Package",
			BaseSalary: money.FromCents(500000),
			Currency:   money.MAD,
		}

		err := service.CreateCompensationPackage(ctx, pkg)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, pkg.ID)
		assert.False(t, pkg.CreatedAt.IsZero())
		assert.False(t, pkg.UpdatedAt.IsZero())
	})

	t.Run("preserves provided UUID", func(t *testing.T) {
		t.Parallel()

		providedID := uuid.New()

		mock := &mockCompensationPackageRepository{
			createFunc: func(ctx context.Context, pkg *domain.EmployeeCompensationPackage) error {
				assert.Equal(t, providedID, pkg.ID)
				return nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		pkg := &domain.EmployeeCompensationPackage{
			ID:         providedID,
			OrgID:      uuid.New(),
			Name:       "Test Package",
			BaseSalary: money.FromCents(500000),
			Currency:   money.MAD,
		}

		err := service.CreateCompensationPackage(ctx, pkg)

		require.NoError(t, err)
		assert.Equal(t, providedID, pkg.ID)
	})

	t.Run("returns error for invalid package (below SMIG)", func(t *testing.T) {
		t.Parallel()

		mock := &mockCompensationPackageRepository{}
		service := application.NewCompensationPackageService(mock)

		pkg := &domain.EmployeeCompensationPackage{
			OrgID:      uuid.New(),
			Name:       "Test Package",
			BaseSalary: money.FromCents(200000), // 2000.00 MAD - below SMIG
			Currency:   money.MAD,
		}

		err := service.CreateCompensationPackage(ctx, pkg)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid compensation package")
		assert.ErrorIs(t, err, domain.ErrInvalidEmployeeCompensationPackageBaseSalary)
	})

	t.Run("returns error for invalid currency", func(t *testing.T) {
		t.Parallel()

		mock := &mockCompensationPackageRepository{}
		service := application.NewCompensationPackageService(mock)

		pkg := &domain.EmployeeCompensationPackage{
			OrgID:      uuid.New(),
			Name:       "Test Package",
			BaseSalary: money.FromCents(500000),
			Currency:   "USD", // Not supported
		}

		err := service.CreateCompensationPackage(ctx, pkg)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid compensation package")
		assert.ErrorIs(t, err, money.ErrCurrencyNotSupported)
	})

	t.Run("wraps repository error", func(t *testing.T) {
		t.Parallel()

		repoErr := errors.New("database connection failed")

		mock := &mockCompensationPackageRepository{
			createFunc: func(ctx context.Context, pkg *domain.EmployeeCompensationPackage) error {
				return repoErr
			},
		}

		service := application.NewCompensationPackageService(mock)

		pkg := &domain.EmployeeCompensationPackage{
			OrgID:      uuid.New(),
			Name:       "Test Package",
			BaseSalary: money.FromCents(500000),
			Currency:   money.MAD,
		}

		err := service.CreateCompensationPackage(ctx, pkg)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create compensation package")
		assert.ErrorIs(t, err, repoErr)
	})
}

// ===============================================================================
// UPDATE TESTS
// ===============================================================================

func TestCompensationPackageService_UpdateCompensationPackage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("updates package successfully when not in use", func(t *testing.T) {
		t.Parallel()

		pkg := createTestCompensationPackage()
		beforeUpdate := pkg.UpdatedAt

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, pkgID uuid.UUID) (int64, error) {
				return 0, nil // Not in use
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, pkgID uuid.UUID) (int64, error) {
				return 0, nil // Not in use
			},
			updateFunc: func(ctx context.Context, p *domain.EmployeeCompensationPackage) error {
				// Verify UpdatedAt was updated
				assert.True(t, p.UpdatedAt.After(beforeUpdate))
				return nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		// Simulate a small delay to ensure UpdatedAt changes
		time.Sleep(10 * time.Millisecond)

		err := service.UpdateCompensationPackage(ctx, pkg)

		require.NoError(t, err)
		assert.True(t, pkg.UpdatedAt.After(beforeUpdate))
	})

	t.Run("returns ErrCompensationPackageInUse when employees using it", func(t *testing.T) {
		t.Parallel()

		pkg := createTestCompensationPackage()

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, pkgID uuid.UUID) (int64, error) {
				return 5, nil // 5 employees using it
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, pkgID uuid.UUID) (int64, error) {
				return 0, nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		err := service.UpdateCompensationPackage(ctx, pkg)

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrCompensationPackageInUse)
	})

	t.Run("returns ErrCompensationPackageInUse when payroll results using it", func(t *testing.T) {
		t.Parallel()

		pkg := createTestCompensationPackage()

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, pkgID uuid.UUID) (int64, error) {
				return 0, nil
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, pkgID uuid.UUID) (int64, error) {
				return 10, nil // 10 payroll results using it
			},
		}

		service := application.NewCompensationPackageService(mock)

		err := service.UpdateCompensationPackage(ctx, pkg)

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrCompensationPackageInUse)
	})

	t.Run("returns ErrCompensationPackageInUse when both employees and payroll results using it", func(t *testing.T) {
		t.Parallel()

		pkg := createTestCompensationPackage()

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, pkgID uuid.UUID) (int64, error) {
				return 3, nil
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, pkgID uuid.UUID) (int64, error) {
				return 7, nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		err := service.UpdateCompensationPackage(ctx, pkg)

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrCompensationPackageInUse)
	})

	t.Run("returns error when usage check fails", func(t *testing.T) {
		t.Parallel()

		pkg := createTestCompensationPackage()
		checkErr := errors.New("database timeout")

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, pkgID uuid.UUID) (int64, error) {
				return 0, checkErr
			},
		}

		service := application.NewCompensationPackageService(mock)

		err := service.UpdateCompensationPackage(ctx, pkg)

		require.Error(t, err)
		assert.ErrorIs(t, err, checkErr)
		assert.Contains(t, err.Error(), "failed to count employees using package")
	})

	t.Run("returns error for invalid package data", func(t *testing.T) {
		t.Parallel()

		pkg := createTestCompensationPackage()
		pkg.BaseSalary = money.FromCents(200000) // Below SMIG

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, pkgID uuid.UUID) (int64, error) {
				return 0, nil
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, pkgID uuid.UUID) (int64, error) {
				return 0, nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		err := service.UpdateCompensationPackage(ctx, pkg)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid compensation package")
		assert.ErrorIs(t, err, domain.ErrInvalidEmployeeCompensationPackageBaseSalary)
	})

	t.Run("translates ErrRecordNotFound to ErrCompensationPackageNotFound", func(t *testing.T) {
		t.Parallel()

		pkg := createTestCompensationPackage()

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, pkgID uuid.UUID) (int64, error) {
				return 0, nil
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, pkgID uuid.UUID) (int64, error) {
				return 0, nil
			},
			updateFunc: func(ctx context.Context, p *domain.EmployeeCompensationPackage) error {
				return sqlite.ErrRecordNotFound
			},
		}

		service := application.NewCompensationPackageService(mock)

		err := service.UpdateCompensationPackage(ctx, pkg)

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrCompensationPackageNotFound)
	})

	t.Run("translates repository ErrCompensationPackageInUse (defense in depth)", func(t *testing.T) {
		t.Parallel()

		pkg := createTestCompensationPackage()

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, pkgID uuid.UUID) (int64, error) {
				return 0, nil // Service check passes
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, pkgID uuid.UUID) (int64, error) {
				return 0, nil // Service check passes
			},
			updateFunc: func(ctx context.Context, p *domain.EmployeeCompensationPackage) error {
				// But repository check fails (race condition)
				return sqlite.ErrCompensationPackageInUse
			},
		}

		service := application.NewCompensationPackageService(mock)

		err := service.UpdateCompensationPackage(ctx, pkg)

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrCompensationPackageInUse)
	})
}

// ===============================================================================
// DELETE TESTS
// ===============================================================================

func TestCompensationPackageService_DeleteCompensationPackage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("deletes package successfully when not in use", func(t *testing.T) {
		t.Parallel()

		pkgID := uuid.New()

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, nil
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, nil
			},
			deleteFunc: func(ctx context.Context, id uuid.UUID) error {
				assert.Equal(t, pkgID, id)
				return nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		err := service.DeleteCompensationPackage(ctx, pkgID)

		require.NoError(t, err)
	})

	t.Run("returns ErrCompensationPackageInUse when employees using it", func(t *testing.T) {
		t.Parallel()

		pkgID := uuid.New()

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 2, nil // In use
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		err := service.DeleteCompensationPackage(ctx, pkgID)

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrCompensationPackageInUse)
	})

	t.Run("returns ErrCompensationPackageInUse when payroll results using it", func(t *testing.T) {
		t.Parallel()

		pkgID := uuid.New()

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, nil
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 5, nil // In use
			},
		}

		service := application.NewCompensationPackageService(mock)

		err := service.DeleteCompensationPackage(ctx, pkgID)

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrCompensationPackageInUse)
	})

	t.Run("translates ErrRecordNotFound to ErrCompensationPackageNotFound", func(t *testing.T) {
		t.Parallel()

		pkgID := uuid.New()

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, nil
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, nil
			},
			deleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return sqlite.ErrRecordNotFound
			},
		}

		service := application.NewCompensationPackageService(mock)

		err := service.DeleteCompensationPackage(ctx, pkgID)

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrCompensationPackageNotFound)
	})
}

// ===============================================================================
// RESTORE TESTS
// ===============================================================================

func TestCompensationPackageService_RestoreCompensationPackage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("restores package successfully without usage check", func(t *testing.T) {
		t.Parallel()

		pkgID := uuid.New()

		mock := &mockCompensationPackageRepository{
			restoreFunc: func(ctx context.Context, id uuid.UUID) error {
				assert.Equal(t, pkgID, id)
				return nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		err := service.RestoreCompensationPackage(ctx, pkgID)

		require.NoError(t, err)
	})

	t.Run("translates ErrRecordNotFound to ErrCompensationPackageNotFound", func(t *testing.T) {
		t.Parallel()

		pkgID := uuid.New()

		mock := &mockCompensationPackageRepository{
			restoreFunc: func(ctx context.Context, id uuid.UUID) error {
				return sqlite.ErrRecordNotFound
			},
		}

		service := application.NewCompensationPackageService(mock)

		err := service.RestoreCompensationPackage(ctx, pkgID)

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrCompensationPackageNotFound)
	})
}

// ===============================================================================
// HARD DELETE TESTS
// ===============================================================================

func TestCompensationPackageService_HardDeleteCompensationPackage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("hard deletes package successfully when not in use", func(t *testing.T) {
		t.Parallel()

		pkgID := uuid.New()

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, nil
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, nil
			},
			hardDeleteFunc: func(ctx context.Context, id uuid.UUID) error {
				assert.Equal(t, pkgID, id)
				return nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		err := service.HardDeleteCompensationPackage(ctx, pkgID)

		require.NoError(t, err)
	})

	t.Run("returns ErrCompensationPackageInUse when in use", func(t *testing.T) {
		t.Parallel()

		pkgID := uuid.New()

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 1, nil // In use
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		err := service.HardDeleteCompensationPackage(ctx, pkgID)

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrCompensationPackageInUse)
	})

	t.Run("translates ErrRecordNotFound to ErrCompensationPackageNotFound", func(t *testing.T) {
		t.Parallel()

		pkgID := uuid.New()

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, nil
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, nil
			},
			hardDeleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return sqlite.ErrRecordNotFound
			},
		}

		service := application.NewCompensationPackageService(mock)

		err := service.HardDeleteCompensationPackage(ctx, pkgID)

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrCompensationPackageNotFound)
	})
}

// ===============================================================================
// QUERY TESTS
// ===============================================================================

func TestCompensationPackageService_GetCompensationPackage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("returns package successfully", func(t *testing.T) {
		t.Parallel()

		expected := createTestCompensationPackage()

		mock := &mockCompensationPackageRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.EmployeeCompensationPackage, error) {
				assert.Equal(t, expected.ID, id)
				return expected, nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		result, err := service.GetCompensationPackage(ctx, expected.ID)

		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("translates ErrRecordNotFound to ErrCompensationPackageNotFound", func(t *testing.T) {
		t.Parallel()

		pkgID := uuid.New()

		mock := &mockCompensationPackageRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.EmployeeCompensationPackage, error) {
				return nil, sqlite.ErrRecordNotFound
			},
		}

		service := application.NewCompensationPackageService(mock)

		result, err := service.GetCompensationPackage(ctx, pkgID)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, application.ErrCompensationPackageNotFound)
	})
}

func TestCompensationPackageService_GetCompensationPackageIncludingDeleted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("returns package including deleted", func(t *testing.T) {
		t.Parallel()

		expected := createTestCompensationPackage()
		deletedAt := time.Now().UTC()
		expected.DeletedAt = &deletedAt

		mock := &mockCompensationPackageRepository{
			findByIDIncludingDeletedFunc: func(ctx context.Context, id uuid.UUID) (*domain.EmployeeCompensationPackage, error) {
				assert.Equal(t, expected.ID, id)
				return expected, nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		result, err := service.GetCompensationPackageIncludingDeleted(ctx, expected.ID)

		require.NoError(t, err)
		assert.Equal(t, expected, result)
		assert.NotNil(t, result.DeletedAt)
	})

	t.Run("translates ErrRecordNotFound to ErrCompensationPackageNotFound", func(t *testing.T) {
		t.Parallel()

		pkgID := uuid.New()

		mock := &mockCompensationPackageRepository{
			findByIDIncludingDeletedFunc: func(ctx context.Context, id uuid.UUID) (*domain.EmployeeCompensationPackage, error) {
				return nil, sqlite.ErrRecordNotFound
			},
		}

		service := application.NewCompensationPackageService(mock)

		result, err := service.GetCompensationPackageIncludingDeleted(ctx, pkgID)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, application.ErrCompensationPackageNotFound)
	})
}

func TestCompensationPackageService_ListCompensationPackages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("returns all packages", func(t *testing.T) {
		t.Parallel()

		pkg1 := createTestCompensationPackage()
		pkg2 := createTestCompensationPackage()
		expected := []*domain.EmployeeCompensationPackage{pkg1, pkg2}

		orgID := uuid.New()
		mock := &mockCompensationPackageRepository{
			findAllFunc: func(ctx context.Context, id uuid.UUID) ([]*domain.EmployeeCompensationPackage, error) {
				return expected, nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		result, err := service.ListCompensationPackages(ctx, orgID)

		require.NoError(t, err)
		assert.Equal(t, expected, result)
		assert.Len(t, result, 2)
	})

	t.Run("returns empty slice when no packages", func(t *testing.T) {
		t.Parallel()

		orgID := uuid.New()
		mock := &mockCompensationPackageRepository{
			findAllFunc: func(ctx context.Context, id uuid.UUID) ([]*domain.EmployeeCompensationPackage, error) {
				return []*domain.EmployeeCompensationPackage{}, nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		result, err := service.ListCompensationPackages(ctx, orgID)

		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestCompensationPackageService_ListCompensationPackagesIncludingDeleted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("returns all packages including deleted", func(t *testing.T) {
		t.Parallel()

		pkg1 := createTestCompensationPackage()
		pkg2 := createTestCompensationPackage()
		deletedAt := time.Now().UTC()
		pkg2.DeletedAt = &deletedAt

		expected := []*domain.EmployeeCompensationPackage{pkg1, pkg2}

		orgID := uuid.New()
		mock := &mockCompensationPackageRepository{
			findAllIncludingDeletedFunc: func(ctx context.Context, id uuid.UUID) ([]*domain.EmployeeCompensationPackage, error) {
				return expected, nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		result, err := service.ListCompensationPackagesIncludingDeleted(ctx, orgID)

		require.NoError(t, err)
		assert.Equal(t, expected, result)
		assert.Len(t, result, 2)
		assert.NotNil(t, result[1].DeletedAt)
	})
}

// ===============================================================================
// USAGE QUERY TESTS
// ===============================================================================

func TestCompensationPackageService_IsCompensationPackageInUse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("returns true when employees using package", func(t *testing.T) {
		t.Parallel()

		pkgID := uuid.New()

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 3, nil
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		inUse, err := service.IsCompensationPackageInUse(ctx, pkgID)

		require.NoError(t, err)
		assert.True(t, inUse)
	})

	t.Run("returns true when payroll results using package", func(t *testing.T) {
		t.Parallel()

		pkgID := uuid.New()

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, nil
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 7, nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		inUse, err := service.IsCompensationPackageInUse(ctx, pkgID)

		require.NoError(t, err)
		assert.True(t, inUse)
	})

	t.Run("returns false when not in use", func(t *testing.T) {
		t.Parallel()

		pkgID := uuid.New()

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, nil
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		inUse, err := service.IsCompensationPackageInUse(ctx, pkgID)

		require.NoError(t, err)
		assert.False(t, inUse)
	})

	t.Run("returns error when count fails", func(t *testing.T) {
		t.Parallel()

		pkgID := uuid.New()
		countErr := errors.New("database timeout")

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, countErr
			},
		}

		service := application.NewCompensationPackageService(mock)

		inUse, err := service.IsCompensationPackageInUse(ctx, pkgID)

		require.Error(t, err)
		assert.False(t, inUse)
		assert.ErrorIs(t, err, countErr)
	})
}

func TestCompensationPackageService_GetCompensationPackageUsageCount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("returns correct counts", func(t *testing.T) {
		t.Parallel()

		pkgID := uuid.New()

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 5, nil
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 12, nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		empCount, resultCount, err := service.GetCompensationPackageUsageCount(ctx, pkgID)

		require.NoError(t, err)
		assert.Equal(t, int64(5), empCount)
		assert.Equal(t, int64(12), resultCount)
	})

	t.Run("returns zeros when not in use", func(t *testing.T) {
		t.Parallel()

		pkgID := uuid.New()

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, nil
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, nil
			},
		}

		service := application.NewCompensationPackageService(mock)

		empCount, resultCount, err := service.GetCompensationPackageUsageCount(ctx, pkgID)

		require.NoError(t, err)
		assert.Equal(t, int64(0), empCount)
		assert.Equal(t, int64(0), resultCount)
	})

	t.Run("returns error when employee count fails", func(t *testing.T) {
		t.Parallel()

		pkgID := uuid.New()
		countErr := errors.New("database error")

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, countErr
			},
		}

		service := application.NewCompensationPackageService(mock)

		empCount, resultCount, err := service.GetCompensationPackageUsageCount(ctx, pkgID)

		require.Error(t, err)
		assert.Equal(t, int64(0), empCount)
		assert.Equal(t, int64(0), resultCount)
		assert.ErrorIs(t, err, countErr)
	})

	t.Run("returns error when payroll result count fails", func(t *testing.T) {
		t.Parallel()

		pkgID := uuid.New()
		countErr := errors.New("database error")

		mock := &mockCompensationPackageRepository{
			countEmployeesUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 5, nil
			},
			countPayrollResultsUsingFunc: func(ctx context.Context, id uuid.UUID) (int64, error) {
				return 0, countErr
			},
		}

		service := application.NewCompensationPackageService(mock)

		empCount, resultCount, err := service.GetCompensationPackageUsageCount(ctx, pkgID)

		require.Error(t, err)
		assert.Equal(t, int64(0), empCount)
		assert.Equal(t, int64(0), resultCount)
		assert.ErrorIs(t, err, countErr)
	})
}
