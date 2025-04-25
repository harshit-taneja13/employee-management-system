package controllers

import (
	"bytes"
	"context"
	"employee-management/models"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// MockEmployeeService implements the necessary methods for testing controllers
type MockEmployeeService struct {
    mock.Mock
}

func (m *MockEmployeeService) CreateEmployee(ctx context.Context, emp models.Employee) (interface{}, error) {
    args := m.Called(ctx, emp)
    return args.Get(0), args.Error(1)
}

func (m *MockEmployeeService) GetAllEmployees(ctx context.Context, page, limit int) ([]models.Employee, int64, error) {
    args := m.Called(ctx, page, limit)
    return args.Get(0).([]models.Employee), args.Get(1).(int64), args.Error(2)
}

func (m *MockEmployeeService) GetEmployeeByID(ctx context.Context, id string) (models.Employee, error) {
    args := m.Called(ctx, id)
    return args.Get(0).(models.Employee), args.Error(1)
}

func (m *MockEmployeeService) UpdateEmployee(ctx context.Context, id string, emp models.Employee) (int64, error) {
    args := m.Called(ctx, id, emp)
    return args.Get(0).(int64), args.Error(1)
}

func (m *MockEmployeeService) DeleteEmployee(ctx context.Context, id string) (int64, error) {
    args := m.Called(ctx, id)
    return args.Get(0).(int64), args.Error(1)
}

func (m *MockEmployeeService) DeleteAllEmployees(ctx context.Context) (int64, error) {
    args := m.Called(ctx)
    return args.Get(0).(int64), args.Error(1)
}

func (m *MockEmployeeService) HealthCheck(ctx context.Context) error {
    args := m.Called(ctx)
    return args.Error(0)
}



func TestCreateEmployeeHandler(t *testing.T) {
    // Test cases
    testCases := []struct {
        name           string
        payload        interface{}
        setupMock      func(mock *MockEmployeeService)
        expectedStatus int
        expectedError  bool
    }{
		{
            name: "Valid employee",
            payload: models.Employee{
                Name:       "John Doe",
                Department: "Engineering",
                Age:        30,
                Email:      "john.doe@example.com",
            },
            setupMock: func(mockService *MockEmployeeService) {
                objectID := bson.NewObjectID()
                mockService.On("CreateEmployee", mock.Anything, mock.AnythingOfType("models.Employee")).
                    Return(objectID, nil)
            },
            expectedStatus: http.StatusCreated,
            expectedError:  false,
        },
		{
            name: "Duplicate email",
            payload: models.Employee{
                Name:       "Jane Doe",
                Department: "HR",
                Age:        28,
                Email:      "existing@example.com",
            },
            setupMock: func(mockService *MockEmployeeService) {
                mockService.On("CreateEmployee", mock.Anything, mock.AnythingOfType("models.Employee")).
                    Return(nil, errors.New("email address is already in use"))
            },
            expectedStatus: http.StatusConflict,
            expectedError:  true,
        },
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			mockService := new(MockEmployeeService)
			controller := NewEmployeeController(mockService)
			
			// Setup mock expectations
			tc.setupMock(mockService)

			// Create request
			payloadBytes, _ := json.Marshal(tc.payload)
			req, _ := http.NewRequest("POST", "/employees", bytes.NewBuffer(payloadBytes))
			rr := httptest.NewRecorder()
			
			// Call handler
            controller.CreateEmployee(rr, req)
            
            // Assert
            assert.Equal(t, tc.expectedStatus, rr.Code)
            
            // Parse response body
            var response Response
            json.Unmarshal(rr.Body.Bytes(), &response)
            
            if tc.expectedError {
                assert.NotEmpty(t, response.Error)
            } else {
                assert.NotEmpty(t, response.Message)
                assert.NotNil(t, response.Data)
            }
            
            // Verify mock
            mockService.AssertExpectations(t)
		})
	}
}


