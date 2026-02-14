package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sqlite "github.com/iamoeg/bootdev-capstone/internal/adapter/sqlite"
	"github.com/iamoeg/bootdev-capstone/internal/application"
	"github.com/iamoeg/bootdev-capstone/internal/domain"
)

// ===============================================================================
// MOCK REPOSITORY
// ===============================================================================

// mockOrgRepo is a mock implementation of organizationRepository for testing.
// Each method can be overridden with custom behavior using function fields.
type mockOrgRepo struct {
	createFunc                   func(context.Context, *domain.Organization) error
	updateFunc                   func(context.Context, *domain.Organization) error
	deleteFunc                   func(context.Context, uuid.UUID) error
	restoreFunc                  func(context.Context, uuid.UUID) error
	hardDeleteFunc               func(context.Context, uuid.UUID) error
	findByIDFunc                 func(context.Context, uuid.UUID) (*domain.Organization, error)
	findByIDIncludingDeletedFunc func(context.Context, uuid.UUID) (*domain.Organization, error)
	findAllFunc                  func(context.Context) ([]*domain.Organization, error)
	findAllIncludingDeletedFunc  func(context.Context) ([]*domain.Organization, error)
}

func (m *mockOrgRepo) Create(ctx context.Context, org *domain.Organization) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, org)
	}
	return nil
}

func (m *mockOrgRepo) Update(ctx context.Context, org *domain.Organization) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, org)
	}
	return nil
}

func (m *mockOrgRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockOrgRepo) Restore(ctx context.Context, id uuid.UUID) error {
	if m.restoreFunc != nil {
		return m.restoreFunc(ctx, id)
	}
	return nil
}

func (m *mockOrgRepo) HardDelete(ctx context.Context, id uuid.UUID) error {
	if m.hardDeleteFunc != nil {
		return m.hardDeleteFunc(ctx, id)
	}
	return nil
}

func (m *mockOrgRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, sqlite.ErrRecordNotFound
}

func (m *mockOrgRepo) FindByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	if m.findByIDIncludingDeletedFunc != nil {
		return m.findByIDIncludingDeletedFunc(ctx, id)
	}
	return nil, sqlite.ErrRecordNotFound
}

func (m *mockOrgRepo) FindAll(ctx context.Context) ([]*domain.Organization, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc(ctx)
	}
	return []*domain.Organization{}, nil
}

func (m *mockOrgRepo) FindAllIncludingDeleted(ctx context.Context) ([]*domain.Organization, error) {
	if m.findAllIncludingDeletedFunc != nil {
		return m.findAllIncludingDeletedFunc(ctx)
	}
	return []*domain.Organization{}, nil
}

// ===============================================================================
// TEST HELPERS
// ===============================================================================

// createValidOrg creates a valid organization for testing.
// Each field is set to satisfy domain validation rules.
func createValidOrg() *domain.Organization {
	return &domain.Organization{
		Name:      "Test Organization",
		Address:   "123 Test Street, Casablanca",
		Activity:  "Software Development",
		LegalForm: domain.LegalFormSARL,
		ICENum:    "123456789012345",
		IFNum:     "12345678",
		RCNum:     "123456",
		CNSSNum:   "1234567",
		BankRIB:   "123456789012345678901234",
	}
}

// ===============================================================================
// CREATE ORGANIZATION TESTS
// ===============================================================================

