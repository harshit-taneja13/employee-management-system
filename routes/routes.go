package routes

import (
	"employee-management/controllers"

	"github.com/gorilla/mux"
)

// SetupRoutes registers all API endpoints.
func SetupRoutes(employeeController *controllers.EmployeeController) *mux.Router {
	router := mux.NewRouter()

	// Create a new employee.
	router.HandleFunc("/employees", employeeController.CreateEmployee).Methods("POST")
	// Retrieve all employees.
	router.HandleFunc("/employees", employeeController.GetAllEmployees).Methods("GET")
	// Retrieve a specific employee by ID.
	router.HandleFunc("/employees/{id}", employeeController.GetEmployee ).Methods("GET")
	// Update a specific employee by ID.
	router.HandleFunc("/employees/{id}", employeeController.UpdateEmployee).Methods("PUT")
	// Delete a specific employee by ID.
	router.HandleFunc("/employees/{id}", employeeController.DeleteEmployee).Methods("DELETE")
	// Delete all employees.
	router.HandleFunc("/employees", employeeController.DeleteAllEmployees ).Methods("DELETE")
	// To check system status
	router.HandleFunc("/health" , employeeController.HealthCheck ).Methods("GET")

	return router
}

