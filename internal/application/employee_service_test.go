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

type mockEmployeeRepository struct {
	createFunc                             func(context.Context, *domain.Employee) error
	updateFunc                             func(context.Context, *domain.Employee) error
	deleteFunc                             func(context.Context, uuid.UUID) error
	restoreFunc                            func(context.Context, uuid.UUID) error
	hardDeleteFunc                         func(context.Context, uuid.UUID) error
	findByIDFunc                           func(context.Context, uuid.UUID) (*domain.Employee, error)
	findByIDIncludingDeletedFunc           func(context.Context, uuid.UUID) (*domain.Employee, error)
	findAllFunc                            func(context.Context) ([]*domain.Employee, error)
	findAllIncludingDeletedFunc            func(context.Context) ([]*domain.Employee, error)
	getNextSerialNumberFunc                func(context.Context, uuid.UUID) (int, error)
	findByOrgAndSerialNumFunc              func(context.Context, uuid.UUID, int) (*domain.Employee, error)
	findByOrganizationFunc                 func(context.Context, uuid.UUID) ([]*domain.Employee, error)
	findByOrganizationIncludingDeletedFunc func(context.Context, uuid.UUID) ([]*domain.Employee, error)
}

func (m *mockEmployeeRepository) Create(ctx context.Context, emp *domain.Employee) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, emp)
	}
	return nil
}

func (m *mockEmployeeRepository) Update(ctx context.Context, emp *domain.Employee) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, emp)
	}
	return nil
}

func (m *mockEmployeeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockEmployeeRepository) Restore(ctx context.Context, id uuid.UUID) error {
	if m.restoreFunc != nil {
		return m.restoreFunc(ctx, id)
	}
	return nil
}

func (m *mockEmployeeRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	if m.hardDeleteFunc != nil {
		return m.hardDeleteFunc(ctx, id)
	}
	return nil
}

func (m *mockEmployeeRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Employee, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, sqlite.ErrRecordNotFound
}

func (m *mockEmployeeRepository) FindByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*domain.Employee, error) {
	if m.findByIDIncludingDeletedFunc != nil {
		return m.findByIDIncludingDeletedFunc(ctx, id)
	}
	return nil, sqlite.ErrRecordNotFound
}

func (m *mockEmployeeRepository) FindAll(ctx context.Context) ([]*domain.Employee, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc(ctx)
	}
	return []*domain.Employee{}, nil
}

func (m *mockEmployeeRepository) FindAllIncludingDeleted(ctx context.Context) ([]*domain.Employee, error) {
	if m.findAllIncludingDeletedFunc != nil {
		return m.findAllIncludingDeletedFunc(ctx)
	}
	return []*domain.Employee{}, nil
}

func (m *mockEmployeeRepository) GetNextSerialNumber(ctx context.Context, orgID uuid.UUID) (int, error) {
	if m.getNextSerialNumberFunc != nil {
		return m.getNextSerialNumberFunc(ctx, orgID)
	}
	return 1, nil
}

func (m *mockEmployeeRepository) FindByOrgAndSerialNum(ctx context.Context, orgID uuid.UUID, serialNum int) (*domain.Employee, error) {
	if m.findByOrgAndSerialNumFunc != nil {
		return m.findByOrgAndSerialNumFunc(ctx, orgID, serialNum)
	}
	return nil, sqlite.ErrRecordNotFound
}

func (m *mockEmployeeRepository) FindByOrganization(ctx context.Context, orgID uuid.UUID) ([]*domain.Employee, error) {
	if m.findByOrganizationFunc != nil {
		return m.findByOrganizationFunc(ctx, orgID)
	}
	return []*domain.Employee{}, nil
}

func (m *mockEmployeeRepository) FindByOrganizationIncludingDeleted(ctx context.Context, orgID uuid.UUID) ([]*domain.Employee, error) {
	if m.findByOrganizationIncludingDeletedFunc != nil {
		return m.findByOrganizationIncludingDeletedFunc(ctx, orgID)
	}
	return []*domain.Employee{}, nil
}

// ===============================================================================
// TEST HELPERS
// ===============================================================================

