package webpages

// Handlers for admin-related pages.

import (
	sql_wrapper "BlueDevil-Engine/sql"
	structures "BlueDevil-Engine/structures"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func HandleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	// Read in the admin dashboard template and render it.
	log.Println("Serving admin dashboard")
	http.ServeFile(w, r, "templates/admin.html")
}

func HandleManageUsers(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/admin.html")
}

func HandleManageTeams(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/admin.html")
}

func HandleManageServices(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/admin.html")
}

func HandleManageMappings(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/admin.html")
}

func HandleManageScoring(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/admin.html")
}

func HandleManageInjects(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/admin.html")
}

func HandleCompetitionSettings(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/admin.html")
}

func HandleApiServices(w http.ResponseWriter, r *http.Request) {
	// If GET, return list of services, if POST, save a service., if DELETE, delete a service.
	log.Println("API Services endpoint hit with method " + r.Method)
	switch r.Method {
	case http.MethodGet:
		GetServicesHandler(w, r)
	case http.MethodPost:
		SaveServiceHandler(w, r)
	case http.MethodDelete:
		DeleteServiceHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func GetServicesHandler(w http.ResponseWriter, r *http.Request) {
	services, err := sql_wrapper.GetAllServices()
	if err != nil {
		http.Error(w, "Failed to get services: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if services == nil {
		services = []structures.Service{}
	}

	if err != nil {
		http.Error(w, "Failed to get services: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(services); err != nil {
		http.Error(w, "Failed to encode services: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// SaveServiceHandler accepts a JSON object for a service and either creates or updates it in memory.
func SaveServiceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var svc structures.Service
	if err := json.NewDecoder(r.Body).Decode(&svc); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	log.Println("Saving Service following json" + svc.Name)

	if err := sql_wrapper.SaveService(&svc); err != nil {
		http.Error(w, "Failed to save service: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(svc); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// DeleteServiceHandler deletes a service by id.
func DeleteServiceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := sql_wrapper.DeleteServiceByID(req.ID); err != nil {
		http.Error(w, "Failed to delete service: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Teams API
func HandleApiTeams(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		teams, err := sql_wrapper.GetAllTeams()
		if err != nil {
			http.Error(w, "Failed to get teams: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(teams)
	case http.MethodPost:
		var t structures.Team
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := sql_wrapper.CreateTeam(&t); err != nil {
			http.Error(w, "Failed to create team: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(t)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Team members API (simple endpoints)
func HandleTeamMembers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// expects ?team_id=NN
		q := r.URL.Query().Get("team_id")
		if q == "" {
			http.Error(w, "team_id required", http.StatusBadRequest)
			return
		}
		var teamID int
		_, err := fmt.Sscanf(q, "%d", &teamID)
		if err != nil {
			http.Error(w, "invalid team_id", http.StatusBadRequest)
			return
		}
		users, err := sql_wrapper.GetUsersInTeam(teamID)
		if err != nil {
			http.Error(w, "Failed to get users: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	case http.MethodPost:
		var req struct {
			TeamID, UserID int `json:"team_id" json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := sql_wrapper.AddUserToTeam(req.TeamID, req.UserID); err != nil {
			http.Error(w, "Failed to add user to team: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		var req struct {
			TeamID, UserID int `json:"team_id" json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := sql_wrapper.RemoveUserFromTeam(req.TeamID, req.UserID); err != nil {
			http.Error(w, "Failed to remove user from team: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Scoring boxes API
func HandleApiBoxes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		boxes, err := sql_wrapper.GetAllScoringBoxes()
		if err != nil {
			http.Error(w, "Failed to get boxes: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(boxes)
	case http.MethodPost:
		var b structures.ScoringBox
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := sql_wrapper.SaveScoringBox(&b); err != nil {
			http.Error(w, "Failed to save box: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(b)
	case http.MethodDelete:
		var req struct {
			ID int `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := sql_wrapper.DeleteScoringBox(req.ID); err != nil {
			http.Error(w, "Failed to delete box: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