func TestOrganizationService_CreateOrganization(t *testing.T) {
	t.Parallel()

	t.Run("generates UUID when not provided", func(t *testing.T) {
		t.Parallel()

		var capturedOrg *domain.Organization
		mock := &mockOrgRepo{
			createFunc: func(ctx context.Context, org *domain.Organization) error {
				capturedOrg = org
				return nil
			},
		}

		service := application.NewOrganizationService(mock)
		org := createValidOrg()
		// Don't set ID - should be generated

		err := service.CreateOrganization(context.Background(), org)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, capturedOrg.ID)
		assert.Equal(t, capturedOrg.ID, org.ID) // Modified in-place
	})

	t.Run("uses provided UUID", func(t *testing.T) {
		t.Parallel()

		expectedID := uuid.New()
		var capturedOrg *domain.Organization
		mock := &mockOrgRepo{
			createFunc: func(ctx context.Context, org *domain.Organization) error {
				capturedOrg = org
				return nil
			},
		}

		service := application.NewOrganizationService(mock)
		org := createValidOrg()
		org.ID = expectedID

		err := service.CreateOrganization(context.Background(), org)

		require.NoError(t, err)
		assert.Equal(t, expectedID, capturedOrg.ID)
		assert.Equal(t, expectedID, org.ID)
	})

	t.Run("sets CreatedAt timestamp", func(t *testing.T) {
		t.Parallel()

		var capturedOrg *domain.Organization
		mock := &mockOrgRepo{
			createFunc: func(ctx context.Context, org *domain.Organization) error {
				capturedOrg = org
				return nil
			},
		}

		service := application.NewOrganizationService(mock)
		org := createValidOrg()
		before := time.Now().UTC()

		err := service.CreateOrganization(context.Background(), org)

		require.NoError(t, err)
		assert.True(t, capturedOrg.CreatedAt.After(before) || capturedOrg.CreatedAt.Equal(before))
		assert.True(t, capturedOrg.CreatedAt.Before(time.Now().UTC().Add(time.Second)))
	})

	t.Run("sets UpdatedAt timestamp equal to CreatedAt", func(t *testing.T) {
		t.Parallel()

		var capturedOrg *domain.Organization
		mock := &mockOrgRepo{
			createFunc: func(ctx context.Context, org *domain.Organization) error {
				capturedOrg = org
				return nil
			},
		}

		service := application.NewOrganizationService(mock)
		org := createValidOrg()

		err := service.CreateOrganization(context.Background(), org)

		require.NoError(t, err)
		assert.Equal(t, capturedOrg.CreatedAt, capturedOrg.UpdatedAt)
	})

	t.Run("ensures DeletedAt is nil", func(t *testing.T) {
		t.Parallel()

		var capturedOrg *domain.Organization
		mock := &mockOrgRepo{
			createFunc: func(ctx context.Context, org *domain.Organization) error {
				capturedOrg = org
				return nil
			},
		}

		service := application.NewOrganizationService(mock)
		org := createValidOrg()
		// Set DeletedAt - should be cleared
		deletedTime := time.Now().UTC()
		org.DeletedAt = &deletedTime

		err := service.CreateOrganization(context.Background(), org)

		require.NoError(t, err)
		assert.Nil(t, capturedOrg.DeletedAt)
	})

	t.Run("validates organization before creating", func(t *testing.T) {
		t.Parallel()

		mock := &mockOrgRepo{
			createFunc: func(ctx context.Context, org *domain.Organization) error {
				t.Fatal("should not call repo.Create for invalid organization")
				return nil
			},
		}

		service := application.NewOrganizationService(mock)
		org := createValidOrg()
		org.Name = "" // Invalid - name is required

		err := service.CreateOrganization(context.Background(), org)

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrOrgNameRequired)
	})

	t.Run("validates legal form before creating", func(t *testing.T) {
		t.Parallel()

		mock := &mockOrgRepo{
			createFunc: func(ctx context.Context, org *domain.Organization) error {
				t.Fatal("should not call repo.Create for invalid organization")
				return nil
			},
		}

		service := application.NewOrganizationService(mock)
		org := createValidOrg()
		org.LegalForm = "INVALID" // Invalid legal form

		err := service.CreateOrganization(context.Background(), org)

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrOrgLegalFormNotSupported)
	})

	t.Run("translates duplicate error to ErrOrganizationExists", func(t *testing.T) {
		t.Parallel()

		mock := &mockOrgRepo{
			createFunc: func(ctx context.Context, org *domain.Organization) error {
				return sqlite.ErrDuplicateRecord
			},
		}

		service := application.NewOrganizationService(mock)
		org := createValidOrg()

		err := service.CreateOrganization(context.Background(), org)

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrOrganizationExists)
	})

	t.Run("propagates other repository errors", func(t *testing.T) {
		t.Parallel()

		repoErr := errors.New("database connection failed")
		mock := &mockOrgRepo{
			createFunc: func(ctx context.Context, org *domain.Organization) error {
				return repoErr
			},
		}

		service := application.NewOrganizationService(mock)
		org := createValidOrg()

		err := service.CreateOrganization(context.Background(), org)

		require.Error(t, err)
		assert.ErrorIs(t, err, repoErr)
	})

	t.Run("successfully creates valid organization", func(t *testing.T) {
		t.Parallel()

		var capturedOrg *domain.Organization
		mock := &mockOrgRepo{
			createFunc: func(ctx context.Context, org *domain.Organization) error {
				capturedOrg = org
				return nil
			},
		}

		service := application.NewOrganizationService(mock)
		org := createValidOrg()

		err := service.CreateOrganization(context.Background(), org)

		require.NoError(t, err)
		assert.NotNil(t, capturedOrg)
		assert.Equal(t, org.Name, capturedOrg.Name)
		assert.Equal(t, org.ICENum, capturedOrg.ICENum)
	})
}