func createTestEmployee(orgID, compPackID uuid.UUID) *domain.Employee {
	return &domain.Employee{
		ID:                    uuid.New(),
		OrgID:                 orgID,
		FullName:              "Ahmed Ali",
		DisplayName:           "Ahmed",
		Address:               "123 Main St, Casablanca",
		EmailAddress:          "ahmed@example.com",
		PhoneNumber:           "+212612345678",
		BirthDate:             time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		Gender:                domain.GenderMale,
		MaritalStatus:         domain.MaritalStatusSingle,
		NumDependents:         0,
		NumKids:               0,
		CINNum:                "AB123456",
		CNSSNum:               "123456789",
		HireDate:              time.Now().UTC().AddDate(0, -1, 0), // Hired 1 month ago
		Position:              "Software Engineer",
		CompensationPackageID: compPackID,
		BankRIB:               "123456789012345678901234",
	}
}

// ===============================================================================
// CREATE TESTS
// ===============================================================================

func TestEmployeeService_CreateEmployee(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	compPackID := uuid.New()

	tests := []struct {
		name          string
		employee      *domain.Employee
		mockRepo      *mockEmployeeRepository
		wantErr       bool
		expectedError error
		validate      func(*testing.T, *domain.Employee)
	}{
		{
			name:     "creates employee successfully",
			employee: createTestEmployee(orgID, compPackID),
			mockRepo: &mockEmployeeRepository{
				getNextSerialNumberFunc: func(ctx context.Context, id uuid.UUID) (int, error) {
					assert.Equal(t, orgID, id)
					return 1, nil
				},
				createFunc: func(ctx context.Context, emp *domain.Employee) error {
					assert.NotEqual(t, uuid.Nil, emp.ID)
					assert.Equal(t, 1, emp.SerialNum)
					assert.NotZero(t, emp.CreatedAt)
					assert.NotZero(t, emp.UpdatedAt)
					assert.Nil(t, emp.DeletedAt)
					return nil
				},
			},
			wantErr: false,
			validate: func(t *testing.T, emp *domain.Employee) {
				assert.NotEqual(t, uuid.Nil, emp.ID)
				assert.Equal(t, 1, emp.SerialNum)
				assert.NotZero(t, emp.CreatedAt)
				assert.NotZero(t, emp.UpdatedAt)
				assert.Nil(t, emp.DeletedAt)
			},
		},
		{
			name: "generates UUID if not provided",
			employee: func() *domain.Employee {
				emp := createTestEmployee(orgID, compPackID)
				emp.ID = uuid.Nil // Explicitly set to Nil
				return emp
			}(),
			mockRepo: &mockEmployeeRepository{
				getNextSerialNumberFunc: func(ctx context.Context, id uuid.UUID) (int, error) {
					return 1, nil
				},
				createFunc: func(ctx context.Context, emp *domain.Employee) error {
					return nil
				},
			},
			wantErr: false,
			validate: func(t *testing.T, emp *domain.Employee) {
				assert.NotEqual(t, uuid.Nil, emp.ID, "should generate UUID")
			},
		},
		{
			name:     "generates sequential serial numbers",
			employee: createTestEmployee(orgID, compPackID),
			mockRepo: &mockEmployeeRepository{
				getNextSerialNumberFunc: func(ctx context.Context, id uuid.UUID) (int, error) {
					return 42, nil
				},
				createFunc: func(ctx context.Context, emp *domain.Employee) error {
					return nil
				},
			},
			wantErr: false,
			validate: func(t *testing.T, emp *domain.Employee) {
				assert.Equal(t, 42, emp.SerialNum)
			},
		},
		{
			name:     "returns error when serial number generation fails",
			employee: createTestEmployee(orgID, compPackID),
			mockRepo: &mockEmployeeRepository{
				getNextSerialNumberFunc: func(ctx context.Context, id uuid.UUID) (int, error) {
					return 0, errors.New("database error")
				},
			},
			wantErr:       true,
			expectedError: nil, // Just check that error occurs
		},
		{
			name: "returns error for invalid employee data",
			employee: func() *domain.Employee {
				emp := createTestEmployee(orgID, compPackID)
				emp.FullName = "" // Invalid
				return emp
			}(),
			mockRepo: &mockEmployeeRepository{
				getNextSerialNumberFunc: func(ctx context.Context, id uuid.UUID) (int, error) {
					return 1, nil
				},
			},
			wantErr:       true,
			expectedError: domain.ErrEmployeeFullNameRequired,
		},
		{
			name:     "returns ErrEmployeeExists when CIN/CNSS duplicates",
			employee: createTestEmployee(orgID, compPackID),
			mockRepo: &mockEmployeeRepository{
				getNextSerialNumberFunc: func(ctx context.Context, id uuid.UUID) (int, error) {
					return 1, nil
				},
				createFunc: func(ctx context.Context, emp *domain.Employee) error {
					return sqlite.ErrDuplicateRecord
				},
			},
			wantErr:       true,
			expectedError: application.ErrEmployeeExists,
		},
		{
			name:     "propagates repository errors",
			employee: createTestEmployee(orgID, compPackID),
			mockRepo: &mockEmployeeRepository{
				getNextSerialNumberFunc: func(ctx context.Context, id uuid.UUID) (int, error) {
					return 1, nil
				},
				createFunc: func(ctx context.Context, emp *domain.Employee) error {
					return errors.New("database error")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := application.NewEmployeeService(tt.mockRepo)
			err := service.CreateEmployee(context.Background(), tt.employee)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError)
				}
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, tt.employee)
				}
			}
		})
	}
}

