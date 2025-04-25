package repository

import (
	"context"
	"employee-management/models"
	"log"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// TestSetup encapsulates test DB resources
type TestSetup struct{
	Client	*mongo.Client
	Repository EmployeeRepository
	DBName string
	Collection string
}

func SetupIntegrationTest(t *testing.T) *TestSetup {
	// Skip integration tests when running in short mode
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Load test configuration
    if err := godotenv.Load("../.env.test"); err != nil {
        // Use default test settings if no .env.test file
        log.Println("Warning: No .env.test file found")
    }
    
    // Use test database URI or default to localhost
    mongoURI := os.Getenv("TEST_MONGO_URI")
    if mongoURI == "" {
        mongoURI = "mongodb://localhost:27017"
    }
    
    // Connect to MongoDB
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    client, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
    if err != nil {
        t.Fatalf("Failed to connect to MongoDB: %v", err)
    }
    
    // Ping the database to verify connection
    err = client.Ping(ctx, readpref.Primary())
    if err != nil {
        t.Fatalf("Failed to ping MongoDB: %v", err)
    }
    
    // Use test database and collection names
    dbName := "test_employee_db"
    collName := "test_employees"
    
    // Clear existing data
    err = client.Database(dbName).Collection(collName).Drop(ctx)
    if err != nil {
        log.Printf("Warning when dropping collection: %v", err)
    }

	// Create repository with test database
    repo := NewMongoEmployeeRepository(client, dbName, collName)
    
    return &TestSetup{
        Client:     client,
        Repository: repo,
        DBName:     dbName,
        Collection: collName,
    }
}

// TeardownIntegrationTest cleans up test resources
func (ts *TestSetup) TeardownIntegrationTest(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // Drop test collection
    err := ts.Client.Database(ts.DBName).Collection(ts.Collection).Drop(ctx)
    if err != nil {
        log.Printf("Warning when dropping collection during teardown: %v", err)
    }
    
    // Disconnect from MongoDB
    if err := ts.Client.Disconnect(ctx); err != nil {
        t.Fatalf("Failed to disconnect from MongoDB: %v", err)
    }
}

// Test Create method
func TestIntegrationCreate(t *testing.T) {
    // Setup test environment
    ts := SetupIntegrationTest(t)
    defer ts.TeardownIntegrationTest(t)
    
    ctx := context.Background()
    
    // Test successful employee creation
    t.Run("Create new employee", func(t *testing.T) {
        // Prepare test employee
        newEmployee := models.Employee{
            Name:       "New Employee",
            Department: "IT",
            Age:        28,
            Email:      "new@example.com",
        }
        
        // Create employee
        result, err := ts.Repository.Create(ctx, newEmployee)
        
        // Assertions
        assert.NoError(t, err)
        assert.NotNil(t, result)
        
        // Verify employee was actually saved by retrieving it
        // First convert the result to ObjectID
        insertedID, ok := result.(bson.ObjectID)
        assert.True(t, ok, "Result should be a bson.ObjectID")
        
        // Now retrieve the employee and verify
        savedEmployee, err := ts.Repository.FindByID(ctx, insertedID.Hex())
        assert.NoError(t, err)
        assert.Equal(t, newEmployee.Name, savedEmployee.Name)
        assert.Equal(t, newEmployee.Email, savedEmployee.Email)
        assert.Equal(t, newEmployee.Department, savedEmployee.Department)
        assert.Equal(t, newEmployee.Age, savedEmployee.Age)
    })
}

// Test FindAll integration
func TestIntegrationFindAll(t *testing.T) {
    // Setup test environment
    ts := SetupIntegrationTest(t)
    defer ts.TeardownIntegrationTest(t)
    
    ctx := context.Background()
    
    // Insert test data
    testEmployees := []models.Employee{
        {
            ID:         bson.NewObjectID(),
            Name:       "Alice Johnson",
            Department: "Engineering",
            Age:        30,
            Email:      "alice@example.com",
        },
        {
            ID:         bson.NewObjectID(),
            Name:       "Bob Smith",
            Department: "Marketing",
            Age:        35,
            Email:      "bob@example.com",
        },
        {
            ID:         bson.NewObjectID(),
            Name:       "Charlie Brown",
            Department: "Finance",
            Age:        40,
            Email:      "charlie@example.com",
        },
    }

    // Insert each test employee
    for _, emp := range testEmployees {
        _, err := ts.Client.Database(ts.DBName).Collection(ts.Collection).InsertOne(ctx, emp)
        assert.NoError(t, err)
    }
    
    // Test FindAll with default pagination
    t.Run("Default pagination", func(t *testing.T) {
        employees, count, err := ts.Repository.FindAll(ctx, 1, 10)
        
        assert.NoError(t, err)
        assert.Equal(t, int64(3), count)
        assert.Len(t, employees, 3)
    })
    
    // Test pagination - page 1, limit 2
    t.Run("Pagination page 1 limit 2", func(t *testing.T) {
        employees, count, err := ts.Repository.FindAll(ctx, 1, 2)
        
        assert.NoError(t, err)
        assert.Equal(t, int64(3), count)
        assert.Len(t, employees, 2)
    })
    
    // Test pagination - page 2, limit 2
    t.Run("Pagination page 2 limit 2", func(t *testing.T) {
        employees, count, err := ts.Repository.FindAll(ctx, 2, 2)
        
        assert.NoError(t, err)
        assert.Equal(t, int64(3), count)
        assert.Len(t, employees, 1)
    })
}

// Test FindByID integration
func TestIntegrationFindByID(t *testing.T) {
    // Setup test environment
    ts := SetupIntegrationTest(t)
    defer ts.TeardownIntegrationTest(t)
    
    ctx := context.Background()
    
    // Create test employee
    testID := bson.NewObjectID()
    testEmployee := models.Employee{
        ID:         testID,
        Name:       "Test User",
        Department: "Testing",
        Age:        25,
        Email:      "test@example.com",
    }
    
    // Insert into database
    _, err := ts.Client.Database(ts.DBName).Collection(ts.Collection).InsertOne(ctx, testEmployee)
    assert.NoError(t, err)
    
    // Test finding existing employee
    t.Run("Find existing employee", func(t *testing.T) {
        employee, err := ts.Repository.FindByID(ctx, testID.Hex())
        
        assert.NoError(t, err)
        assert.Equal(t, testEmployee.ID, employee.ID)
        assert.Equal(t, testEmployee.Name, employee.Name)
        assert.Equal(t, testEmployee.Email, employee.Email)
    })
    
    // Test with non-existent ID
    t.Run("Find non-existent employee", func(t *testing.T) {
        nonExistentID := bson.NewObjectID().Hex()
        _, err := ts.Repository.FindByID(ctx, nonExistentID)
        
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "employee not found")
    })


    // Test with invalid ID
    t.Run("Invalid ID format", func(t *testing.T) {
        _, err := ts.Repository.FindByID(ctx, "invalid-id")
        
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "invalid ID format")
    })
}

