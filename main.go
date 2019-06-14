package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/pulsejet/cerium/controllers"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	router := mux.NewRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	fmt.Println(port)
	rand.Seed(time.Now().UnixNano())

	router.HandleFunc("/api/form", controllers.CreateForm).Methods("POST")
	router.HandleFunc("/api/form/{id}", controllers.CreateForm).Methods("PUT")
	router.HandleFunc("/api/form/{id}", controllers.GetForm).Methods("GET")
	router.HandleFunc("/api/response/{formid}", controllers.CreateResponse).Methods("POST")
	router.HandleFunc("/api/response/{formid}", controllers.GetResponses).Methods("GET")

	router.HandleFunc("/api/login", controllers.Login).Methods("POST", "GET")

	err = http.ListenAndServe(":"+port, router)
	if err != nil {
		fmt.Print(err)
	}
}