func TestGetEmployeeHandler(t *testing.T) {
    // Test cases
    testCases := []struct {
        name           string
        employeeID     string
        setupMock      func(mock *MockEmployeeService)
        expectedStatus int
    }{
        {
            name:       "Valid employee ID",
            employeeID: "507f1f77bcf86cd799439011",
            setupMock: func(mockService *MockEmployeeService) {
                employee := models.Employee{
                    ID: func() bson.ObjectID {
                        id, err := bson.ObjectIDFromHex("507f1f77bcf86cd799439011")
                        if err != nil {
                            t.Fatalf("failed to parse ObjectID: %v", err)
                        }
                        return id
                    }(),
                    Name:       "John Doe",
                    Department: "Engineering",
                    Age:        30,
                    Email:      "john.doe@example.com",
                }
                mockService.On("GetEmployeeByID", mock.Anything, "507f1f77bcf86cd799439011").
                    Return(employee, nil)
            },
            expectedStatus: http.StatusOK,
        },
        {
            name:       "Employee not found",
            employeeID: "507f1f77bcf86cd799439012",
            setupMock: func(mockService *MockEmployeeService) {
                mockService.On("GetEmployeeByID", mock.Anything, "507f1f77bcf86cd799439012").
                    Return(models.Employee{}, errors.New("employee not found"))
            },
            expectedStatus: http.StatusNotFound,
        },
	}
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Setup
            mockService := new(MockEmployeeService)
            controller := NewEmployeeController(mockService)
            
            // Setup mock expectations
            tc.setupMock(mockService)
            
            // Create request
            req, _ := http.NewRequest("GET", "/employees/"+tc.employeeID, nil)
            
            // Add URL parameters to request
            vars := map[string]string{
                "id": tc.employeeID,
            }
            req = mux.SetURLVars(req, vars)
            
            rr := httptest.NewRecorder()
            
            // Call handler
            controller.GetEmployee(rr, req)
            
            // Assert
            assert.Equal(t, tc.expectedStatus, rr.Code)
            
            // Verify mock
            mockService.AssertExpectations(t)
        })
    }
}


// TestGetAllEmployeesHandler tests the GetAllEmployees controller method
func TestGetAllEmployeesHandler(t *testing.T) {
    // Test cases
    testCases := []struct {
        name           string
        queryParams    map[string]string
        setupMock      func(mock *MockEmployeeService)
        expectedStatus int
        expectedItems  int
    }{
        {
            name: "Get first page with default limit",
            queryParams: map[string]string{
                "page": "1",
            },
            setupMock: func(mockService *MockEmployeeService) {
                employees := []models.Employee{
                    {
                        ID: func() bson.ObjectID {
                            id, err := bson.ObjectIDFromHex("507f1f77bcf86cd799439011")
                            if err != nil {
                                t.Fatalf("failed to parse ObjectID: %v", err)
                            }
                            return id
                        }(),
                        Name:       "John Doe",
                        Department: "Engineering",
                        Age:        30,
                        Email:      "john@example.com",
                    },
                    {
                        ID: func() bson.ObjectID {
                            id, err := bson.ObjectIDFromHex("507f1f77bcf86cd799439012")
                            if err != nil {
                                t.Fatalf("failed to parse ObjectID: %v", err)
                            }
                            return id
                        }(),
                        Name:       "Jane Smith",
                        Department: "HR",
                        Age:        28,
                        Email:      "jane@example.com",
                    },
                }
                mockService.On("GetAllEmployees", mock.Anything, 1, 10).Return(employees, int64(2), nil)
            },
            expectedStatus: http.StatusOK,
            expectedItems:  2,
        },
        {
            name: "Get page with custom limit",
            queryParams: map[string]string{
                "page":  "2",
                "limit": "5",
            },
            setupMock: func(mockService *MockEmployeeService) {
                employees := []models.Employee{
                    {
                        ID: func() bson.ObjectID {
                            id, err := bson.ObjectIDFromHex("507f1f77bcf86cd799439013")
                            if err != nil {
                                t.Fatalf("failed to parse ObjectID: %v", err)
                            }
                            return id
                        }(),
                        Name:       "Bob Johnson",
                        Department: "Finance",
                        Age:        35,
                        Email:      "bob@example.com",
                    },
                }
                mockService.On("GetAllEmployees", mock.Anything, 2, 5).Return(employees, int64(6), nil)
            },
            expectedStatus: http.StatusOK,
            expectedItems:  1,
        },
        {
            name: "Database error",
            queryParams: map[string]string{
                "page": "1",
            },
            setupMock: func(mockService *MockEmployeeService) {
                mockService.On("GetAllEmployees", mock.Anything, 1, 10).Return(
                    []models.Employee{}, int64(0), errors.New("database error"))
            },
            expectedStatus: http.StatusInternalServerError,
            expectedItems:  0,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Setup
            mockService := new(MockEmployeeService)
            controller := NewEmployeeController(mockService)

            // Setup mock expectations
            tc.setupMock(mockService)

            // Create request with query parameters
            req, _ := http.NewRequest("GET", "/employees", nil)
            q := req.URL.Query()
            for key, value := range tc.queryParams {
                q.Add(key, value)
            }
            req.URL.RawQuery = q.Encode()

            rr := httptest.NewRecorder()

            // Call handler
            controller.GetAllEmployees(rr, req)

            // Assert status code
            assert.Equal(t, tc.expectedStatus, rr.Code)

            // If success, check the body contains expected items
            if tc.expectedStatus == http.StatusOK {
                var response Response
                json.Unmarshal(rr.Body.Bytes(), &response)

                data, ok := response.Data.(map[string]interface{})
                assert.True(t, ok)

                employees, ok := data["employees"].([]interface{})
                assert.True(t, ok)
                assert.Len(t, employees, tc.expectedItems)

                pagination, ok := data["pagination"].(map[string]interface{})
                assert.True(t, ok)
                assert.NotNil(t, pagination["totalItems"])
            }

            // Verify mock
            mockService.AssertExpectations(t)
        })
    }
}