// Test Update method
func TestIntegrationUpdate(t *testing.T) {
    // Setup test environment
    ts := SetupIntegrationTest(t)
    defer ts.TeardownIntegrationTest(t)
    
    ctx := context.Background()
    
    // Create test employee
    originalEmployee := models.Employee{
        ID:         bson.NewObjectID(),
        Name:       "Original Name",
        Department: "Original Dept",
        Age:        30,
        Email:      "original@example.com",
    }
    
    // Insert into database
    _, err := ts.Client.Database(ts.DBName).Collection(ts.Collection).InsertOne(ctx, originalEmployee)
    assert.NoError(t, err)
    
    // Test full update
    t.Run("Full update", func(t *testing.T) {
        // Prepare update data
        updatedEmployee := models.Employee{
            Name:       "Updated Name",
            Department: "Updated Dept",
            Age:        35,
            Email:      "updated@example.com",
        }
        
        // Update employee
        count, err := ts.Repository.Update(ctx, originalEmployee.ID.Hex(), updatedEmployee)

        assert.NoError(t, err)
        assert.Equal(t, int64(1), count)
        
        // Verify changes by retrieving updated employee
        retrievedEmployee, err := ts.Repository.FindByID(ctx, originalEmployee.ID.Hex())
        assert.NoError(t, err)
        assert.Equal(t, updatedEmployee.Name, retrievedEmployee.Name)
        assert.Equal(t, updatedEmployee.Department, retrievedEmployee.Department)
        assert.Equal(t, updatedEmployee.Age, retrievedEmployee.Age)
        assert.Equal(t, updatedEmployee.Email, retrievedEmployee.Email)
    })
	 
    // Test partial update (only name)
    t.Run("Partial update", func(t *testing.T) {
        // Create another test employee
        partialEmployee := models.Employee{
            ID:         bson.NewObjectID(),
            Name:       "Partial Name",
            Department: "Partial Dept",
            Age:        40,
            Email:      "partial@example.com",
        }
        
        // Insert into database
        _, err := ts.Client.Database(ts.DBName).Collection(ts.Collection).InsertOne(ctx, partialEmployee)
        assert.NoError(t, err)
        
        // Update only the name
        nameUpdate := models.Employee{
            Name: "Updated Partial Name",
            // Other fields left empty
        }
		  
        // Update employee
        count, err := ts.Repository.Update(ctx, partialEmployee.ID.Hex(), nameUpdate)
        
        // Assertions
        assert.NoError(t, err)
        assert.Equal(t, int64(1), count)
        
        // Verify only name was updated
        retrievedEmployee, err := ts.Repository.FindByID(ctx, partialEmployee.ID.Hex())
        assert.NoError(t, err)
        assert.Equal(t, nameUpdate.Name, retrievedEmployee.Name)
        assert.Equal(t, partialEmployee.Department, retrievedEmployee.Department)
        assert.Equal(t, partialEmployee.Age, retrievedEmployee.Age)
        assert.Equal(t, partialEmployee.Email, retrievedEmployee.Email)
    })
    
    // Test update with invalid ID
    t.Run("Invalid ID", func(t *testing.T) {
        updatedEmployee := models.Employee{
            Name: "Won't Be Updated",
        }
        
        count, err := ts.Repository.Update(ctx, "invalid-id", updatedEmployee)
        
        assert.Error(t, err)
        assert.Equal(t, int64(0), count)
        assert.Contains(t, err.Error(), "invalid ID format")
    })
    
    // Test update for non-existent employee
    t.Run("Non-existent employee", func(t *testing.T) {
        nonExistentID := bson.NewObjectID().Hex()
        updatedEmployee := models.Employee{
            Name: "Won't Be Updated",
        }
        
        count, err := ts.Repository.Update(ctx, nonExistentID, updatedEmployee)
        
        assert.NoError(t, err) // No error should be returned if ID is valid but not found
        assert.Equal(t, int64(0), count) // But modified count should be 0
    })
}

