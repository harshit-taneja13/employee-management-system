package services

import (
	"context"
	"employee-management/models"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEmployeeRepo implements the EmployeeRepository interface for testing
type MockEmployeeRepo struct {
    mock.Mock
}

func (m *MockEmployeeRepo) FindAll(ctx context.Context, page, limit int) ([]models.Employee, int64, error) {
    args := m.Called(ctx, page, limit)
    return args.Get(0).([]models.Employee), args.Get(1).(int64), args.Error(2)
}

func (m *MockEmployeeRepo) FindByID(ctx context.Context, id string) (models.Employee, error) {
    args := m.Called(ctx, id)
    return args.Get(0).(models.Employee), args.Error(1)
}

func (m *MockEmployeeRepo) Create(ctx context.Context, employee models.Employee) (interface{}, error) {
    args := m.Called(ctx, employee)
    return args.Get(0), args.Error(1)
}

func (m *MockEmployeeRepo) Update(ctx context.Context, id string, employee models.Employee) (int64, error) {
    args := m.Called(ctx, id, employee)
    return args.Get(0).(int64), args.Error(1)
}

func (m *MockEmployeeRepo) Delete(ctx context.Context, id string) (int64, error) {
    args := m.Called(ctx, id)
    return args.Get(0).(int64), args.Error(1)
}

func (m *MockEmployeeRepo) DeleteAll(ctx context.Context) (int64, error) {
    args := m.Called(ctx)
    return args.Get(0).(int64), args.Error(1)
}

func (m *MockEmployeeRepo) CheckEmailExists(ctx context.Context, email string, excludeID ...string) (bool, error) {
    args := m.Called(ctx, email, excludeID)
    return args.Bool(0), args.Error(1)
}

func (m *MockEmployeeRepo) Ping(ctx context.Context) error {
    args := m.Called(ctx)
    return args.Error(0)
}

func TestCreateEmployee(t *testing.T){
	// setup
	mockRepo := new(MockEmployeeRepo)
	service := NewEmployeeService(mockRepo)
	ctx := context.Background()

	// test cases
	testCases := [] struct {
		name string 
		employee models.Employee
		emailExists bool
		expectedError bool
		setupMock func()
	}{
		{
			name: "Valid employee",
            employee: models.Employee{
                Name:       "John Doe",
                Department: "Engineering",
                Age:        30,
                Email:      "john.doe@example.com",
            },
            emailExists:   false,
            expectedError: false,
            setupMock: func() {
                mockRepo.On("CheckEmailExists", ctx, "john.doe@example.com", []string(nil)).Return(false, nil)
                mockRepo.On("Create", ctx, mock.AnythingOfType("models.Employee")).Return("created-id", nil)
            },
		},
		{
			name: "Email already exists",
            employee: models.Employee{
                Name:       "Jane Doe",
                Department: "HR",
                Age:        28,
                Email:      "jane.doe@example.com",
            },
            emailExists:   true,
            expectedError: true,
            setupMock: func() {
                mockRepo.On("CheckEmailExists", ctx, "jane.doe@example.com", []string(nil)).Return(true, nil)
            },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// reset mock between tests
			mockRepo = new(MockEmployeeRepo)
			service = NewEmployeeService(mockRepo)

			// setup mocks
			tc.setupMock()

			// execute
			result, err := service.CreateEmployee(ctx, tc.employee)

			// assert
			if tc.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			}else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			// verify all expectations were met
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestValidateEmployee(t *testing.T) {
	service := NewEmployeeService(nil)

	testCases := [] struct {
		name string
		employee models.Employee
		expectedError bool
		errorMessage string
	}{
		{
			name : "Valid employee",
			employee: models.Employee{
				Name: "John Doe",
				Department: "Engineering",
				Age : 30,
				Email: "john.doe@example.com",
			},
			expectedError: false,
		},{
			name: "Empty name",
            employee: models.Employee{
                Name:       "",
                Department: "Engineering",
                Age:        30,
                Email:      "john.doe@example.com",
            },
            expectedError: true,
            errorMessage:  "name is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.ValidateEmployee(tc.employee)

			if tc.expectedError {
				assert.Error(t, err)
				assert.Equal(t, tc.errorMessage, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test for getAllEmployees method
func TestGetAllEmployees(t *testing.T){
	mockRepo := new(MockEmployeeRepo)
	service := NewEmployeeService(mockRepo)
	ctx := context.Background()

	employees := []models.Employee{
        {
            Name:       "John Doe",
            Department: "Engineering",
            Age:        30,
            Email:      "john.doe@example.com",
        },
        {
            Name:       "Jane Smith",
            Department: "HR",
            Age:        28,
            Email:      "jane.smith@example.com",
        },
    }

	testCases := [] struct {
		name string
		page int
		limit int
		mockSetup func()
		expectedResult []models.Employee
		expectedCount int64
		expectedError bool
	}{
		{
            name:  "Success case",
            page:  1,
            limit: 10,
            mockSetup: func() {
                mockRepo.On("FindAll", ctx, 1, 10).Return(employees, int64(2), nil)
            },
            expectedResult: employees,
            expectedCount:  2,
            expectedError:  false,
        },
        {
            name:  "Empty result",
            page:  1,
            limit: 10,
            mockSetup: func() {
                mockRepo.On("FindAll", ctx, 1, 10).Return([]models.Employee{}, int64(0), nil)
            },
            expectedResult: []models.Employee{},
            expectedCount:  0,
            expectedError:  false,
        },
		{
            name:  "Database error",
            page:  1,
            limit: 10,
            mockSetup: func() {
                mockRepo.On("FindAll", ctx, 1, 10).Return([]models.Employee{}, int64(0), errors.New("database error"))
            },
            expectedResult: nil,
            expectedCount:  0,
            expectedError:  true,
        },
	}

	for _, tc := range testCases {
		t.Run(tc.name, func (t *testing.T) {
			// reset mock between tests
			mockRepo = new(MockEmployeeRepo)
			service = NewEmployeeService(mockRepo)

			// setup mocks
			tc.mockSetup()

			// execute
			result, count, err := service.GetAllEmployees(ctx, tc.page, tc.limit)
			
			// Assert
			if tc.expectedError {
				assert.Error(t, err)
			} else{
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedCount, count)
				assert.Equal(t, tc.expectedResult, result)
			}

			// verifying if all expectations were met
			mockRepo.AssertExpectations(t)
		})
	}
}

// Test for GetEmployeeByID method
func TestGetEmployeeByID(t *testing.T) {
	mockRepo := new(MockEmployeeRepo)
    service := NewEmployeeService(mockRepo)
    ctx := context.Background()
    
    employee := models.Employee{
        Name:       "John Doe",
        Department: "Engineering",
        Age:        30,
        Email:      "john.doe@example.com",
    }

	testCases := [] struct {
        name           string
        id             string
        mockSetup      func()
        expectedResult models.Employee
        expectedError  bool
    }{
        {
            name: "Existing employee",
            id:   "existing-id",
            mockSetup: func() {
                mockRepo.On("FindByID", ctx, "existing-id").Return(employee, nil)
            },
            expectedResult: employee,
            expectedError:  false,
        },
        {
            name: "Non-existing employee",
            id:   "non-existing-id",
            mockSetup: func() {
                mockRepo.On("FindByID", ctx, "non-existing-id").Return(models.Employee{}, errors.New("employee not found"))
            },
            expectedResult: models.Employee{},
            expectedError:  true,
        },
    }

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset mock between tests
            mockRepo = new(MockEmployeeRepo)
            service = NewEmployeeService(mockRepo)
            
            // Setup mocks
            tc.mockSetup()
            
            // Execute
            result, err := service.GetEmployeeByID(ctx, tc.id)
            
            // Assert
            if tc.expectedError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tc.expectedResult, result)
            }
            
            // Verify all expectations were met
            mockRepo.AssertExpectations(t)
		})
	}
}

// Test for UpdateEmployee method
func TestUpdateEmployee(t *testing.T) {
    mockRepo := new(MockEmployeeRepo)
    service := NewEmployeeService(mockRepo)
    ctx := context.Background()
    
	testCases := []struct {
        name           string
        id             string
        employee       models.Employee
        mockSetup      func()
        expectedCount  int64
        expectedError  bool
    }{
		{
            name: "Successful update",
            id:   "existing-id",
            employee: models.Employee{
                Name:       "Updated Name",
                Department: "Updated Department",
                Age:        35,
                Email:      "updated@example.com",
            },
            mockSetup: func() {
                mockRepo.On("CheckEmailExists", ctx, "updated@example.com", []string{"existing-id"}).Return(false, nil)
                mockRepo.On("Update", ctx, "existing-id", mock.AnythingOfType("models.Employee")).Return(int64(1), nil)
            },
            expectedCount: 1,
            expectedError: false,
        },
        {
            name: "Email already exists",
            id:   "employee-id",
            employee: models.Employee{
                Name:       "John Doe",
                Department: "Engineering",
                Age:        30,
                Email:      "existing@example.com",
            },
            mockSetup: func() {
                mockRepo.On("CheckEmailExists", ctx, "existing@example.com", []string{"employee-id"}).Return(true, nil)
            },
            expectedCount: 0,
            expectedError: true,
        },
		{
            name: "No changes",
            id:   "existing-id",
            employee: models.Employee{
                Name:       "John Doe",
                Department: "Engineering",
                Age:        30,
                Email:      "john.doe@example.com",
            },
            mockSetup: func() {
                mockRepo.On("CheckEmailExists", ctx, "john.doe@example.com", []string{"existing-id"}).Return(false, nil)
                mockRepo.On("Update", ctx, "existing-id", mock.AnythingOfType("models.Employee")).Return(int64(0), nil)
            },
            expectedCount: 0,
            expectedError: false,
        },
	}
	
	for _, tc := range testCases {
        t.Run(tc.name , func(t *testing.T) {
            // Reset mock between tests
            mockRepo = new(MockEmployeeRepo)
            service = NewEmployeeService(mockRepo)
            
            // Setup mocks
            tc.mockSetup()
            
            // Execute
            count, err := service.UpdateEmployee(ctx, tc.id, tc.employee)

            // Assert 
            if tc.expectedError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tc.expectedCount, count)
            }   

            // Verify all expectations were met
            mockRepo.AssertExpectations(t)
        })  
    }
}

