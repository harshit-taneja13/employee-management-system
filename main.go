package main

import (
	"employee-management/database"
	"employee-management/routes"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main () {
	// connect to mongoDB
	err := database.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
		
	// Set up the router
	router := routes.SetupRoutes()

	PORT := os.Getenv("PORT")
	fmt.Println("Server is running on port " + PORT + "...")
	err = http.ListenAndServe(":" + PORT, router)
	if err != nil {
		log.Fatal("Server error:", err)
	}
		
}