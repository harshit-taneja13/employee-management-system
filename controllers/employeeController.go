package controllers

import (
	"context"
	"employee-management/models"
	"employee-management/services"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// Constants for the application
const (
	DefaultTimeout  = 5 * time.Second
	DefaultPageSize = 10
	MaxPageSize     = 100
)

// Response is a standardized API response structure
type Response struct {
	Status  int         `json:"-"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type EmployeeController struct {
	service *services.EmployeeService
}

func NewEmployeeController(service *services.EmployeeService) *EmployeeController {
	return &EmployeeController{service: service}
}

// Helper functions for consistent responses
func respondJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if payload != nil {
		json.NewEncoder(w).Encode(payload)
	}
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondJSON(w, code, Response{
		Status: code,
		Error:  message,
	})
	log.Printf("Error response: %d - %s", code, message)
}

// CreateEmployee handles POST /employees to add a new employee
func (c *EmployeeController) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received create employee request from %s", r.RemoteAddr)
	var employee models.Employee

	if err := json.NewDecoder(r.Body).Decode(&employee); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), DefaultTimeout)
	defer cancel()

	result, err := c.service.CreateEmployee(ctx, employee)
	if err != nil {
		if err.Error() == "email address is already in use" {
			respondWithError(w, http.StatusConflict, err.Error())
		} else {
			respondWithError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	respondJSON(w, http.StatusCreated, Response{
		Status:  http.StatusCreated,
		Message: "Employee created successfully",
		Data:    result,
	})
}

// GetAllEmployees handles GET /employees to retrieve all employees
func (c *EmployeeController) GetAllEmployees(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received get all employees request from %s", r.RemoteAddr)

	// Parse pagination parameters
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page <= 0 {
		page = 1
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit <= 0 || limit > MaxPageSize {
		limit = DefaultPageSize
	}

	ctx, cancel := context.WithTimeout(r.Context(), DefaultTimeout)
	defer cancel()

	employees, totalCount, err := c.service.GetAllEmployees(ctx, page, limit)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error fetching employees")
		return
	}

	totalPages := (totalCount + int64(limit) - 1) / int64(limit)
	respondJSON(w, http.StatusOK, Response{
		Status:  http.StatusOK,
		Message: fmt.Sprintf("Retrieved %d employees", len(employees)),
		Data: map[string]interface{}{
			"employees": employees,
			"pagination": map[string]interface{}{
				"currentPage": page,
				"pageSize":    limit,
				"totalItems":  totalCount,
				"totalPages":  totalPages,
			},
		},
	})
}

// GetEmployee handles GET /employees/{id} to retrieve a specific employee
func (c *EmployeeController) GetEmployee(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	// Use the 'id' parameter directly.
	id := params["id"]

	log.Printf("Received get employee request for ID: %s", id)
	var employee models.Employee

	ctx, cancel := context.WithTimeout(r.Context(), DefaultTimeout)
	defer cancel()
	employee, err := c.service.GetEmployeeByID(ctx, id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Employee not found")
		return
	}

	respondJSON(w, http.StatusOK, Response{
		Status: http.StatusOK,
		Data:   employee,
	})
}

// UpdateEmployee handles PUT /employees/{id} to update an employee.
func (c *EmployeeController) UpdateEmployee(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	log.Printf("Received update employee request for ID: %s", id)
	var employee models.Employee

	if err := json.NewDecoder(r.Body).Decode(&employee); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), DefaultTimeout)
	defer cancel()

	modifiedCount, err := c.service.UpdateEmployee(ctx, id, employee)
	if err != nil {
		respondWithError(w, http.StatusConflict, err.Error())
		return
	}
	if modifiedCount == 0 {
		respondWithError(w, http.StatusNotFound, "Employee not Modified")
		return
	}

	respondJSON(w, http.StatusOK, Response{
		Status:  http.StatusOK,
		Message: "Employee updated successfully",
		Data:    map[string]interface{}{"modifiedCount": modifiedCount},
	})

}

// DeleteEmployee handles DELETE /employees/{id} to remove an employee.
func (c *EmployeeController) DeleteEmployee(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	log.Printf("Received delete employee request for ID: %s", id)
	ctx, cancel := context.WithTimeout(r.Context(), DefaultTimeout)
	defer cancel()

	deletedCount, err := c.service.DeleteEmployee(ctx, id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error deleting employee")
		return
	}

	if deletedCount == 0 {
		respondWithError(w, http.StatusNotFound, "Employee not found")
		return
	}
	respondJSON(w, http.StatusOK, Response{
		Status:  http.StatusOK,
		Message: "Employee deleted successfully",
		Data:    map[string]interface{}{"deletedCount": deletedCount},
	})
}

// DeleteAllEmployee handles DELETE /employees to remove all employees
func (c *EmployeeController) DeleteAllEmployees(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received delete all employees request from %s", r.RemoteAddr)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	deletedCount, err := c.service.DeleteAllEmployees(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error deleting employees")
		return
	}

	respondJSON(w, http.StatusOK, Response{
		Status:  http.StatusOK,
		Message: "All employees deleted successfully",
		Data:    map[string]interface{}{"deletedCount": deletedCount},
	})
}

// HealthCheck handles GET /health to check system status
func (c *EmployeeController) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := c.service.HealthCheck(ctx)
	if err != nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "database unavailable",
			"error":  err.Error(),
		})
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}
