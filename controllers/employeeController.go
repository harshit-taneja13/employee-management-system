package controllers

import (
	"context"
	"employee-management/database"
	"employee-management/models"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Constants for the application
const (
    DatabaseName   = "employeedb"
    CollectionName = "employees"
    DefaultTimeout = 5 * time.Second
	DefaultPageSize  = 10
    MaxPageSize      = 100
)

// Response is a standardized API response structure
type Response struct {
    Status  int         `json:"-"`
    Message string      `json:"message,omitempty"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

// getEmployeeCollection returns a handle to the "employees" collection
func getEmployeeCollection() *mongo.Collection {
	return database.Client.Database(DatabaseName).Collection(CollectionName)
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


// validateEmployee validates the employee data
func validateEmployee(employee models.Employee) error {
    if employee.Name == "" {
        return errors.New("name is required")
    }
    if employee.Department == "" {
        return errors.New("department is required")
    }
    if employee.Age <= 0 {
        return errors.New("age must be positive")
    }
    if employee.Email == "" {
        return errors.New("email is required")
    }
    return nil
}

// checkEmailExists verifies if an email is already in use
// If excludeID is provided, it will exclude that employee from the check (for updates)
func checkEmailExists(ctx context.Context, email string, excludeID ...bson.ObjectID) (bool, error) {
    collection := getEmployeeCollection()
    
    filter := bson.M{"email": email}
    
    // If we're updating an employee, exclude the current employee from the check
    if len(excludeID) > 0 && !excludeID[0].IsZero() {
        filter = bson.M{
            "email": email,
            "_id": bson.M{"$ne": excludeID[0]},
        }
    }
    
    // Count documents with this email
    count, err := collection.CountDocuments(ctx, filter)
    if err != nil {
        return false, fmt.Errorf("error checking email existence: %w", err)
    }
    
    return count > 0, nil
}
	
// CreateEmployee handles POST /employees to add a new employee
func CreateEmployee(w http.ResponseWriter, r *http.Request){
	log.Printf("Received create employee request from %s", r.RemoteAddr)
    var employee models.Employee

    if err := json.NewDecoder(r.Body).Decode(&employee); err != nil {
        respondWithError(w, http.StatusBadRequest, "Invalid request payload")
        return
    }

    if err := validateEmployee(employee); err != nil {
        respondWithError(w, http.StatusBadRequest, err.Error())
        return
    }

    ctx, cancel := context.WithTimeout(r.Context(), DefaultTimeout)
    defer cancel()

	// check if email already exists 
	exists, err := checkEmailExists(ctx, employee.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error checking email uniqueness")
        log.Printf("Database error: %v", err)
        return
	}
	if exists {
		respondWithError(w, http.StatusConflict, "Email address is already in use")
		return
	}

    employee.ID = bson.NewObjectID()
    collection := getEmployeeCollection()

    result, err := collection.InsertOne(ctx, employee)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Error while inserting employee")
        log.Printf("Database error: %v", err)
        return
    }

    log.Printf("Employee created with ID: %s", employee.ID.Hex())
    respondJSON(w, http.StatusCreated, Response{
        Status:  http.StatusCreated,
        Message: "Employee created successfully",
        Data:    result,
    })
}

// GetAllEmployees handles GET /employees to retrieve all employees
func GetAllEmployees(w http.ResponseWriter, r *http.Request) {
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
	
	skip := (page - 1) * limit
    
    collection := getEmployeeCollection()
    ctx, cancel := context.WithTimeout(r.Context(), DefaultTimeout)
    defer cancel()

	// Define options with pagination and sorting
    findOptions := options.Find().
        SetLimit(int64(limit)).
        SetSkip(int64(skip)).
        SetSort(bson.D{{"name", 1}})

    cursor, err := collection.Find(ctx, bson.M{}, findOptions)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Error while fetching employees")
        log.Printf("Database error: %v", err)
        return
    }
    defer cursor.Close(ctx)

    var employees []models.Employee
    if err := cursor.All(ctx, &employees); err != nil {
        respondWithError(w, http.StatusInternalServerError, "Error decoding employees")
        log.Printf("Cursor decode error: %v", err)
        return
    }

	// Get total count for pagination info
    count, err := collection.CountDocuments(ctx, bson.M{})
    if err != nil {
        log.Printf("Count error: %v", err)
        // Continue anyway, not critical
    }
	
	respondJSON(w, http.StatusOK, Response{
        Status:  http.StatusOK,
        Message: fmt.Sprintf("Retrieved %d employees", len(employees)),
        Data: map[string]interface{}{
            "employees": employees,
            "pagination": map[string]interface{}{
                "currentPage": page,
                "pageSize":    limit,
                "totalItems":  count,
                "totalPages":  (count + int64(limit) - 1) / int64(limit),
            },
        },
    })
}

// GetEmployee handles GET /employees/{id} to retrieve a specific employee
func GetEmployee(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
    empID, err := bson.ObjectIDFromHex(params["id"])
    if err != nil {
        respondWithError(w, http.StatusBadRequest, "Invalid employee ID")
        return
    }

    log.Printf("Received get employee request for ID: %s", empID.Hex())
    collection := getEmployeeCollection()
    var employee models.Employee

    ctx, cancel := context.WithTimeout(r.Context(), DefaultTimeout)
    defer cancel()
    
    err = collection.FindOne(ctx, bson.M{"_id": empID}).Decode(&employee)
    if err != nil {
        if errors.Is(err, mongo.ErrNoDocuments) {
            respondWithError(w, http.StatusNotFound, "Employee not found")
        } else {
            respondWithError(w, http.StatusInternalServerError, "Database error")
            log.Printf("Database error: %v", err)
        }
        return
    }

    respondJSON(w, http.StatusOK, Response{
        Status: http.StatusOK,
        Data:   employee,
    })
}


// UpdateEmployee handles PUT /employees/{id} to update an employee.
func UpdateEmployee(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
    empID, err := bson.ObjectIDFromHex(params["id"])
    if err != nil {
        respondWithError(w, http.StatusBadRequest, "Invalid employee ID")
        return
    }

    log.Printf("Received update employee request for ID: %s", empID.Hex())
    var employee models.Employee
    
    if err := json.NewDecoder(r.Body).Decode(&employee); err != nil {
        respondWithError(w, http.StatusBadRequest, "Invalid request payload")
        return
    }

    if err := validateEmployee(employee); err != nil {
        respondWithError(w, http.StatusBadRequest, err.Error())
        return
    }
	ctx, cancel := context.WithTimeout(r.Context(), DefaultTimeout)
	defer cancel()

	// checking if updated email already exists (excluding this employee)
	exists, err := checkEmailExists(ctx , employee.Email, empID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError , "Error checking email uniqueness")
		log.Printf("Database error: %v", err)
        return
	}

	if exists {
        respondWithError(w, http.StatusConflict, "Email address is already in use by another employee")
        return
    }

    collection := getEmployeeCollection()

    update := bson.M{
        "$set": bson.M{
            "name":       employee.Name,
            "department": employee.Department,
            "age":        employee.Age,
            "email":      employee.Email,
        },
    }

	result, err := collection.UpdateOne(ctx, bson.M{"_id": empID}, update)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Error updating employee")
        log.Printf("Database error: %v", err)
        return
    }

    if result.MatchedCount == 0 {
        respondWithError(w, http.StatusNotFound, "Employee not found")
        return
    }

    respondJSON(w, http.StatusOK, Response{
        Status:  http.StatusOK,
        Message: "Employee updated successfully",
        Data:    map[string]interface{}{"modifiedCount": result.ModifiedCount},
    })

}

// DeleteEmployee handles DELETE /employees/{id} to remove an employee.
func DeleteEmployee(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
    empID, err := bson.ObjectIDFromHex(params["id"])
    if err != nil {
        respondWithError(w, http.StatusBadRequest, "Invalid employee ID")
        return
    }

    log.Printf("Received delete employee request for ID: %s", empID.Hex())
    collection := getEmployeeCollection()
    ctx, cancel := context.WithTimeout(r.Context(), DefaultTimeout)
    defer cancel()

    result, err := collection.DeleteOne(ctx, bson.M{"_id": empID})
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Error deleting employee")
        log.Printf("Database error: %v", err)
        return
    }

    if result.DeletedCount == 0 {
        respondWithError(w, http.StatusNotFound, "Employee not found")
        return
    }

    respondJSON(w, http.StatusOK, Response{
        Status:  http.StatusOK,
        Message: "Employee deleted successfully",
        Data:    map[string]interface{}{"deletedCount": result.DeletedCount},
    })
}

// DeleteAllEmployee handles DELETE /employees to remove all employees
func DeleteAllEmployees(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received delete all employees request from %s", r.RemoteAddr)
    collection := getEmployeeCollection()
    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()

    result, err := collection.DeleteMany(ctx, bson.M{})
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Error deleting employees")
        log.Printf("Database error: %v", err)
        return
    }

    respondJSON(w, http.StatusOK, Response{
        Status:  http.StatusOK,
        Message: "All employees deleted successfully",
        Data:    map[string]interface{}{"deletedCount": result.DeletedCount},
    })
}

// HealthCheck handles GET /health to check system status
func HealthCheck(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    
    err := database.Client.Ping(ctx, nil)
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