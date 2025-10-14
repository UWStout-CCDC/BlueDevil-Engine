package webpages

// Handlers for admin-related pages.

import (
	"net/http"
)

func HandleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Admin Dashboard - Under Construction"))
}

func HandleManageUsers(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Manage Users - Under Construction"))
}

func HandleManageTeams(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Manage Teams - Under Construction"))
}

func HandleManageServices(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Manage Services - Under Construction"))
}

func HandleManageMappings(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Manage Mappings - Under Construction"))
}

func HandleManageScoring(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Manage Scoring - Under Construction"))
}

func HandleCreateInjects(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Create Injects - Under Construction"))
}

func HandleScoreInjects(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Score Injects - Under Construction"))
}

func HandleCompetitionSettings(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Competition Settings - Under Construction"))
}
