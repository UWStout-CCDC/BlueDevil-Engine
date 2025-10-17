package webpages

// Handlers for admin-related pages.

import (
	sql_wrapper "BlueDevil-Engine/sql"
	structures "BlueDevil-Engine/structures"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
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
	case http.MethodPut:
		var t structures.Team
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if t.ID == 0 {
			http.Error(w, "Team ID is required for update", http.StatusBadRequest)
			return
		}
		if err := sql_wrapper.CreateTeam(&t); err != nil {
			http.Error(w, "Failed to update team: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(t)
	case http.MethodDelete:
		var req struct {
			ID int `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := sql_wrapper.DeleteTeam(req.ID); err != nil {
			http.Error(w, "Failed to delete team: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
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

// Users API
func HandleApiUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users, err := sql_wrapper.GetAllUsersWithTeams()
		if err != nil {
			http.Error(w, "Failed to get users: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	case http.MethodPost:
		// Assign user to team
		var req struct {
			UserID int `json:"user_id"`
			TeamID int `json:"team_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Remove user from any existing team first
		if err := sql_wrapper.RemoveUserFromAllTeams(req.UserID); err != nil {
			http.Error(w, "Failed to remove user from teams: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Add user to new team if team_id > 0
		if req.TeamID > 0 {
			if err := sql_wrapper.AddUserToTeam(req.TeamID, req.UserID); err != nil {
				http.Error(w, "Failed to add user to team: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Competition API
func HandleApiCompetition(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		comp, err := sql_wrapper.GetCompetition()
		if err != nil {
			http.Error(w, "Failed to get competition: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(comp)
	case http.MethodPost:
		var req struct {
			Action        string `json:"action"` // "schedule", "start", "stop", "reset"
			ScheduledTime string `json:"scheduled_time,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		comp, err := sql_wrapper.GetCompetition()
		if err != nil {
			http.Error(w, "Failed to get competition: "+err.Error(), http.StatusInternalServerError)
			return
		}

		switch req.Action {
		case "schedule":
			comp.Status = "scheduled"
			comp.ScheduledTime = req.ScheduledTime
		case "start":
			comp.Status = "running"
			comp.StartedTime = time.Now().Format(time.RFC3339)
		case "stop":
			comp.Status = "stopped"
			comp.StoppedTime = time.Now().Format(time.RFC3339)
		case "reset":
			// Reset all competition scoring data, service checks, and scores
			if err := sql_wrapper.ResetAllScoringData(); err != nil {
				http.Error(w, "Failed to reset scoring data: "+err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"message": "All scoring data reset successfully"})
			return
		default:
			http.Error(w, "Invalid action", http.StatusBadRequest)
			return
		}

		if err := sql_wrapper.UpdateCompetition(comp); err != nil {
			http.Error(w, "Failed to update competition: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(comp)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Service Matrix API - returns data for the dashboard service configuration matrix
func HandleApiServiceMatrix(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all teams, services, and scoring boxes
	teams, err := sql_wrapper.GetAllTeams()
	if err != nil {
		http.Error(w, "Failed to get teams: "+err.Error(), http.StatusInternalServerError)
		return
	}

	services, err := sql_wrapper.GetAllServices()
	if err != nil {
		http.Error(w, "Failed to get services: "+err.Error(), http.StatusInternalServerError)
		return
	}

	boxes, err := sql_wrapper.GetAllScoringBoxes()
	if err != nil {
		http.Error(w, "Failed to get boxes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Build a map of team_id -> service_id -> box for quick lookup
	boxMap := make(map[int]map[int]*structures.ScoringBox)
	for i := range boxes {
		box := &boxes[i]
		if _, exists := boxMap[box.TeamID]; !exists {
			boxMap[box.TeamID] = make(map[int]*structures.ScoringBox)
		}
		boxMap[box.TeamID][box.ServiceID] = box
	}

	// Build response
	response := struct {
		Teams    []structures.Team                      `json:"teams"`
		Services []structures.Service                   `json:"services"`
		BoxMap   map[int]map[int]*structures.ScoringBox `json:"box_map"`
	}{
		Teams:    teams,
		Services: services,
		BoxMap:   boxMap,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleApiInfo returns informational tables: service IP scheme, default passwords, and
// environment login info for a given team (dynamic per-team content). The handler accepts
// an optional query param `team_id` to scope env login info to a team.
func HandleApiInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// optional team_id query param
	q := r.URL.Query().Get("team_id")
	teamID := 0
	if q != "" {
		if v, err := strconv.Atoi(q); err == nil {
			teamID = v
		}
	}

	// Fetch services for generating service-specific entries
	services, _ := sql_wrapper.GetAllServices()

	// Build placeholder service IP scheme table
	var svcIPScheme []map[string]string
	for _, s := range services {
		svcIPScheme = append(svcIPScheme, map[string]string{
			"service": s.Name,
			"scheme":  fmt.Sprintf("10.%d.%d.x (team.%d.service.%s)", 0, s.ID, teamID, s.Name),
		})
	}
	if len(svcIPScheme) == 0 {
		// add some lorem placeholder rows
		svcIPScheme = append(svcIPScheme, map[string]string{"service": "Service-A", "scheme": "10.TEAM.SVC.x"})
		svcIPScheme = append(svcIPScheme, map[string]string{"service": "Service-B", "scheme": "10.TEAM.SVC.y"})
	}

	// Default passwords placeholder
	defaultPw := []map[string]string{
		{"account": "admin", "password": "Password123!", "notes": "Lorem ipsum"},
		{"account": "root", "password": "toor", "notes": "Lorem ipsum"},
	}

	// Environment login info (dynamic per team)
	var envLogins []map[string]string
	if teamID == 0 {
		// if no team specified, include a sample entry
		envLogins = append(envLogins, map[string]string{"service": "Example", "url": "http://example.team.local/", "username": "team1", "password": "pass-1"})
	} else {
		// generate one login per service for this team
		for _, s := range services {
			envLogins = append(envLogins, map[string]string{
				"service":  s.Name,
				"url":      fmt.Sprintf("https://%s.team%d.example.com", s.Name, teamID),
				"username": fmt.Sprintf("team%d_%s_user", teamID, s.Name),
				"password": fmt.Sprintf("pwd-%d-%s", teamID, s.Name),
			})
		}
		if len(envLogins) == 0 {
			envLogins = append(envLogins, map[string]string{"service": "Example", "url": "http://example.team.local/", "username": fmt.Sprintf("team%d", teamID), "password": "pass-1"})
		}
	}

	resp := map[string]interface{}{
		"service_ip_scheme": svcIPScheme,
		"default_passwords": defaultPw,
		"env_logins":        envLogins,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
