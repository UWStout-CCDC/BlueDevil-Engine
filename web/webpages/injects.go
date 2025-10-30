package webpages

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	sql_wrapper "BlueDevil-Engine/sql"
	structures "BlueDevil-Engine/structures"
)

// HandleInjectsPage renders the list of released injects (server-side)
func HandleInjectsPage(w http.ResponseWriter, r *http.Request) {
	log.Println("Serving injects list page")

	// user info
	var isLoggedIn, isAdmin bool
	var userName string
	if u := r.Context().Value(CtxUserKey); u != nil {
		switch v := u.(type) {
		case structures.User:
			isLoggedIn = true
			isAdmin = v.Is_Admin
			userName = v.Name
		case *structures.User:
			isLoggedIn = true
			isAdmin = v.Is_Admin
			userName = v.Name
		}
	}

	injects, err := sql_wrapper.GetAllInjects()
	if err != nil {
		log.Println("Failed to load injects:", err)
		http.Error(w, "failed to load injects", http.StatusInternalServerError)
		return
	}

	// filter to released only (ReleaseTime is minutes after competition start)
	now := time.Now().UTC()
	comp, _ := sql_wrapper.GetCompetition()

	visible := []structures.Inject{}
	for _, in := range injects {
		// Admins should see everything
		var isAdminLocal bool
		if u := r.Context().Value(CtxUserKey); u != nil {
			switch v := u.(type) {
			case structures.User:
				isAdminLocal = v.Is_Admin
			case *structures.User:
				isAdminLocal = v.Is_Admin
			}
		}
		if injectVisibleToUser(&in, comp, now, isAdminLocal) {
			visible = append(visible, in)
		}
	}

	data := map[string]interface{}{
		"Active":     "injects",
		"IsAdmin":    isAdmin,
		"IsLoggedIn": isLoggedIn,
		"UserName":   userName,
		"Injects":    visible,
	}

	tmpl, err := template.ParseFiles("templates/injects.html")
	if err != nil {
		log.Println("injects template parse error:", err)
		http.ServeFile(w, r, "templates/injects.html")
		return
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Println("injects template exec error:", err)
		http.Error(w, "template render error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

// HandleInjectViewPage renders a single inject view (and submit form when logged in)
func HandleInjectViewPage(w http.ResponseWriter, r *http.Request, injectID string) {
	log.Println("Serving inject view", injectID)

	var isLoggedIn, isAdmin bool
	var userName string
	if u := r.Context().Value(CtxUserKey); u != nil {
		switch v := u.(type) {
		case structures.User:
			isLoggedIn = true
			isAdmin = v.Is_Admin
			userName = v.Name
		case *structures.User:
			isLoggedIn = true
			isAdmin = v.Is_Admin
			userName = v.Name
		}
	}

	in, err := sql_wrapper.GetInjectByID(injectID)
	if err != nil || in == nil {
		http.NotFound(w, r)
		return
	}

	// enforce visibility: non-admins cannot view unreleased injects
	var isAdminLocal bool
	if u := r.Context().Value(CtxUserKey); u != nil {
		switch v := u.(type) {
		case structures.User:
			isAdminLocal = v.Is_Admin
		case *structures.User:
			isAdminLocal = v.Is_Admin
		}
	}
	comp, _ := sql_wrapper.GetCompetition()
	if !injectVisibleToUser(in, comp, time.Now().UTC(), isAdminLocal) {
		http.NotFound(w, r)
		return
	}

	data := map[string]interface{}{
		"Active":     "injects",
		"IsAdmin":    isAdmin,
		"IsLoggedIn": isLoggedIn,
		"UserName":   userName,
		"Inject":     in,
	}

	tmpl, err := template.ParseFiles("templates/inject_view.html")
	if err != nil {
		log.Println("inject view template parse error:", err)
		http.ServeFile(w, r, "templates/inject_view.html")
		return
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Println("inject view template exec error:", err)
		http.Error(w, "template render error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

// HandleInjectsRoot inspects the path under /injects/ and either serves the
// list, a single inject view, or forwards to the static file server for PDFs.
func HandleInjectsRoot(w http.ResponseWriter, r *http.Request) {
	// Trim the prefix to get the remainder
	p := strings.TrimPrefix(r.URL.Path, "/injects/")
	if p == "" || p == "/" || p == "/injects" {
		HandleInjectsPage(w, r)
		return
	}
	// If it looks like a file (has a dot), delegate to the static file server
	if strings.Contains(p, ".") {
		fs := http.StripPrefix("/injects/", http.FileServer(http.Dir("injects")))
		fs.ServeHTTP(w, r)
		return
	}
	// Otherwise treat as inject id and serve view
	// ensure no trailing slash
	id := strings.TrimSuffix(p, "/")
	HandleInjectViewPage(w, r, id)
}

// User endpoint: submit an inject PDF (authenticated users submit to team folder)
func HandleApiSubmitInject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// user must be in context (AuthPromptMiddleware provides it)
	userCtx := r.Context().Value(CtxUserKey)
	if userCtx == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// Expect multipart form: inject_id, file
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

	// Extract user from context (we expect a structures.User stored under CtxUserKey)
	var subject string
	if u := r.Context().Value(CtxUserKey); u != nil {
		switch v := u.(type) {
		case structures.User:
			subject = v.Subject
		case *structures.User:
			subject = v.Subject
		case map[string]interface{}:
			if s, ok := v["sub"].(string); ok {
				subject = s
			}
		}
	}

	// Determine team for user
	teamID := 0
	if subject != "" {
		if t, err := sql_wrapper.GetUserTeamBySubject(subject); err == nil && t != nil {
			teamID = t.ID
		}
	}

	// Ensure the inject is visible to this user (don't allow submitting for unreleased injects)
	inj, _ := sql_wrapper.GetInjectByID(injectID)
	comp, _ := sql_wrapper.GetCompetition()
	var isAdminLocal bool
	if u := r.Context().Value(CtxUserKey); u != nil {
		switch v := u.(type) {
		case structures.User:
			isAdminLocal = v.Is_Admin
		case *structures.User:
			isAdminLocal = v.Is_Admin
		}
	}
	if !injectVisibleToUser(inj, comp, time.Now().UTC(), isAdminLocal) {
		http.Error(w, "inject not available", http.StatusForbidden)
		return
	}

	// ensure submissions dir exists
	dir := fmt.Sprintf("submissions/%d", teamID)
	if err := ensureDir(dir); err != nil {
		http.Error(w, "Failed to ensure submissions dir: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// save file as {injectID}_{team}_{timestamp}.pdf
	ts := time.Now().Unix()
	fn := fmt.Sprintf("%s_team%d_%d.pdf", injectID, teamID, ts)
	outPath := dir + "/" + fn
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

	// record submission in DB
	sub := &structures.InjectSubmission{
		InjectID: injectID,
		TeamID:   teamID,
		Filename: fn,
	}
	if err := sql_wrapper.AddInjectSubmission(sub); err != nil {
		http.Error(w, "Failed to record submission: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": sub.ID, "filename": fn, "path": outPath, "original": header.Filename})
}

// injectVisibleToUser returns true if the inject should be visible to a non-admin user
// given the competition state and current time. Admins should bypass this check.
func injectVisibleToUser(in *structures.Inject, comp *structures.Competition, now time.Time, isAdmin bool) bool {
	if in == nil {
		return false
	}
	if isAdmin {
		return true
	}

	// If ReleaseTime is zero, it's immediately visible
	if in.ReleaseTime == 0 {
		return true
	}

	// If competition has a started_time, interpret ReleaseTime as minutes after start
	if comp != nil && comp.StartedTime != "" {
		if t, err := time.Parse(time.RFC3339, comp.StartedTime); err == nil {
			release := t.UTC().Add(time.Duration(in.ReleaseTime) * time.Minute)
			if !release.After(now) {
				return true
			}
			return false
		}
	}

	// If competition isn't started (no valid started_time), hide future releases.
	// We treat the inject as unreleased until a competition start is known.
	return false
}
