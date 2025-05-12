package services

import (
	"context"
	"employee-management/models"
	"employee-management/repository"
	"errors"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type EmployeeService struct {
	repo repository.EmployeeRepository
}

type EmployeeServiceInterface interface {
    CreateEmployee(ctx context.Context, emp models.Employee) (interface{}, error)
    GetAllEmployees(ctx context.Context, page, limit int) ([]models.Employee, int64, error)
    GetEmployeeByID(ctx context.Context, id string) (models.Employee, error)
    UpdateEmployee(ctx context.Context, id string, emp models.Employee) (int64, error)
    DeleteEmployee(ctx context.Context, id string) (int64, error)
    DeleteAllEmployees(ctx context.Context) (int64, error)
    HealthCheck(ctx context.Context) error
}

// Ensure EmployeeService implements EmployeeServiceInterface
var _ EmployeeServiceInterface = (*EmployeeService)(nil)

// NewEmployeeService creates a new service instance.
func NewEmployeeService(repo repository.EmployeeRepository) *EmployeeService {
	return &EmployeeService{repo: repo}
}

// ValidateEmployee validates the employee data
func (s *EmployeeService) ValidateEmployee(emp models.Employee) error {
	if strings.TrimSpace(emp.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(emp.Department) == "" {
		return errors.New("department is required")
	}
	if emp.Age < 0 {
		return errors.New("age must be positive")
	}
	if strings.TrimSpace(emp.Email) == "" {
		return errors.New("email is required")
	}
	if !strings.Contains(emp.Email, "@") || !strings.Contains(emp.Email, ".") {
		return errors.New("invalid email format")
	}
	return nil
}

func (s *EmployeeService) CreateEmployee(ctx context.Context, emp models.Employee) (interface{}, error) {
	err := s.ValidateEmployee(emp)
	if err != nil {
		return nil, err
	}
	exists, err := s.repo.CheckEmailExists(ctx, emp.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("email address is already in use")
	}
	emp.ID = bson.NewObjectID()
	return s.repo.Create(ctx, emp)
}

func (s *EmployeeService) GetAllEmployees(ctx context.Context, page, limit int) ([]models.Employee, int64, error) {
	return s.repo.FindAll(ctx, page, limit)
}

func (s *EmployeeService) GetEmployeeByID(ctx context.Context, id string) (models.Employee, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *EmployeeService) UpdateEmployee(ctx context.Context, id string, emp models.Employee) (int64, error) {

	exists, err := s.repo.CheckEmailExists(ctx, emp.Email, id)
	if err != nil {
		return 0, err
	}
	if exists {
		return 0, errors.New("email address is already in use by another employee")
	}

	return s.repo.Update(ctx, id, emp)
}

func (s *EmployeeService) DeleteEmployee(ctx context.Context, id string) (int64, error) {
	return s.repo.Delete(ctx, id)
}

func (s *EmployeeService) DeleteAllEmployees(ctx context.Context) (int64, error) {
	return s.repo.DeleteAll(ctx)
}

func (s *EmployeeService) HealthCheck(ctx context.Context) error {
	return s.repo.Ping(ctx)
}
