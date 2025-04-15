package routes

import (
	"employee-management/controllers"

	"github.com/gorilla/mux"
)

// SetupRoutes registers all API endpoints.
func SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	// Create a new employee.
	router.HandleFunc("/employees", controllers.CreateEmployee).Methods("POST")
	// Retrieve all employees.
	router.HandleFunc("/employees", controllers.GetAllEmployees).Methods("GET")
	// Retrieve a specific employee by ID.
	router.HandleFunc("/employees/{id}", controllers.GetEmployee).Methods("GET")
	// Update a specific employee by ID.
	router.HandleFunc("/employees/{id}", controllers.UpdateEmployee).Methods("PUT")
	// Delete a specific employee by ID.
	router.HandleFunc("/employees/{id}", controllers.DeleteEmployee).Methods("DELETE")
	// Delete all employees.
	router.HandleFunc("/employees", controllers.DeleteAllEmployees).Methods("DELETE")
	// To check system status
	router.HandleFunc("/health" , controllers.HealthCheck ).Methods("GET")

	return router
}