// TestUpdateEmployeeHandler tests the UpdateEmployee controller method
func TestUpdateEmployeeHandler(t *testing.T) {
    testCases := []struct {
        name           string
        employeeID     string
        payload        interface{}
        setupMock      func(mock *MockEmployeeService)
        expectedStatus int
    }{
        {
            name:       "Valid update",
            employeeID: "507f1f77bcf86cd799439011",
            payload: models.Employee{
                Name:       "Updated Name",
                Department: "Updated Department",
                Age:        32,
                Email:      "updated@example.com",
            },
            setupMock: func(mockService *MockEmployeeService) {
                mockService.On("UpdateEmployee", mock.Anything, "507f1f77bcf86cd799439011", mock.AnythingOfType("models.Employee")).
                    Return(int64(1), nil)
            },
            expectedStatus: http.StatusOK,
        },
        {
            name:       "Duplicate email",
            employeeID: "507f1f77bcf86cd799439011",
            payload: models.Employee{
                Email: "existing@example.com",
            },
            setupMock: func(mockService *MockEmployeeService) {
                mockService.On("UpdateEmployee", mock.Anything, "507f1f77bcf86cd799439011", mock.AnythingOfType("models.Employee")).
                    Return(int64(0), errors.New("email address is already in use by another employee"))
            },
            expectedStatus: http.StatusConflict,
        },
        {
            name:       "Employee not found",
            employeeID: "507f1f77bcf86cd799439099",
            payload: models.Employee{
                Name: "New Name",
            },
            setupMock: func(mockService *MockEmployeeService) {
                mockService.On("UpdateEmployee", mock.Anything, "507f1f77bcf86cd799439099", mock.AnythingOfType("models.Employee")).
                    Return(int64(0), nil)
            },
            expectedStatus: http.StatusNotFound,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Setup
            mockService := new(MockEmployeeService)
            controller := NewEmployeeController(mockService)

            // Setup mock expectations
            tc.setupMock(mockService)

            // Create request
            payloadBytes, _ := json.Marshal(tc.payload)
            req, _ := http.NewRequest("PUT", "/employees/"+tc.employeeID, bytes.NewBuffer(payloadBytes))

            // Add URL parameters
            vars := map[string]string{
                "id": tc.employeeID,
            }
            req = mux.SetURLVars(req, vars)

            rr := httptest.NewRecorder()

            // Call handler
            controller.UpdateEmployee(rr, req)

            // Assert status code
            assert.Equal(t, tc.expectedStatus, rr.Code)

            // Verify mock
            mockService.AssertExpectations(t)
        })
    }
}

// TestDeleteEmployeeHandler tests the DeleteEmployee controller method
func TestDeleteEmployeeHandler(t *testing.T) {
    testCases := []struct {
        name           string
        employeeID     string
        setupMock      func(mock *MockEmployeeService)
        expectedStatus int
    }{
        {
            name:       "Successful deletion",
            employeeID: "507f1f77bcf86cd799439011",
            setupMock: func(mockService *MockEmployeeService) {
                mockService.On("DeleteEmployee", mock.Anything, "507f1f77bcf86cd799439011").
                    Return(int64(1), nil)
            },
            expectedStatus: http.StatusOK,
        },
        {
            name:       "Employee not found",
            employeeID: "507f1f77bcf86cd799439099",
            setupMock: func(mockService *MockEmployeeService) {
                mockService.On("DeleteEmployee", mock.Anything, "507f1f77bcf86cd799439099").
                    Return(int64(0), nil)
            },
            expectedStatus: http.StatusNotFound,
        },
        {
            name:       "Database error",
            employeeID: "507f1f77bcf86cd799439011",
            setupMock: func(mockService *MockEmployeeService) {
                mockService.On("DeleteEmployee", mock.Anything, "507f1f77bcf86cd799439011").
                    Return(int64(0), errors.New("database error"))
            },
            expectedStatus: http.StatusInternalServerError,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Setup
            mockService := new(MockEmployeeService)
            controller := NewEmployeeController(mockService)

            // Setup mock expectations
            tc.setupMock(mockService)

            // Create request
            req, _ := http.NewRequest("DELETE", "/employees/"+tc.employeeID, nil)

            // Add URL parameters
            vars := map[string]string{
                "id": tc.employeeID,
            }
            req = mux.SetURLVars(req, vars)

            rr := httptest.NewRecorder()

            // Call handler
            controller.DeleteEmployee(rr, req)

            // Assert status code
            assert.Equal(t, tc.expectedStatus, rr.Code)

            // Verify mock
            mockService.AssertExpectations(t)
        })
    }
}