// Test Delete method
func TestIntegrationDelete(t *testing.T) {
    // Setup test environment
    ts := SetupIntegrationTest(t)
    defer ts.TeardownIntegrationTest(t)
    
    ctx := context.Background()
    
    // Create test employee
    employeeToDelete := models.Employee{
        ID:         bson.NewObjectID(),
        Name:       "To Delete",
        Department: "Temporary",
        Age:        25,
        Email:      "delete@example.com",
    }
    
    // Insert into database
    _, err := ts.Client.Database(ts.DBName).Collection(ts.Collection).InsertOne(ctx, employeeToDelete)
    assert.NoError(t, err)
    
    // Test successful deletion
    t.Run("Delete existing employee", func(t *testing.T) {
        count, err := ts.Repository.Delete(ctx, employeeToDelete.ID.Hex())
        
        assert.NoError(t, err)
        assert.Equal(t, int64(1), count)
        
        // Verify employee is actually deleted
        _, err = ts.Repository.FindByID(ctx, employeeToDelete.ID.Hex())
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "employee not found")
    })

	// Test deletion with invalid ID
    t.Run("Invalid ID", func(t *testing.T) {
        count, err := ts.Repository.Delete(ctx, "invalid-id")
        
        assert.Error(t, err)
        assert.Equal(t, int64(0), count)
        assert.Contains(t, err.Error(), "invalid ID format")
    })
    
    // Test deletion of non-existent employee
    t.Run("Non-existent employee", func(t *testing.T) {
        nonExistentID := bson.NewObjectID().Hex()
        
        count, err := ts.Repository.Delete(ctx, nonExistentID)
        
        assert.NoError(t, err) // No error for valid but non-existent ID
        assert.Equal(t, int64(0), count) // But deleted count should be 0
    })
}

