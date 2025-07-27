package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/gorilla/mux"
	handler "github.com/rgonzalez12/dbd-analytics/internal/handlers"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	r := mux.NewRouter()

	// home route
	r.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Stats and Analytics coming soon...")
	})

	// dynamic route
	r.HandleFunc("/api/player/{steamID}", handler.GetPlayerStats).Methods("GET")

	fmt.Println("Starting dbd-analytics on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
