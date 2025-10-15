package webpages

// Handlers for admin-related pages.

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
)

// In-memory services store for demo/editing purposes. In production this should be
// backed by persistent storage (database).
var (
	servicesLock sync.RWMutex
	servicesData = []map[string]interface{}{
		{
			"id":   1,
			"name": "Service A",
			"host": "192.0.2.10",
			"checks": []map[string]interface{}{
				{"id": "1", "name": "SSH check", "command": "nc -zv localhost 22"},
				{"id": "2", "name": "Disk space", "command": "df -h /"},
			},
		},
		{
			"id":   2,
			"name": "Service B",
			"host": "198.51.100.5",
			"checks": []map[string]interface{}{
				{
					"id":      "1",
					"name":    "HTTP check",
					"command": "curl -fsS https://example.com/health",
					"regexes": []map[string]interface{}{
						{"id": "1", "pattern": "<title>(.*?)</title>", "description": "Page title"},
						{"id": "2", "pattern": "<h1.*?>(.*?)</h1>", "description": "First heading"},
					},
				},
				{"id": "2", "name": "DB connectivity", "command": "pg_isready -h dbhost -p 5432"},
			},
		},
		{
			"id":   3,
			"name": "Service C",
			"checks": []map[string]interface{}{
				{"id": "1", "name": "Ping", "command": "ping -c1 8.8.8.8"},
			},
		},
	}
)

// In-memory credentials store (for demo). Keyed by credential id or name.
var (
	credsLock sync.RWMutex
	credsData = map[string]map[string]string{
		// example: "shared-db": {"username":"admin","password":"s3cr3t"}
	}
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

func HandleCreateInjects(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/admin.html")
}

func HandleScoreInjects(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/admin.html")
}

func HandleCompetitionSettings(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/admin.html")
}

func GetServicesHandler(w http.ResponseWriter, r *http.Request) {
	// Return the in-memory services list as JSON
	w.Header().Set("Content-Type", "application/json")
	servicesLock.RLock()
	defer servicesLock.RUnlock()
	if err := json.NewEncoder(w).Encode(servicesData); err != nil {
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
	var svc map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&svc); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	servicesLock.Lock()
	defer servicesLock.Unlock()

	// If id is present, update; otherwise append with a new id (max+1)
	if idv, ok := svc["id"]; ok && idv != nil {
		// find and replace
		for i, s := range servicesData {
			if s["id"] == idv {
				servicesData[i] = svc
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(svc)
				return
			}
		}
		// not found; append
		servicesData = append(servicesData, svc)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(svc)
		return
	}

	// assign new id
	maxID := 0
	for _, s := range servicesData {
		if idnum, ok := s["id"].(float64); ok {
			if int(idnum) > maxID {
				maxID = int(idnum)
			}
		}
		if idnum, ok := s["id"].(int); ok {
			if idnum > maxID {
				maxID = idnum
			}
		}
	}
	svc["id"] = maxID + 1
	servicesData = append(servicesData, svc)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(svc)
}

// DeleteServiceHandler deletes a service by id.
func DeleteServiceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ID interface{} `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	servicesLock.Lock()
	defer servicesLock.Unlock()
	for i, s := range servicesData {
		if s["id"] == body.ID {
			servicesData = append(servicesData[:i], servicesData[i+1:]...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// try numeric compare
		if idnum, ok := s["id"].(float64); ok {
			if idnum == body.ID {
				servicesData = append(servicesData[:i], servicesData[i+1:]...)
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
	}
	http.Error(w, "Service not found", http.StatusNotFound)
}

func GetTeamsHandler(w http.ResponseWriter, r *http.Request) {
	// Placeholder: Return a list of teams in JSON format.
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`[
		{"id": 1, "name": "Team Alpha"},
		{"id": 2, "name": "Team Beta"}
	]`))
}

func GetUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Placeholder: Return a list of users in JSON format.
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`[
		{"id": 1, "username": "admin", "role": "administrator"},
		{"id": 2, "username": "user1", "role": "user"}
	]`))
}

func GetMappingsHandler(w http.ResponseWriter, r *http.Request) {
	// Placeholder: Return a list of box mappings in JSON format.
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`[
		{"id": 1, "box": "Box 1", "team": "Team Alpha"},
		{"id": 2, "box": "Box 2", "team": "Team Beta"}
	]`))
}

func GetScoresHandler(w http.ResponseWriter, r *http.Request) {
	// Placeholder: Return a list of scores in JSON format.
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`[
		{"team": "Team Alpha", "score": 150},
		{"team": "Team Beta", "score": 120}
	]`))
}