// ===============================================================================
// UPDATE ORGANIZATION TESTS
// ===============================================================================

func TestOrganizationService_UpdateOrganization(t *testing.T) {
	t.Parallel()

	t.Run("updates UpdatedAt timestamp", func(t *testing.T) {
		t.Parallel()

		var capturedOrg *domain.Organization
		mock := &mockOrgRepo{
			updateFunc: func(ctx context.Context, org *domain.Organization) error {
				capturedOrg = org
				return nil
			},
		}

		service := application.NewOrganizationService(mock)
		org := createValidOrg()
		org.ID = uuid.New()
		org.CreatedAt = time.Now().UTC().Add(-24 * time.Hour) // Created yesterday
		org.UpdatedAt = org.CreatedAt                         // Initially same
		before := time.Now().UTC()

		err := service.UpdateOrganization(context.Background(), org)

		require.NoError(t, err)
		assert.True(t, capturedOrg.UpdatedAt.After(before) || capturedOrg.UpdatedAt.Equal(before))
		assert.True(t, capturedOrg.UpdatedAt.After(capturedOrg.CreatedAt))
	})

	t.Run("does not modify CreatedAt", func(t *testing.T) {
		t.Parallel()

		var capturedOrg *domain.Organization
		mock := &mockOrgRepo{
			updateFunc: func(ctx context.Context, org *domain.Organization) error {
				capturedOrg = org
				return nil
			},
		}

		service := application.NewOrganizationService(mock)
		org := createValidOrg()
		org.ID = uuid.New()
		originalCreatedAt := time.Now().UTC().Add(-24 * time.Hour)
		org.CreatedAt = originalCreatedAt

		err := service.UpdateOrganization(context.Background(), org)

		require.NoError(t, err)
		assert.Equal(t, originalCreatedAt, capturedOrg.CreatedAt)
	})

	t.Run("validates organization before updating", func(t *testing.T) {
		t.Parallel()

		mock := &mockOrgRepo{
			updateFunc: func(ctx context.Context, org *domain.Organization) error {
				t.Fatal("should not call repo.Update for invalid organization")
				return nil
			},
		}

		service := application.NewOrganizationService(mock)
		org := createValidOrg()
		org.ID = uuid.New()
		org.LegalForm = "INVALID" // Invalid

		err := service.UpdateOrganization(context.Background(), org)

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrOrgLegalFormNotSupported)
	})

	t.Run("translates not found error to ErrOrganizationNotFound", func(t *testing.T) {
		t.Parallel()

		mock := &mockOrgRepo{
			updateFunc: func(ctx context.Context, org *domain.Organization) error {
				return sqlite.ErrRecordNotFound
			},
		}

		service := application.NewOrganizationService(mock)
		org := createValidOrg()
		org.ID = uuid.New()

		err := service.UpdateOrganization(context.Background(), org)

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrOrganizationNotFound)
	})

	t.Run("translates duplicate error to ErrOrganizationExists", func(t *testing.T) {
		t.Parallel()

		mock := &mockOrgRepo{
			updateFunc: func(ctx context.Context, org *domain.Organization) error {
				return sqlite.ErrDuplicateRecord
			},
		}

		service := application.NewOrganizationService(mock)
		org := createValidOrg()
		org.ID = uuid.New()

		err := service.UpdateOrganization(context.Background(), org)

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrOrganizationExists)
	})

	t.Run("propagates other repository errors", func(t *testing.T) {
		t.Parallel()

		repoErr := errors.New("database connection failed")
		mock := &mockOrgRepo{
			updateFunc: func(ctx context.Context, org *domain.Organization) error {
				return repoErr
			},
		}

		service := application.NewOrganizationService(mock)
		org := createValidOrg()
		org.ID = uuid.New()

		err := service.UpdateOrganization(context.Background(), org)

		require.Error(t, err)
		assert.ErrorIs(t, err, repoErr)
	})

	t.Run("successfully updates valid organization", func(t *testing.T) {
		t.Parallel()

		var capturedOrg *domain.Organization
		mock := &mockOrgRepo{
			updateFunc: func(ctx context.Context, org *domain.Organization) error {
				capturedOrg = org
				return nil
			},
		}

		service := application.NewOrganizationService(mock)
		org := createValidOrg()
		org.ID = uuid.New()
		org.Name = "Updated Name"

		err := service.UpdateOrganization(context.Background(), org)

		require.NoError(t, err)
		assert.Equal(t, "Updated Name", capturedOrg.Name)
	})
}