// ===============================================================================
// UPDATE TESTS
// ===============================================================================

func TestEmployeeService_UpdateEmployee(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	compPackID := uuid.New()

	tests := []struct {
		name          string
		employee      *domain.Employee
		mockRepo      *mockEmployeeRepository
		wantErr       bool
		expectedError error
		validate      func(*testing.T, *domain.Employee)
	}{
		{
			name: "updates employee successfully",
			employee: func() *domain.Employee {
				emp := createTestEmployee(orgID, compPackID)
				emp.ID = uuid.New()
				emp.SerialNum = 1
				emp.CreatedAt = time.Now().UTC().Add(-24 * time.Hour)
				return emp
			}(),
			mockRepo: &mockEmployeeRepository{
				updateFunc: func(ctx context.Context, emp *domain.Employee) error {
					assert.NotZero(t, emp.UpdatedAt)
					return nil
				},
			},
			wantErr: false,
			validate: func(t *testing.T, emp *domain.Employee) {
				assert.NotZero(t, emp.UpdatedAt)
			},
		},
		{
			name: "returns error for invalid employee data",
			employee: func() *domain.Employee {
				emp := createTestEmployee(orgID, compPackID)
				emp.ID = uuid.New()
				emp.SerialNum = 1
				emp.FullName = "" // Invalid
				return emp
			}(),
			mockRepo:      &mockEmployeeRepository{},
			wantErr:       true,
			expectedError: domain.ErrEmployeeFullNameRequired,
		},
		{
			name: "returns ErrEmployeeNotFound when employee doesn't exist",
			employee: func() *domain.Employee {
				emp := createTestEmployee(orgID, compPackID)
				emp.ID = uuid.New()
				emp.SerialNum = 1
				return emp
			}(),
			mockRepo: &mockEmployeeRepository{
				updateFunc: func(ctx context.Context, emp *domain.Employee) error {
					return sqlite.ErrRecordNotFound
				},
			},
			wantErr:       true,
			expectedError: application.ErrEmployeeNotFound,
		},
		{
			name: "returns ErrEmployeeExists when CIN/CNSS conflicts",
			employee: func() *domain.Employee {
				emp := createTestEmployee(orgID, compPackID)
				emp.ID = uuid.New()
				emp.SerialNum = 1
				return emp
			}(),
			mockRepo: &mockEmployeeRepository{
				updateFunc: func(ctx context.Context, emp *domain.Employee) error {
					return sqlite.ErrDuplicateRecord
				},
			},
			wantErr:       true,
			expectedError: application.ErrEmployeeExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := application.NewEmployeeService(tt.mockRepo)
			err := service.UpdateEmployee(context.Background(), tt.employee)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError)
				}
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, tt.employee)
				}
			}
		})
	}
}

// ===============================================================================
// DELETE TESTS
// ===============================================================================

