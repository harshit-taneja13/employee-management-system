package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

var Client *mongo.Client

// ConnectDB establishes connection to MongoDB
func ConnectDB() error{
	// load .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Env load error", err)
	}
	log.Println("Env file loaded")


	clientOptions := options.Client().ApplyURI(os.Getenv("MONGO_URI"))
	
	client, err := mongo.Connect(clientOptions)
	
	if err != nil {
		return fmt.Errorf("Error connecting to MongoDB: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
	defer cancel()

	err = client.Ping(ctx , readpref.Primary())
	if err != nil {
		return fmt.Errorf("Error pinging MongoDB: %w", err)
	}
	
	Client = client
	log.Println("Connected to MongoDB!")

	return nil
}