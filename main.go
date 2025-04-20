package main

import (
	"employee-management/controllers"
	"employee-management/database"
	"employee-management/repository"
	"employee-management/routes"
	"employee-management/services"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main () {
	// load .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Env load error", err)
	}
	log.Println("Env file loaded")

	// connect to mongoDB
	err = database.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	
	empRepo := repository.NewMongoEmployeeRepository(database.Client, os.Getenv("DATABASE_NAME"), os.Getenv("COLLECTION_NAME"))
	empService := services.NewEmployeeService(empRepo)
	empController := controllers.NewEmployeeController(empService)
	// Set up the router
	router := routes.SetupRoutes(empController)

	PORT := os.Getenv("PORT")
	fmt.Println("Server is running on port " + PORT + "...")
	err = http.ListenAndServe(":" + PORT, router)
	if err != nil {
		log.Fatal("Server error:", err)
	}
		
}