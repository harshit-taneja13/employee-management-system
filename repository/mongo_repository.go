package repository

import (
	"context"
	"employee-management/models"
	"errors"
	"fmt"
	"strings"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// This function finds all the employees in the model acc to page and limit
// Parameters:
// 		context,
// 		page,
// 		limit
// Returns:
// 		[]models.Employee: Slice containing all employees
//   	error: Any error encountered during retrieval
func (r *MongoEmployeeRepository) FindAll(ctx context.Context, page, limit int) ([]models.Employee, int64, error){
	var employees []models.Employee
	skip := (page -1) *limit

	// Define options with pagination and sorting
	findOptions := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(skip)).
		SetSort(bson.D{{"name", 1}})
	
	cursor, err := r.getCollection().Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("error decoding employees: %w", err)
	}	
	defer cursor.Close(ctx)

	err = cursor.All(ctx, &employees)
	if err != nil {
		return nil, 0, fmt.Errorf("error decoding employees : %w", err)
	}

	count, err := r.getCollection().CountDocuments(ctx, bson.M{})
	if err != nil {
		return employees, 0, fmt.Errorf("error counting documents %w", err)
	}

	return employees, count, nil
}


func (r *MongoEmployeeRepository) FindByID(ctx context.Context, id string) (models.Employee, error ){
	var employee models.Employee

	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return employee, fmt.Errorf("invalid ID format : %w", err)
	}

	err = r.getCollection().FindOne(ctx, bson.M{"_id" : objectID} ).Decode(&employee)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return employee, fmt.Errorf("employee not found: %w", err )
		}
		return employee, fmt.Errorf("database Error : %w", err)
	}
	return employee,nil
}

// Create adds a new employee to the database
// Parameters:
// 		context,
// 		employee: The employee model to be inserted
// Returns:
// 		insertedid - the usnique id of the inserted employee,
//   	error: Any error encountered during insertion
func (r *MongoEmployeeRepository) Create(ctx context.Context, employee models.Employee) (interface{}, error){
	result, err := r.getCollection().InsertOne(ctx , employee)
	if err != nil {
		return nil, fmt.Errorf("error creating employee : %w", err )
	}
	return result.InsertedID, nil
}

func (r *MongoEmployeeRepository) Update(ctx context.Context, id string, employee models.Employee) (int64, error ){
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return 0, fmt.Errorf("invalid ID format : %w", err)
	}
	update := bson.M{}

	if strings.TrimSpace(employee.Name) != "" {
		update["name"] = employee.Name
	}
	if strings.TrimSpace(employee.Email) != "" {
		update["email"] = employee.Email
	}
	if strings.TrimSpace(employee.Department) != "" {
		update["department"] = employee.Department
	}
	if employee.Age > 0 {
		update["age"] = employee.Age
	} 

	result, err := r.getCollection().UpdateOne(ctx, bson.M{"_id" : objectID} , bson.M{"$set" : update})
	if err != nil {
		return 0, fmt.Errorf("error updating employee: %w", err)
	}
	return result.ModifiedCount,nil
}

func (r *MongoEmployeeRepository) Delete(ctx context.Context, id string)(int64, error){
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return 0, fmt.Errorf("invalid ID format: %w", err)
	}
	result, err := r.getCollection().DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return 0, fmt.Errorf("error deleting all employees : %w", err)
	}
	return result.DeletedCount, nil
}

func (r *MongoEmployeeRepository) DeleteAll(ctx context.Context) (int64, error){
	result, err := r.getCollection().DeleteMany(ctx, bson.M{})
	if err != nil{
		return 0, fmt.Errorf("error deleting all employees : %w", err)
	}
	return result.DeletedCount,nil
}


// CheckEmailExists verifies if an email is already in use
// If excludeID is provided, it will exclude that employee from the check (for updates)
func (r* MongoEmployeeRepository) CheckEmailExists(ctx context.Context, email string, excludeID ...string)(bool, error){
	filter := bson.M{"email": email}
	if len(excludeID) > 0 && excludeID[0] != "" {
		objectID, err := bson.ObjectIDFromHex(excludeID[0])
		if err != nil {
			return false, fmt.Errorf("invalid ID format : %w", err)
		}
		filter = bson.M{
			"email": email,
			// here we are checking id != excluded id
			"_id": bson.M{"$ne" :objectID},
		}
	}
	count, err := r.getCollection().CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("error checking email existence: %w", err)
	}
	return count>0, nil
}

func (r *MongoEmployeeRepository) Ping(ctx context.Context) error {
    return r.client.Ping(ctx, readpref.Primary())
}