// ===============================================================================
// DELETE ORGANIZATION TESTS
// ===============================================================================

func TestOrganizationService_DeleteOrganization(t *testing.T) {
	t.Parallel()

	t.Run("calls repository Delete with correct ID", func(t *testing.T) {
		t.Parallel()

		expectedID := uuid.New()
		var capturedID uuid.UUID
		mock := &mockOrgRepo{
			deleteFunc: func(ctx context.Context, id uuid.UUID) error {
				capturedID = id
				return nil
			},
		}

		service := application.NewOrganizationService(mock)

		err := service.DeleteOrganization(context.Background(), expectedID)

		require.NoError(t, err)
		assert.Equal(t, expectedID, capturedID)
	})

	t.Run("translates not found error to ErrOrganizationNotFound", func(t *testing.T) {
		t.Parallel()

		mock := &mockOrgRepo{
			deleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return sqlite.ErrRecordNotFound
			},
		}

		service := application.NewOrganizationService(mock)

		err := service.DeleteOrganization(context.Background(), uuid.New())

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrOrganizationNotFound)
	})

	t.Run("propagates other repository errors", func(t *testing.T) {
		t.Parallel()

		repoErr := errors.New("database connection failed")
		mock := &mockOrgRepo{
			deleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return repoErr
			},
		}

		service := application.NewOrganizationService(mock)

		err := service.DeleteOrganization(context.Background(), uuid.New())

		require.Error(t, err)
		assert.ErrorIs(t, err, repoErr)
	})

	t.Run("successfully deletes organization", func(t *testing.T) {
		t.Parallel()

		mock := &mockOrgRepo{
			deleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return nil
			},
		}

		service := application.NewOrganizationService(mock)

		err := service.DeleteOrganization(context.Background(), uuid.New())

		require.NoError(t, err)
	})
}

// ===============================================================================
// RESTORE ORGANIZATION TESTS
// ===============================================================================