// TestDeleteAllEmployeesHandler tests the DeleteAllEmployees controller method
func TestDeleteAllEmployeesHandler(t *testing.T) {
    testCases := []struct {
        name           string
        setupMock      func(mock *MockEmployeeService)
        expectedStatus int
        deletedCount   int64
    }{
        {
            name: "Successful deletion of all employees",
            setupMock: func(mockService *MockEmployeeService) {
                mockService.On("DeleteAllEmployees", mock.Anything).Return(int64(5), nil)
            },
            expectedStatus: http.StatusOK,
            deletedCount:   5,
        },
        {
            name: "No employees to delete",
            setupMock: func(mockService *MockEmployeeService) {
                mockService.On("DeleteAllEmployees", mock.Anything).Return(int64(0), nil)
            },
            expectedStatus: http.StatusOK,
            deletedCount:   0,
        },
        {
            name: "Database error",
            setupMock: func(mockService *MockEmployeeService) {
                mockService.On("DeleteAllEmployees", mock.Anything).Return(int64(0), errors.New("database error"))
            },
            expectedStatus: http.StatusInternalServerError,
            deletedCount:   0,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Setup
            mockService := new(MockEmployeeService)
            controller := NewEmployeeController(mockService)

            // Setup mock expectations
            tc.setupMock(mockService)

            // Create request
            req, _ := http.NewRequest("DELETE", "/employees", nil)
            rr := httptest.NewRecorder()

            // Call handler
            controller.DeleteAllEmployees(rr, req)

            // Assert status code
            assert.Equal(t, tc.expectedStatus, rr.Code)

            // For successful responses, verify the count
            if tc.expectedStatus == http.StatusOK {
                var response Response
                json.Unmarshal(rr.Body.Bytes(), &response)
                
                data, ok := response.Data.(map[string]interface{})
                assert.True(t, ok)
                
                count, ok := data["deletedCount"].(float64)
                assert.True(t, ok)
                assert.Equal(t, float64(tc.deletedCount), count)
            }

            // Verify mock
            mockService.AssertExpectations(t)
        })
    }
}

// TestHealthCheckHandler tests the HealthCheck controller method
func TestHealthCheckHandler(t *testing.T) {
    testCases := []struct {
        name           string
        setupMock      func(mock *MockEmployeeService)
        expectedStatus int
        isHealthy      bool
    }{
        {
            name: "System is healthy",
            setupMock: func(mockService *MockEmployeeService) {
                mockService.On("HealthCheck", mock.Anything).Return(nil)
            },
            expectedStatus: http.StatusOK,
            isHealthy:      true,
        },
        {
            name: "Database unavailable",
            setupMock: func(mockService *MockEmployeeService) {
                mockService.On("HealthCheck", mock.Anything).Return(errors.New("database connection failed"))
            },
            expectedStatus: http.StatusServiceUnavailable,
            isHealthy:      false,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Setup
            mockService := new(MockEmployeeService)
            controller := NewEmployeeController(mockService)

            // Setup mock expectations
            tc.setupMock(mockService)

            // Create request
            req, _ := http.NewRequest("GET", "/health", nil)
            rr := httptest.NewRecorder()

            // Call handler
            controller.HealthCheck(rr, req)

            // Assert status code
            assert.Equal(t, tc.expectedStatus, rr.Code)

            // Parse and verify response
            var response map[string]string
            json.Unmarshal(rr.Body.Bytes(), &response)

            if tc.isHealthy {
                assert.Equal(t, "healthy", response["status"])
            } else {
                assert.Equal(t, "database unavailable", response["status"])
                assert.NotEmpty(t, response["error"])
            }

            // Verify mock
            mockService.AssertExpectations(t)
        })
    }
}