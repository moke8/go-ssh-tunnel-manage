package api

import (
	"encoding/json"
	"net/http"
	"ssh-manage/models"
	"ssh-manage/services"
	"time"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	switch r.URL.Path {
	case "/api/users":
		handleUsers(w, r)
	case "/api/connections":
		handleConnections(w, r)
	case "/api/stats":
		handleStats(w, r)
	default:
		http.NotFound(w, r)
	}
}

func handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users := services.GetAllUsers()
		json.NewEncoder(w).Encode(users)
	case http.MethodPost:
		var user models.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		user.Created = time.Now()
		user.Active = true
		
		if err := services.AddUser(&user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	connections := services.GetAllConnections()
	json.NewEncoder(w).Encode(connections)
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	stats := services.GetStatistics()
	json.NewEncoder(w).Encode(stats)
}