func TestOrganizationService_RestoreOrganization(t *testing.T) {
	t.Parallel()

	t.Run("calls repository Restore with correct ID", func(t *testing.T) {
		t.Parallel()

		expectedID := uuid.New()
		var capturedID uuid.UUID
		mock := &mockOrgRepo{
			restoreFunc: func(ctx context.Context, id uuid.UUID) error {
				capturedID = id
				return nil
			},
		}

		service := application.NewOrganizationService(mock)

		err := service.RestoreOrganization(context.Background(), expectedID)

		require.NoError(t, err)
		assert.Equal(t, expectedID, capturedID)
	})

	t.Run("translates not found error to ErrOrganizationNotFound", func(t *testing.T) {
		t.Parallel()

		mock := &mockOrgRepo{
			restoreFunc: func(ctx context.Context, id uuid.UUID) error {
				return sqlite.ErrRecordNotFound
			},
		}

		service := application.NewOrganizationService(mock)

		err := service.RestoreOrganization(context.Background(), uuid.New())

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrOrganizationNotFound)
	})

	t.Run("propagates other repository errors", func(t *testing.T) {
		t.Parallel()

		repoErr := errors.New("database connection failed")
		mock := &mockOrgRepo{
			restoreFunc: func(ctx context.Context, id uuid.UUID) error {
				return repoErr
			},
		}

		service := application.NewOrganizationService(mock)

		err := service.RestoreOrganization(context.Background(), uuid.New())

		require.Error(t, err)
		assert.ErrorIs(t, err, repoErr)
	})

	t.Run("successfully restores organization", func(t *testing.T) {
		t.Parallel()

		mock := &mockOrgRepo{
			restoreFunc: func(ctx context.Context, id uuid.UUID) error {
				return nil
			},
		}

		service := application.NewOrganizationService(mock)

		err := service.RestoreOrganization(context.Background(), uuid.New())

		require.NoError(t, err)
	})
}

// ===============================================================================
// HARD DELETE ORGANIZATION TESTS
// ===============================================================================

func TestOrganizationService_HardDeleteOrganization(t *testing.T) {
	t.Parallel()

	t.Run("calls repository HardDelete with correct ID", func(t *testing.T) {
		t.Parallel()

		expectedID := uuid.New()
		var capturedID uuid.UUID
		mock := &mockOrgRepo{
			hardDeleteFunc: func(ctx context.Context, id uuid.UUID) error {
				capturedID = id
				return nil
			},
		}

		service := application.NewOrganizationService(mock)

		err := service.HardDeleteOrganization(context.Background(), expectedID)

		require.NoError(t, err)
		assert.Equal(t, expectedID, capturedID)
	})

	t.Run("translates not found error to ErrOrganizationNotFound", func(t *testing.T) {
		t.Parallel()

		mock := &mockOrgRepo{
			hardDeleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return sqlite.ErrRecordNotFound
			},
		}

		service := application.NewOrganizationService(mock)

		err := service.HardDeleteOrganization(context.Background(), uuid.New())

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrOrganizationNotFound)
	})

	t.Run("propagates other repository errors", func(t *testing.T) {
		t.Parallel()

		repoErr := errors.New("database connection failed")
		mock := &mockOrgRepo{
			hardDeleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return repoErr
			},
		}

		service := application.NewOrganizationService(mock)

		err := service.HardDeleteOrganization(context.Background(), uuid.New())

		require.Error(t, err)
		assert.ErrorIs(t, err, repoErr)
	})

	t.Run("successfully hard deletes organization", func(t *testing.T) {
		t.Parallel()

		mock := &mockOrgRepo{
			hardDeleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return nil
			},
		}

		service := application.NewOrganizationService(mock)

		err := service.HardDeleteOrganization(context.Background(), uuid.New())

		require.NoError(t, err)
	})
}

// ===============================================================================
// GET ORGANIZATION TESTS
// ===============================================================================

