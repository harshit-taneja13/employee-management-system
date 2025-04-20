package repository

import (
	"context"
	"employee-management/models"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MongoEmployeeRepository struct {
	client     *mongo.Client
	database   string
	collection string
}

func NewMongoEmployeeRepository(client *mongo.Client, database, collection string) EmployeeRepository {
	return &MongoEmployeeRepository{
		client:     client,
		database:   database,
		collection: collection,
	}
}

// getCollection returns all the mongoDB collection
func (r *MongoEmployeeRepository) getCollection() *mongo.Collection {
	return r.client.Database(r.database).Collection(r.collection)
}

type EmployeeRepository interface {
	FindAll(ctx context.Context, page, limit int) ([]models.Employee, int64, error)
	FindByID(ctx context.Context, id string) (models.Employee, error)
	Create(ctx context.Context, employee models.Employee) (interface{}, error)
	Update(ctx context.Context, id string, employee models.Employee) (int64, error)
	Delete(ctx context.Context, id string) (int64, error)
	DeleteAll(ctx context.Context) (int64, error)
	CheckEmailExists(ctx context.Context, email string, excludeID ...string) (bool, error)
	Ping(ctx context.Context) error
}