// Test for DeleteEmployee method
func TestDeleteEmployee(t *testing.T) {
    mockRepo := new(MockEmployeeRepo)
    service := NewEmployeeService(mockRepo)
    ctx := context.Background()

    testCases := []struct {
        name           string
        id             string
        mockSetup      func()
        expectedCount  int64
        expectedError  bool
    }{
        {
            name: "Successful deletion",
            id:   "existing-id",
            mockSetup: func() {
                mockRepo.On("Delete", ctx, "existing-id").Return(int64(1), nil)
            },
            expectedCount: 1,
            expectedError: false,
        },
        {
            name: "Employee not found",
            id:   "non-existing-id",
            mockSetup: func() {
                mockRepo.On("Delete", ctx, "non-existing-id").Return(int64(0), nil)
            },
            expectedCount: 0,
            expectedError: false,
        },
        {
            name: "Database error",
            id:   "error-id",
            mockSetup: func() {
                mockRepo.On("Delete", ctx, "error-id").Return(int64(0), errors.New("database error"))
            },
            expectedCount: 0,
            expectedError: true,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name , func(t *testing.T) {
            // Reset mock between tests
            mockRepo = new(MockEmployeeRepo)
            service = NewEmployeeService(mockRepo)
            
            // Setup mocks
            tc.mockSetup()

            // Execute
            count, err := service.DeleteEmployee(ctx, tc.id)

            // Assert
            if tc.expectedError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tc.expectedCount, count)
            }

            // Verify all expectations were met
            mockRepo.AssertExpectations(t)
        })
    }
}