func TestOrganizationService_GetOrganization(t *testing.T) {
	t.Parallel()

	t.Run("calls repository FindByID with correct ID", func(t *testing.T) {
		t.Parallel()

		expectedID := uuid.New()
		var capturedID uuid.UUID
		mock := &mockOrgRepo{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
				capturedID = id
				org := createValidOrg()
				org.ID = id
				return org, nil
			},
		}

		service := application.NewOrganizationService(mock)

		_, err := service.GetOrganization(context.Background(), expectedID)

		require.NoError(t, err)
		assert.Equal(t, expectedID, capturedID)
	})

	t.Run("returns organization from repository", func(t *testing.T) {
		t.Parallel()

		expectedOrg := createValidOrg()
		expectedOrg.ID = uuid.New()
		mock := &mockOrgRepo{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
				return expectedOrg, nil
			},
		}

		service := application.NewOrganizationService(mock)

		org, err := service.GetOrganization(context.Background(), expectedOrg.ID)

		require.NoError(t, err)
		assert.Equal(t, expectedOrg, org)
	})

	t.Run("translates not found error to ErrOrganizationNotFound", func(t *testing.T) {
		t.Parallel()

		mock := &mockOrgRepo{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
				return nil, sqlite.ErrRecordNotFound
			},
		}

		service := application.NewOrganizationService(mock)

		org, err := service.GetOrganization(context.Background(), uuid.New())

		require.Error(t, err)
		assert.Nil(t, org)
		assert.ErrorIs(t, err, application.ErrOrganizationNotFound)
	})

	t.Run("propagates other repository errors", func(t *testing.T) {
		t.Parallel()

		repoErr := errors.New("database connection failed")
		mock := &mockOrgRepo{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
				return nil, repoErr
			},
		}

		service := application.NewOrganizationService(mock)

		org, err := service.GetOrganization(context.Background(), uuid.New())

		require.Error(t, err)
		assert.Nil(t, org)
		assert.ErrorIs(t, err, repoErr)
	})
}

// ===============================================================================
// GET ORGANIZATION INCLUDING DELETED TESTS
// ===============================================================================

func TestOrganizationService_GetOrganizationIncludingDeleted(t *testing.T) {
	t.Parallel()

	t.Run("calls repository FindByIDIncludingDeleted with correct ID", func(t *testing.T) {
		t.Parallel()

		expectedID := uuid.New()
		var capturedID uuid.UUID
		mock := &mockOrgRepo{
			findByIDIncludingDeletedFunc: func(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
				capturedID = id
				org := createValidOrg()
				org.ID = id
				return org, nil
			},
		}

		service := application.NewOrganizationService(mock)

		_, err := service.GetOrganizationIncludingDeleted(context.Background(), expectedID)

		require.NoError(t, err)
		assert.Equal(t, expectedID, capturedID)
	})

	t.Run("returns organization from repository", func(t *testing.T) {
		t.Parallel()

		expectedOrg := createValidOrg()
		expectedOrg.ID = uuid.New()
		mock := &mockOrgRepo{
			findByIDIncludingDeletedFunc: func(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
				return expectedOrg, nil
			},
		}

		service := application.NewOrganizationService(mock)

		org, err := service.GetOrganizationIncludingDeleted(context.Background(), expectedOrg.ID)

		require.NoError(t, err)
		assert.Equal(t, expectedOrg, org)
	})

	t.Run("returns soft-deleted organization", func(t *testing.T) {
		t.Parallel()

		deletedOrg := createValidOrg()
		deletedOrg.ID = uuid.New()
		deletedTime := time.Now().UTC()
		deletedOrg.DeletedAt = &deletedTime

		mock := &mockOrgRepo{
			findByIDIncludingDeletedFunc: func(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
				return deletedOrg, nil
			},
		}

		service := application.NewOrganizationService(mock)

		org, err := service.GetOrganizationIncludingDeleted(context.Background(), deletedOrg.ID)

		require.NoError(t, err)
		assert.NotNil(t, org)
		assert.NotNil(t, org.DeletedAt)
		assert.Equal(t, deletedOrg.ID, org.ID)
	})

	t.Run("translates not found error to ErrOrganizationNotFound", func(t *testing.T) {
		t.Parallel()

		mock := &mockOrgRepo{
			findByIDIncludingDeletedFunc: func(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
				return nil, sqlite.ErrRecordNotFound
			},
		}

		service := application.NewOrganizationService(mock)

		org, err := service.GetOrganizationIncludingDeleted(context.Background(), uuid.New())

		require.Error(t, err)
		assert.Nil(t, org)
		assert.ErrorIs(t, err, application.ErrOrganizationNotFound)
	})

	t.Run("propagates other repository errors", func(t *testing.T) {
		t.Parallel()

		repoErr := errors.New("database connection failed")
		mock := &mockOrgRepo{
			findByIDIncludingDeletedFunc: func(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
				return nil, repoErr
			},
		}

		service := application.NewOrganizationService(mock)

		org, err := service.GetOrganizationIncludingDeleted(context.Background(), uuid.New())

		require.Error(t, err)
		assert.Nil(t, org)
		assert.ErrorIs(t, err, repoErr)
	})
}

