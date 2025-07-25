package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Stats and Analytics coming soon...")
	})

	fmt.Println("Starting dbd-analytics on :8080")
	http.ListenAndServe(":8080", nil)

}
