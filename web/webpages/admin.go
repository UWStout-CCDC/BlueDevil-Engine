package webpages

// Handlers for admin-related pages.

import (
	sql_wrapper "BlueDevil-Engine/sql"
	structures "BlueDevil-Engine/structures"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/phpdave11/gofpdf"
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

// Admin API: list creates and uploads injects
func HandleApiInjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		injects, err := sql_wrapper.GetAllInjects()
		if err != nil {
			http.Error(w, "Failed to list injects: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(injects)
	case http.MethodPost:
		// create a new inject (metadata). Accept JSON
		var in structures.Inject
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if in.InjectID == "" {
			http.Error(w, "inject_id required", http.StatusBadRequest)
			return
		}
		if err := sql_wrapper.CreateInject(&in); err != nil {
			http.Error(w, "Failed to create inject: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(in)
	case http.MethodDelete:
		// Expect JSON body { "inject_id": "ID" }
		var req struct {
			InjectID string `json:"inject_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if req.InjectID == "" {
			http.Error(w, "inject_id required", http.StatusBadRequest)
			return
		}
		// delete DB record
		if err := sql_wrapper.DeleteInjectByInjectID(req.InjectID); err != nil {
			http.Error(w, "Failed to delete inject: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// delete PDF file if present
		fn := "injects/" + req.InjectID + ".pdf"
		_ = os.Remove(fn)
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Admin: upload a PDF for an inject (multipart form upload)
func HandleApiInjectUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Expect form fields: inject_id, file
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse multipart form: "+err.Error(), http.StatusBadRequest)
		return
	}
	injectID := r.FormValue("inject_id")
	if injectID == "" {
		http.Error(w, "inject_id required", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// ensure injects directory exists
	if err := ensureDir("injects"); err != nil {
		http.Error(w, "Failed to ensure injects dir: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// sanitize filename: use injectID + original extension
	fn := injectID + ".pdf"
	outPath := "injects/" + fn
	out, err := os.Create(outPath)
	if err != nil {
		http.Error(w, "Failed to create file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		http.Error(w, "Failed to write file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Update inject record filename
	inj, err := sql_wrapper.GetInjectByID(injectID)
	if err == nil && inj != nil {
		inj.Filename = fn
		sql_wrapper.CreateInject(inj) // ignore error here
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"filename": fn, "path": outPath, "original": header.Filename})
}

// Admin: list submissions for an inject
func HandleApiInjectSubmissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query().Get("inject_id")
	if q == "" {
		http.Error(w, "inject_id required", http.StatusBadRequest)
		return
	}
	subs, err := sql_wrapper.GetSubmissionsForInject(q)
	if err != nil {
		http.Error(w, "Failed to get submissions: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subs)
}

// Generate a simple PDF for an inject (server-side)
func HandleApiInjectGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		InjectID    string `json:"inject_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Deliverable string `json:"deliverable"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.InjectID == "" || req.Title == "" {
		http.Error(w, "inject_id and title required", http.StatusBadRequest)
		return
	}

	// generate PDF using templated layout (gofpdf)
	filename := req.InjectID + ".pdf"
	if err := ensureDir("injects"); err != nil {
		http.Error(w, "failed to create folder: "+err.Error(), http.StatusInternalServerError)
		return
	}
	path := "injects/" + filename
	if err := generateTemplatePDF(path, req.InjectID, req.Title, req.Description, req.Deliverable); err != nil {
		http.Error(w, "failed to generate PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// update inject record if exists
	inj, _ := sql_wrapper.GetInjectByID(req.InjectID)
	if inj != nil {
		inj.Filename = filename
		sql_wrapper.CreateInject(inj)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"filename": filename})
}

// Score a submission (admin)
func HandleApiInjectScore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID       int    `json:"id"`
		Scored   bool   `json:"scored"`
		Score    int    `json:"score"`
		Reviewer string `json:"reviewer"`
		Notes    string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	// load submission
	subs, err := sql_wrapper.GetSubmissionsForInject("")
	if err != nil {
		http.Error(w, "Failed to query: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var found *structures.InjectSubmission
	for i := range subs {
		if subs[i].ID == req.ID {
			found = &subs[i]
			break
		}
	}
	if found == nil {
		http.Error(w, "submission not found", http.StatusBadRequest)
		return
	}
	found.Scored = req.Scored
	found.Score = &req.Score
	found.Reviewer = req.Reviewer
	found.Notes = req.Notes
	if err := sql_wrapper.UpdateInjectSubmission(found); err != nil {
		http.Error(w, "failed to update: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HandleUserInjectPage serves the user-facing inject submission page.
func HandleUserInjectPage(w http.ResponseWriter, r *http.Request) {
	// Auth middleware should ensure the user is logged in. We simply render the template.
	tmpl, err := template.ParseFiles("templates/inject_submit.html")
	if err != nil {
		log.Printf("inject page template parse err: %v", err)
		http.Error(w, "internal", http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		log.Printf("inject page exec err: %v", err)
		http.Error(w, "internal", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

// generateTemplatePDF creates a templated inject PDF at `path` using the provided
// inject metadata. It uses tighter margins, a small info table near the top, and
// renders the description and deliverable sections.
func generateTemplatePDF(path, injectID, title, desc, deliverable string) error {
	// sanitize smart punctuation to ASCII equivalents so gofpdf's core fonts don't show mojibake
	title = sanitizeText(title)
	desc = sanitizeText(desc)
	deliverable = sanitizeText(deliverable)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(title, false)
	pdf.SetAuthor("CCDC", false)

	// tighter margins
	leftMargin := 22.0
	topMargin := 18.0
	rightMargin := 22.0
	pdf.SetMargins(leftMargin, topMargin, rightMargin)
	pdf.SetAutoPageBreak(true, 2)
	pdf.AddPage()

	// logo
	logoPath := "static/logo.png"
	logoW := 30.0
	if _, err := os.Stat(logoPath); err == nil {
		pdf.ImageOptions(logoPath, leftMargin, topMargin, logoW, 0, false, gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")
	}

	// Title to the right of logo
	pdf.SetFont("Helvetica", "B", 30)
	titleX := leftMargin + logoW + 8
	titleY := topMargin + 6
	pdf.SetXY(titleX, titleY)
	pdf.CellFormat(0, 10, "CCDC Inject", "", 1, "L", false, 0, "")

	// Info table: positioned further down to avoid logo overlap
	startX := leftMargin
	// increase vertical offset so the table sits well below the logo/title
	startY := topMargin + 50
	boxW := 185 - leftMargin - rightMargin
	rh := 9.0
	labelW := 32.0

	// draw outer rect for first row and second row combined
	pdf.SetDrawColor(180, 180, 180)
	pdf.SetLineWidth(0.3)
	pdf.Rect(startX, startY, boxW, rh*2, "D")

	// first row: label + value (with light fill background)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetXY(startX, startY)
	pdf.CellFormat(labelW, rh, "INJECT NAME", "LT", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 12)
	pdf.CellFormat(boxW-labelW, rh, title, "LTR", 1, "L", false, 0, "")

	// second row
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(245, 247, 249)
	pdf.CellFormat(labelW, rh, "INJECT ID", "LB", 0, "L", true, 0, "")
	pdf.SetFont("Helvetica", "", 12)
	pdf.SetFillColor(245, 247, 249)
	pdf.CellFormat(boxW-labelW, rh, injectID, "LBR", 1, "L", true, 0, "")

	// spacing
	pdf.Ln(8)

	// body: use the full width available, set via max page width minus margins
	bodyW := 215 - leftMargin - rightMargin
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(0, 7, "INJECT DESCRIPTION:", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 12)
	pdf.MultiCell(bodyW, 6, desc, "", "L", false)

	pdf.Ln(6)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(0, 7, "INJECT DELIVERABLE", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 12)
	if deliverable != "" {
		pdf.MultiCell(bodyW, 6, deliverable, "", "L", false)
	} else {
		pdf.MultiCell(bodyW, 6, "Respond with the requested deliverable as described by the inject.", "", "L", false)
	}

	// Footer with page number (smaller)
	pdf.SetY(-16)
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(0, 10, fmt.Sprintf("Page %d", pdf.PageNo()), "", 0, "C", false, 0, "")

	return pdf.OutputFileAndClose(path)
}

// sanitizeText replaces common smart punctuation with ASCII equivalents.
func sanitizeText(s string) string {
	if s == "" {
		return s
	}
	// common replacements: curly apostrophe, curly quotes, em/en dash, ellipsis
	s = strings.ReplaceAll(s, "’", "'")
	s = strings.ReplaceAll(s, "‘", "'")
	s = strings.ReplaceAll(s, "“", "\"")
	s = strings.ReplaceAll(s, "”", "\"")
	s = strings.ReplaceAll(s, "—", "-")
	s = strings.ReplaceAll(s, "–", "-")
	s = strings.ReplaceAll(s, "…", "...")
	return s
}

// helper: ensure directory exists
func ensureDir(p string) error {
	return os.MkdirAll(p, 0755)
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
			TeamID int `json:"team_id"`
			UserID int `json:"user_id"`
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
			TeamID int `json:"team_id"`
			UserID int `json:"user_id"`
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

// Score history for team/service
func HandleApiScoreHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	qTeam := r.URL.Query().Get("team_id")
	qSvc := r.URL.Query().Get("service_id")
	teamID := 0
	svcID := 0
	if qTeam != "" {
		if v, err := strconv.Atoi(qTeam); err == nil {
			teamID = v
		}
	}
	if qSvc != "" {
		if v, err := strconv.Atoi(qSvc); err == nil {
			svcID = v
		}
	}

	rows, err := sql_wrapper.GetCompetitionServiceHistory(teamID, svcID)
	if err != nil {
		http.Error(w, "Failed to get history: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rows)
}

// Add score adjustment
func HandleApiScoreAdjust(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		TeamID      int    `json:"team_id"`
		Score       int    `json:"score"`
		Round       int    `json:"round"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	id, err := sql_wrapper.AddCompetitionScoreAdjustment(req.TeamID, req.Score, req.Round, req.Description)
	if err != nil {
		http.Error(w, "Failed to add adjustment: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"id": id})
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