// ===============================================================================
// LIST ORGANIZATIONS TESTS
// ===============================================================================

func TestOrganizationService_ListOrganizations(t *testing.T) {
	t.Parallel()

	t.Run("returns organizations from repository", func(t *testing.T) {
		t.Parallel()

		expectedOrgs := []*domain.Organization{
			createValidOrg(),
			createValidOrg(),
		}
		expectedOrgs[0].ID = uuid.New()
		expectedOrgs[0].Name = "Org 1"
		expectedOrgs[1].ID = uuid.New()
		expectedOrgs[1].Name = "Org 2"

		mock := &mockOrgRepo{
			findAllFunc: func(ctx context.Context) ([]*domain.Organization, error) {
				return expectedOrgs, nil
			},
		}

		service := application.NewOrganizationService(mock)

		orgs, err := service.ListOrganizations(context.Background())

		require.NoError(t, err)
		assert.Len(t, orgs, 2)
		assert.Equal(t, expectedOrgs, orgs)
	})

	t.Run("returns empty slice when no organizations exist", func(t *testing.T) {
		t.Parallel()

		mock := &mockOrgRepo{
			findAllFunc: func(ctx context.Context) ([]*domain.Organization, error) {
				return []*domain.Organization{}, nil
			},
		}

		service := application.NewOrganizationService(mock)

		orgs, err := service.ListOrganizations(context.Background())

		require.NoError(t, err)
		assert.Empty(t, orgs)
		assert.NotNil(t, orgs) // Should be empty slice, not nil
	})

	t.Run("propagates repository errors", func(t *testing.T) {
		t.Parallel()

		repoErr := errors.New("database connection failed")
		mock := &mockOrgRepo{
			findAllFunc: func(ctx context.Context) ([]*domain.Organization, error) {
				return nil, repoErr
			},
		}

		service := application.NewOrganizationService(mock)

		orgs, err := service.ListOrganizations(context.Background())

		require.Error(t, err)
		assert.Nil(t, orgs)
		assert.ErrorIs(t, err, repoErr)
	})
}

// ===============================================================================
// LIST ORGANIZATIONS INCLUDING DELETED TESTS
// ===============================================================================