func TestEmployeeService_DeleteEmployee(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		id            uuid.UUID
		mockRepo      *mockEmployeeRepository
		wantErr       bool
		expectedError error
	}{
		{
			name: "deletes employee successfully",
			id:   uuid.New(),
			mockRepo: &mockEmployeeRepository{
				deleteFunc: func(ctx context.Context, id uuid.UUID) error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "returns ErrEmployeeNotFound when employee doesn't exist",
			id:   uuid.New(),
			mockRepo: &mockEmployeeRepository{
				deleteFunc: func(ctx context.Context, id uuid.UUID) error {
					return sqlite.ErrRecordNotFound
				},
			},
			wantErr:       true,
			expectedError: application.ErrEmployeeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := application.NewEmployeeService(tt.mockRepo)
			err := service.DeleteEmployee(context.Background(), tt.id)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEmployeeService_RestoreEmployee(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		id            uuid.UUID
		mockRepo      *mockEmployeeRepository
		wantErr       bool
		expectedError error
	}{
		{
			name: "restores employee successfully",
			id:   uuid.New(),
			mockRepo: &mockEmployeeRepository{
				restoreFunc: func(ctx context.Context, id uuid.UUID) error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "returns ErrEmployeeNotFound when employee doesn't exist",
			id:   uuid.New(),
			mockRepo: &mockEmployeeRepository{
				restoreFunc: func(ctx context.Context, id uuid.UUID) error {
					return sqlite.ErrRecordNotFound
				},
			},
			wantErr:       true,
			expectedError: application.ErrEmployeeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := application.NewEmployeeService(tt.mockRepo)
			err := service.RestoreEmployee(context.Background(), tt.id)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEmployeeService_HardDeleteEmployee(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		id            uuid.UUID
		mockRepo      *mockEmployeeRepository
		wantErr       bool
		expectedError error
	}{
		{
			name: "hard deletes employee successfully",
			id:   uuid.New(),
			mockRepo: &mockEmployeeRepository{
				hardDeleteFunc: func(ctx context.Context, id uuid.UUID) error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "returns ErrEmployeeNotFound when employee doesn't exist",
			id:   uuid.New(),
			mockRepo: &mockEmployeeRepository{
				hardDeleteFunc: func(ctx context.Context, id uuid.UUID) error {
					return sqlite.ErrRecordNotFound
				},
			},
			wantErr:       true,
			expectedError: application.ErrEmployeeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := application.NewEmployeeService(tt.mockRepo)
			err := service.HardDeleteEmployee(context.Background(), tt.id)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ===============================================================================
// QUERY TESTS
// ===============================================================================

func TestEmployeeService_GetEmployee(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	orgID := uuid.New()
	compPackID := uuid.New()
	employee := createTestEmployee(orgID, compPackID)
	employee.ID = id

	tests := []struct {
		name          string
		id            uuid.UUID
		mockRepo      *mockEmployeeRepository
		wantErr       bool
		expectedError error
		validate      func(*testing.T, *domain.Employee)
	}{
		{
			name: "gets employee successfully",
			id:   id,
			mockRepo: &mockEmployeeRepository{
				findByIDFunc: func(ctx context.Context, findID uuid.UUID) (*domain.Employee, error) {
					assert.Equal(t, id, findID)
					return employee, nil
				},
			},
			wantErr: false,
			validate: func(t *testing.T, emp *domain.Employee) {
				assert.Equal(t, id, emp.ID)
				assert.Equal(t, "Ahmed Ali", emp.FullName)
			},
		},
		{
			name: "returns ErrEmployeeNotFound when not found",
			id:   id,
			mockRepo: &mockEmployeeRepository{
				findByIDFunc: func(ctx context.Context, findID uuid.UUID) (*domain.Employee, error) {
					return nil, sqlite.ErrRecordNotFound
				},
			},
			wantErr:       true,
			expectedError: application.ErrEmployeeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := application.NewEmployeeService(tt.mockRepo)
			emp, err := service.GetEmployee(context.Background(), tt.id)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError)
				}
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, emp)
				}
			}
		})
	}
}

func TestEmployeeService_GetEmployeeIncludingDeleted(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	orgID := uuid.New()
	compPackID := uuid.New()
	employee := createTestEmployee(orgID, compPackID)
	employee.ID = id
	deletedAt := time.Now().UTC()
	employee.DeletedAt = &deletedAt

	tests := []struct {
		name          string
		id            uuid.UUID
		mockRepo      *mockEmployeeRepository
		wantErr       bool
		expectedError error
		validate      func(*testing.T, *domain.Employee)
	}{
		{
			name: "gets deleted employee successfully",
			id:   id,
			mockRepo: &mockEmployeeRepository{
				findByIDIncludingDeletedFunc: func(ctx context.Context, findID uuid.UUID) (*domain.Employee, error) {
					assert.Equal(t, id, findID)
					return employee, nil
				},
			},
			wantErr: false,
			validate: func(t *testing.T, emp *domain.Employee) {
				assert.Equal(t, id, emp.ID)
				assert.NotNil(t, emp.DeletedAt)
			},
		},
		{
			name: "returns ErrEmployeeNotFound when not found",
			id:   id,
			mockRepo: &mockEmployeeRepository{
				findByIDIncludingDeletedFunc: func(ctx context.Context, findID uuid.UUID) (*domain.Employee, error) {
					return nil, sqlite.ErrRecordNotFound
				},
			},
			wantErr:       true,
			expectedError: application.ErrEmployeeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := application.NewEmployeeService(tt.mockRepo)
			emp, err := service.GetEmployeeIncludingDeleted(context.Background(), tt.id)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError)
				}
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, emp)
				}
			}
		})
	}
}