// Test DeleteAll method
func TestIntegrationDeleteAll(t *testing.T) {
    // Setup test environment
    ts := SetupIntegrationTest(t)
    defer ts.TeardownIntegrationTest(t)
    
    ctx := context.Background()
    
    // Test deletion from empty collection
    t.Run("Delete from empty collection", func(t *testing.T) {
        count, err := ts.Repository.DeleteAll(ctx)
        
        assert.NoError(t, err)
        assert.Equal(t, int64(0), count)
    })
    
    // Insert multiple employees
    employees := []models.Employee{
        {
            ID:         bson.NewObjectID(),
            Name:       "Delete All 1",
            Department: "Dept 1",
            Age:        30,
            Email:      "delete1@example.com",
        },
        {
            ID:         bson.NewObjectID(),
            Name:       "Delete All 2",
            Department: "Dept 2",
            Age:        35,
            Email:      "delete2@example.com",
        },
        {
            ID:         bson.NewObjectID(),
            Name:       "Delete All 3",
            Department: "Dept 3",
            Age:        40,
            Email:      "delete3@example.com",
        },
	}
    
    for _, emp := range employees {
        _, err := ts.Client.Database(ts.DBName).Collection(ts.Collection).InsertOne(ctx, emp)
        assert.NoError(t, err)
    }
    
    // Test deletion of all employees
    t.Run("Delete all employees", func(t *testing.T) {
        count, err := ts.Repository.DeleteAll(ctx)
        
        assert.NoError(t, err)
        assert.Equal(t, int64(3), count)
        
        // Verify collection is empty
        findResult, count, err := ts.Repository.FindAll(ctx, 1, 10)
        assert.NoError(t, err)
        assert.Equal(t, int64(0), count)
        assert.Empty(t, findResult) 
    })
}

// Test CheckEmailExists method
func TestIntegrationCheckEmailExists(t *testing.T) {
    // Setup test environment
    ts := SetupIntegrationTest(t)
    defer ts.TeardownIntegrationTest(t)
    
    ctx := context.Background()
    
    // Create test employee
    existingEmployee := models.Employee{
        ID:         bson.NewObjectID(),
        Name:       "Existing Employee",
        Department: "Email Test",
        Age:        32,
        Email:      "exists@example.com",
    }
    
    // Insert into database
    _, err := ts.Client.Database(ts.DBName).Collection(ts.Collection).InsertOne(ctx, existingEmployee)
    assert.NoError(t, err)
    
    // Test email exists
    t.Run("Email exists", func(t *testing.T) {
        exists, err := ts.Repository.CheckEmailExists(ctx, "exists@example.com")
        
        assert.NoError(t, err)
        assert.True(t, exists)
    })
    
    // Test email doesn't exist
    t.Run("Email doesn't exist", func(t *testing.T) {
        exists, err := ts.Repository.CheckEmailExists(ctx, "nonexistent@example.com")
        
        assert.NoError(t, err)
        assert.False(t, exists)
    })
    
	// Test with exclusion of ID
    t.Run("Exclude employee's own ID", func(t *testing.T) {
        exists, err := ts.Repository.CheckEmailExists(ctx, "exists@example.com", existingEmployee.ID.Hex())
        
        assert.NoError(t, err)
        assert.False(t, exists)
    })
    
    // Test with invalid ID to exclude
    t.Run("Invalid ID to exclude", func(t *testing.T) {
        _, err := ts.Repository.CheckEmailExists(ctx, "exists@example.com", "invalid-id")
        
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "invalid ID format")
    })
}

// Test Ping method
func TestIntegrationPing(t *testing.T) {
    // Setup test environment
    ts := SetupIntegrationTest(t)
    defer ts.TeardownIntegrationTest(t)
    
    ctx := context.Background()
    
    // Test successful ping
    t.Run("Successful ping", func(t *testing.T) {
        err := ts.Repository.Ping(ctx)
        
        assert.NoError(t, err)
    })
}