// Test for DeleteAllEmployees method
func TestDeleteAllEmployees(t *testing.T) {
    mockRepo := new(MockEmployeeRepo)
    service := NewEmployeeService(mockRepo)
    ctx := context.Background()

    testCases := []struct {
        name           string
        mockSetup      func()
        expectedCount  int64
        expectedError  bool
    }{
        {
            name: "Successful deletion of all employees",
            mockSetup: func() {
                mockRepo.On("DeleteAll", ctx).Return(int64(5), nil)
            },
            expectedCount: 5,
            expectedError: false,
        },
        {
            name: "No employees to delete",
            mockSetup: func() {
                mockRepo.On("DeleteAll", ctx).Return(int64(0), nil)
            },
            expectedCount: 0,
            expectedError: false,
        },
        {
            name: "Database error",
            mockSetup: func() {
                mockRepo.On("DeleteAll", ctx).Return(int64(0), errors.New("database error"))
            },
            expectedCount: 0,
            expectedError: true,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Reset mock between tests
            mockRepo = new(MockEmployeeRepo)
            service = NewEmployeeService(mockRepo)
            
            // Setup mocks
            tc.mockSetup()
            
            // Execute
            count, err := service.DeleteAllEmployees(ctx)

             // Assert
             if tc.expectedError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tc.expectedCount, count)
            }
            
            // Verify all expectations were met
            mockRepo.AssertExpectations(t)
        })
    }
}


// Test for HealthCheck method
func TestHealthCheck(t *testing.T) {
    mockRepo := new(MockEmployeeRepo)
    service := NewEmployeeService(mockRepo)
    ctx := context.Background()
    
    testCases := []struct {
        name          string
        mockSetup     func()
        expectedError bool
    }{
        {
            name: "Database is healthy",
            mockSetup: func() {
                mockRepo.On("Ping", ctx).Return(nil)
            },
            expectedError: false,
        },
        {
            name: "Database is not available",
            mockSetup: func() {
                mockRepo.On("Ping", ctx).Return(errors.New("connection refused"))
            },
            expectedError: true,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Reset mock between tests
            mockRepo = new(MockEmployeeRepo)
            service = NewEmployeeService(mockRepo)
            
            // Setup mocks
            tc.mockSetup()
            
            // Execute
            err := service.HealthCheck(ctx)
            
            // Assert
            if tc.expectedError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
            
            // Verify all expectations were met
            mockRepo.AssertExpectations(t)
        })
    }

}