func TestEmployeeService_ListEmployees(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	compPackID := uuid.New()
	employees := []*domain.Employee{
		createTestEmployee(orgID, compPackID),
		createTestEmployee(orgID, compPackID),
	}

	tests := []struct {
		name     string
		mockRepo *mockEmployeeRepository
		wantErr  bool
		validate func(*testing.T, []*domain.Employee)
	}{
		{
			name: "lists employees successfully",
			mockRepo: &mockEmployeeRepository{
				findAllFunc: func(ctx context.Context) ([]*domain.Employee, error) {
					return employees, nil
				},
			},
			wantErr: false,
			validate: func(t *testing.T, emps []*domain.Employee) {
				assert.Len(t, emps, 2)
			},
		},
		{
			name: "returns empty slice when no employees",
			mockRepo: &mockEmployeeRepository{
				findAllFunc: func(ctx context.Context) ([]*domain.Employee, error) {
					return []*domain.Employee{}, nil
				},
			},
			wantErr: false,
			validate: func(t *testing.T, emps []*domain.Employee) {
				assert.Empty(t, emps)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := application.NewEmployeeService(tt.mockRepo)
			emps, err := service.ListEmployees(context.Background())

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, emps)
				}
			}
		})
	}
}

func TestEmployeeService_ListEmployeesIncludingDeleted(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	compPackID := uuid.New()
	employees := []*domain.Employee{
		createTestEmployee(orgID, compPackID),
		func() *domain.Employee {
			emp := createTestEmployee(orgID, compPackID)
			deletedAt := time.Now().UTC()
			emp.DeletedAt = &deletedAt
			return emp
		}(),
	}

	tests := []struct {
		name     string
		mockRepo *mockEmployeeRepository
		wantErr  bool
		validate func(*testing.T, []*domain.Employee)
	}{
		{
			name: "lists all employees including deleted",
			mockRepo: &mockEmployeeRepository{
				findAllIncludingDeletedFunc: func(ctx context.Context) ([]*domain.Employee, error) {
					return employees, nil
				},
			},
			wantErr: false,
			validate: func(t *testing.T, emps []*domain.Employee) {
				assert.Len(t, emps, 2)
				deletedCount := 0
				for _, emp := range emps {
					if emp.DeletedAt != nil {
						deletedCount++
					}
				}
				assert.Equal(t, 1, deletedCount)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := application.NewEmployeeService(tt.mockRepo)
			emps, err := service.ListEmployeesIncludingDeleted(context.Background())

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, emps)
				}
			}
		})
	}
}

// ===============================================================================
// EMPLOYEE-SPECIFIC QUERY TESTS
// ===============================================================================

