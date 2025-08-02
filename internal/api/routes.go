package api

import (
	"github.com/gorilla/mux"
)

func RegisterRoutes(router *mux.Router) {
	handler := NewHandler()

	router.HandleFunc("/player/{steamid}/summary", handler.GetPlayerSummary).Methods("GET")
	router.HandleFunc("/player/{steamid}/stats", handler.GetPlayerStats).Methods("GET")
}