func TestOrganizationService_ListOrganizationsIncludingDeleted(t *testing.T) {
	t.Parallel()

	t.Run("returns organizations from repository", func(t *testing.T) {
		t.Parallel()

		expectedOrgs := []*domain.Organization{
			createValidOrg(),
			createValidOrg(),
		}
		expectedOrgs[0].ID = uuid.New()
		expectedOrgs[0].Name = "Org 1"
		expectedOrgs[1].ID = uuid.New()
		expectedOrgs[1].Name = "Org 2"

		mock := &mockOrgRepo{
			findAllIncludingDeletedFunc: func(ctx context.Context) ([]*domain.Organization, error) {
				return expectedOrgs, nil
			},
		}

		service := application.NewOrganizationService(mock)

		orgs, err := service.ListOrganizationsIncludingDeleted(context.Background())

		require.NoError(t, err)
		assert.Len(t, orgs, 2)
		assert.Equal(t, expectedOrgs, orgs)
	})

	t.Run("returns both active and deleted organizations", func(t *testing.T) {
		t.Parallel()

		activeOrg := createValidOrg()
		activeOrg.ID = uuid.New()
		activeOrg.Name = "Active Org"

		deletedOrg := createValidOrg()
		deletedOrg.ID = uuid.New()
		deletedOrg.Name = "Deleted Org"
		deletedTime := time.Now().UTC()
		deletedOrg.DeletedAt = &deletedTime

		expectedOrgs := []*domain.Organization{activeOrg, deletedOrg}

		mock := &mockOrgRepo{
			findAllIncludingDeletedFunc: func(ctx context.Context) ([]*domain.Organization, error) {
				return expectedOrgs, nil
			},
		}

		service := application.NewOrganizationService(mock)

		orgs, err := service.ListOrganizationsIncludingDeleted(context.Background())

		require.NoError(t, err)
		assert.Len(t, orgs, 2)

		// Verify both active and deleted orgs are returned
		var hasActive, hasDeleted bool
		for _, org := range orgs {
			if org.DeletedAt == nil {
				hasActive = true
			} else {
				hasDeleted = true
			}
		}
		assert.True(t, hasActive, "should have active organization")
		assert.True(t, hasDeleted, "should have deleted organization")
	})

	t.Run("returns empty slice when no organizations exist", func(t *testing.T) {
		t.Parallel()

		mock := &mockOrgRepo{
			findAllIncludingDeletedFunc: func(ctx context.Context) ([]*domain.Organization, error) {
				return []*domain.Organization{}, nil
			},
		}

		service := application.NewOrganizationService(mock)

		orgs, err := service.ListOrganizationsIncludingDeleted(context.Background())

		require.NoError(t, err)
		assert.Empty(t, orgs)
		assert.NotNil(t, orgs) // Should be empty slice, not nil
	})

	t.Run("propagates repository errors", func(t *testing.T) {
		t.Parallel()

		repoErr := errors.New("database connection failed")
		mock := &mockOrgRepo{
			findAllIncludingDeletedFunc: func(ctx context.Context) ([]*domain.Organization, error) {
				return nil, repoErr
			},
		}

		service := application.NewOrganizationService(mock)

		orgs, err := service.ListOrganizationsIncludingDeleted(context.Background())

		require.Error(t, err)
		assert.Nil(t, orgs)
		assert.ErrorIs(t, err, repoErr)
	})
}

// ===============================================================================
// BENCHMARK TESTS
// ===============================================================================

func BenchmarkOrganizationService_CreateOrganization(b *testing.B) {
	mock := &mockOrgRepo{
		createFunc: func(ctx context.Context, org *domain.Organization) error {
			return nil
		},
	}

	service := application.NewOrganizationService(mock)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		org := createValidOrg()
		_ = service.CreateOrganization(ctx, org)
	}
}

func BenchmarkOrganizationService_GetOrganization(b *testing.B) {
	testOrg := createValidOrg()
	testOrg.ID = uuid.New()

	mock := &mockOrgRepo{
		findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
			return testOrg, nil
		},
	}

	service := application.NewOrganizationService(mock)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetOrganization(ctx, testOrg.ID)
	}
}

func BenchmarkOrganizationService_ListOrganizations(b *testing.B) {
	orgs := make([]*domain.Organization, 100)
	for i := range orgs {
		orgs[i] = createValidOrg()
		orgs[i].ID = uuid.New()
	}

	mock := &mockOrgRepo{
		findAllFunc: func(ctx context.Context) ([]*domain.Organization, error) {
			return orgs, nil
		},
	}

	service := application.NewOrganizationService(mock)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.ListOrganizations(ctx)
	}
}