func TestEmployeeService_GetEmployeeByOrgAndSerialNum(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	compPackID := uuid.New()
	employee := createTestEmployee(orgID, compPackID)
	employee.SerialNum = 5

	tests := []struct {
		name          string
		orgID         uuid.UUID
		serialNum     int
		mockRepo      *mockEmployeeRepository
		wantErr       bool
		expectedError error
		validate      func(*testing.T, *domain.Employee)
	}{
		{
			name:      "gets employee by org and serial num successfully",
			orgID:     orgID,
			serialNum: 5,
			mockRepo: &mockEmployeeRepository{
				findByOrgAndSerialNumFunc: func(ctx context.Context, findOrgID uuid.UUID, findSerialNum int) (*domain.Employee, error) {
					assert.Equal(t, orgID, findOrgID)
					assert.Equal(t, 5, findSerialNum)
					return employee, nil
				},
			},
			wantErr: false,
			validate: func(t *testing.T, emp *domain.Employee) {
				assert.Equal(t, 5, emp.SerialNum)
				assert.Equal(t, orgID, emp.OrgID)
			},
		},
		{
			name:      "returns ErrEmployeeNotFound when not found",
			orgID:     orgID,
			serialNum: 99,
			mockRepo: &mockEmployeeRepository{
				findByOrgAndSerialNumFunc: func(ctx context.Context, findOrgID uuid.UUID, findSerialNum int) (*domain.Employee, error) {
					return nil, sqlite.ErrRecordNotFound
				},
			},
			wantErr:       true,
			expectedError: application.ErrEmployeeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := application.NewEmployeeService(tt.mockRepo)
			emp, err := service.GetEmployeeByOrgAndSerialNum(context.Background(), tt.orgID, tt.serialNum)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError)
				}
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, emp)
				}
			}
		})
	}
}

func TestEmployeeService_ListEmployeesByOrganization(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	compPackID := uuid.New()
	employees := []*domain.Employee{
		func() *domain.Employee {
			emp := createTestEmployee(orgID, compPackID)
			emp.SerialNum = 1
			return emp
		}(),
		func() *domain.Employee {
			emp := createTestEmployee(orgID, compPackID)
			emp.SerialNum = 2
			return emp
		}(),
	}

	tests := []struct {
		name     string
		orgID    uuid.UUID
		mockRepo *mockEmployeeRepository
		wantErr  bool
		validate func(*testing.T, []*domain.Employee)
	}{
		{
			name:  "lists employees by organization successfully",
			orgID: orgID,
			mockRepo: &mockEmployeeRepository{
				findByOrganizationFunc: func(ctx context.Context, findOrgID uuid.UUID) ([]*domain.Employee, error) {
					assert.Equal(t, orgID, findOrgID)
					return employees, nil
				},
			},
			wantErr: false,
			validate: func(t *testing.T, emps []*domain.Employee) {
				assert.Len(t, emps, 2)
				for _, emp := range emps {
					assert.Equal(t, orgID, emp.OrgID)
				}
			},
		},
		{
			name:  "returns empty slice when organization has no employees",
			orgID: orgID,
			mockRepo: &mockEmployeeRepository{
				findByOrganizationFunc: func(ctx context.Context, findOrgID uuid.UUID) ([]*domain.Employee, error) {
					return []*domain.Employee{}, nil
				},
			},
			wantErr: false,
			validate: func(t *testing.T, emps []*domain.Employee) {
				assert.Empty(t, emps)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := application.NewEmployeeService(tt.mockRepo)
			emps, err := service.ListEmployeesByOrganization(context.Background(), tt.orgID)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, emps)
				}
			}
		})
	}
}

func TestEmployeeService_ListEmployeesByOrganizationIncludingDeleted(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	compPackID := uuid.New()
	employees := []*domain.Employee{
		createTestEmployee(orgID, compPackID),
		func() *domain.Employee {
			emp := createTestEmployee(orgID, compPackID)
			deletedAt := time.Now().UTC()
			emp.DeletedAt = &deletedAt
			return emp
		}(),
	}

	tests := []struct {
		name     string
		orgID    uuid.UUID
		mockRepo *mockEmployeeRepository
		wantErr  bool
		validate func(*testing.T, []*domain.Employee)
	}{
		{
			name:  "lists all employees including deleted by organization",
			orgID: orgID,
			mockRepo: &mockEmployeeRepository{
				findByOrganizationIncludingDeletedFunc: func(ctx context.Context, findOrgID uuid.UUID) ([]*domain.Employee, error) {
					assert.Equal(t, orgID, findOrgID)
					return employees, nil
				},
			},
			wantErr: false,
			validate: func(t *testing.T, emps []*domain.Employee) {
				assert.Len(t, emps, 2)
				deletedCount := 0
				for _, emp := range emps {
					assert.Equal(t, orgID, emp.OrgID)
					if emp.DeletedAt != nil {
						deletedCount++
					}
				}
				assert.Equal(t, 1, deletedCount)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := application.NewEmployeeService(tt.mockRepo)
			emps, err := service.ListEmployeesByOrganizationIncludingDeleted(context.Background(), tt.orgID)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, emps)
				}
			}
		})
	}
}
