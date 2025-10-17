package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"crypto/rand"
	"encoding/base64"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"

	sql_wrapper "BlueDevil-Engine/sql"
	structures "BlueDevil-Engine/structures"
	webpages "BlueDevil-Engine/webpages"
)

// Add `github.com/joho/godotenv` to your imports.
var (
	clientID     string
	clientSecret string
	redirectURL  string
	providerURL  string

	provider     *oidc.Provider
	oidcConfig   *oidc.Config
	oauth2Config *oauth2.Config

	states = make(map[string]bool) // In-memory state store for demo purposes

	adminGroup string
)

// Note: request user context key is provided by the webpages package (webpages.CtxUserKey)

func generateState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func init() {
	// Load .env (optional). If .env is absent, we simply fall back to existing env vars.
	_ = godotenv.Load()

	clientID = os.Getenv("OIDC_CLIENT_ID")
	clientSecret = os.Getenv("OIDC_CLIENT_SECRET")

	if v := os.Getenv("REDIRECT_URL"); v != "" {
		redirectURL = v
	}

	if clientID == "" || clientSecret == "" {
		log.Fatal("OIDC_CLIENT_ID and OIDC_CLIENT_SECRET must be set")
	}

	providerURL = os.Getenv("OIDC_ISSUER_URL")
	if providerURL == "" {
		log.Fatal("OIDC_ISSUER_URL must be set")
	}

	redirectURL = os.Getenv("REDIRECT_URL")
	if redirectURL == "" {
		redirectURL = "http://localhost:8000/callback"
	}

	adminGroup = os.Getenv("ADMIN_GROUP")
}

func main() {
	// Initialize the database
	// Read DB settings from environment with sensible defaults.
	dbDriver := os.Getenv("DB_DRIVER")
	if dbDriver == "" {
		dbDriver = os.Getenv("DATABASE_DRIVER")
	}
	if dbDriver == "" {
		dbDriver = "sqlite3"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = os.Getenv("DATABASE_DSN")
	}
	if dbPath == "" {
		dbPath = "./blue_devil.db"
	}
	if err := sql_wrapper.InitDB(dbDriver, dbPath); err != nil {
		log.Fatal(err)
	}
	if err := sql_wrapper.CreateTables(); err != nil {
		log.Fatal(err)
	}
	// Ensure the database is closed on exit
	defer sql_wrapper.CloseDB()

	ctx := context.Background()

	var err error
	// Discover OIDC configuration
	provider, err = oidc.NewProvider(ctx, providerURL)
	if err != nil {
		log.Fatal(err)
	}

	oidcConfig = &oidc.Config{
		ClientID: clientID,
	}

	// OAuth2 config
	oauth2Config = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  redirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	http.HandleFunc("/", webpages.HandleHomepage)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/login-user", handleLoginUser) // login for user, but does not have redirect cookie
	http.HandleFunc("/callback", handleCallback)
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		// Clear the ID token cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "id_token",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   false, // set false for localhost dev
			MaxAge:   -1,
		})
		http.Redirect(w, r, "/", http.StatusFound)
	})

	http.HandleFunc("/scoreboard", handleScoreboard)

	// Public standalone info page (derived from homepage)
	// Use AuthPromptMiddleware so unauthenticated users see a friendly login prompt
	http.Handle("/info", AuthPromptMiddleware(http.HandlerFunc(webpages.HandleInfoPage)))

	// Admin routes
	http.Handle("/admin/", AuthMiddleware(AdminAuthMiddleware(http.HandlerFunc(webpages.HandleAdminDashboard))))
	http.Handle("/admin/users", AuthMiddleware(AdminAuthMiddleware(http.HandlerFunc(webpages.HandleManageUsers))))
	http.Handle("/admin/teams", AuthMiddleware(AdminAuthMiddleware(http.HandlerFunc(webpages.HandleManageTeams))))
	http.Handle("/admin/services", AuthMiddleware(AdminAuthMiddleware(http.HandlerFunc(webpages.HandleManageServices))))
	http.Handle("/admin/box-mapping", AuthMiddleware(AdminAuthMiddleware(http.HandlerFunc(webpages.HandleManageMappings))))
	http.Handle("/admin/scores", AuthMiddleware(AdminAuthMiddleware(http.HandlerFunc(webpages.HandleManageScoring))))
	http.Handle("/admin/injects", AuthMiddleware(AdminAuthMiddleware(http.HandlerFunc(webpages.HandleManageInjects))))
	http.Handle("/admin/competitions", AuthMiddleware(AdminAuthMiddleware(http.HandlerFunc(webpages.HandleCompetitionSettings))))

	// everything that starts with /api/admin send it to the admin api handlers
	http.Handle("/api/admin/services", AuthMiddleware(AdminAuthMiddleware(http.HandlerFunc(webpages.HandleApiServices))))
	http.Handle("/api/admin/teams", AuthMiddleware(AdminAuthMiddleware(http.HandlerFunc(webpages.HandleApiTeams))))
	http.Handle("/api/admin/boxes", AuthMiddleware(AdminAuthMiddleware(http.HandlerFunc(webpages.HandleApiBoxes))))
	http.Handle("/api/admin/users", AuthMiddleware(AdminAuthMiddleware(http.HandlerFunc(webpages.HandleApiUsers))))
	http.Handle("/api/admin/competition", AuthMiddleware(AdminAuthMiddleware(http.HandlerFunc(webpages.HandleApiCompetition))))
	http.Handle("/api/admin/service-matrix", AuthMiddleware(AdminAuthMiddleware(http.HandlerFunc(webpages.HandleApiServiceMatrix))))

	http.Handle("/api/admin/team-members", AuthMiddleware(AdminAuthMiddleware(http.HandlerFunc(webpages.HandleTeamMembers))))

	// Serve static files (e.g., CSS, JS)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("Server started at http://localhost:8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}

func handleLoginUser(w http.ResponseWriter, r *http.Request) {
	// set redirect cookie to /
	http.SetCookie(w, &http.Cookie{
		Name:     "redirect_to",
		Value:    "/",
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // set false for localhost dev
		MaxAge:   300,   // 5 minutes
	})
	http.Redirect(w, r, "/login", http.StatusFound)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {

	state, err := generateState()
	if err != nil {
		http.Error(w, "Failed to generate state: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Save the state in the in-memory store
	states[state] = true

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // set false for localhost dev
		MaxAge:   300,   // 5 minutes
	})

	http.Redirect(w, r, oauth2Config.AuthCodeURL(state), http.StatusFound)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Verify state
	cookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "State cookie not found", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("state") != cookie.Value {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Check if state exists in the in-memory store and is valid
	if _, exists := states[cookie.Value]; !exists {
		http.Error(w, "Invalid or expired state parameter", http.StatusBadRequest)
		return
	}
	// Remove the state from the store to prevent reuse
	delete(states, cookie.Value)

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // set false for localhost dev
		MaxAge:   -1,
	})

	// Parse the authorization code and exchange it for a token
	code := r.URL.Query().Get("code")
	oauth2Token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Extract ID Token from OAuth2 token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No id_token field in oauth2 token", http.StatusInternalServerError)
		return
	}

	// Parse and verify ID Token
	verifier := provider.Verifier(oidcConfig)
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		http.Error(w, "Failed to verify ID token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Extract claims (e.g. email, name)
	var claims struct {
		Email         string   `json:"email"`
		EmailVerified bool     `json:"email_verified"`
		Name          string   `json:"name"`
		Groups        []string `json:"groups"`
	}

	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "Failed to parse claims: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set ID Token in a secure cookie (for demo purposes, not HttpOnly)
	http.SetCookie(w, &http.Cookie{
		Name:     "id_token",
		Value:    rawIDToken,
		Path:     "/",
		HttpOnly: false,
		Secure:   false, // set false for localhost dev
		MaxAge:   3600,  // 1 hour
	})

	// Check if the user is an admin we need to look at json claim "groups" and see if it contains adminGroup
	isAdmin := false
	if adminGroup != "" {
		for _, group := range claims.Groups {
			if group == adminGroup {
				isAdmin = true
				break
			}
		}
	}

	// For demonstration, print user info to server log
	log.Printf("User Info: Email=%s, Name=%s, IsAdmin=%v\n", claims.Email, claims.Name, isAdmin)

	// Update user in the database
	user := &structures.User{
		Email:    claims.Email,
		Name:     claims.Name,
		Subject:  idToken.Subject,
		Is_Admin: isAdmin,
	}
	if err := sql_wrapper.UpdateUser(user); err != nil {
		http.Error(w, "Failed to update user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect to the originally requested page via the redirect_to cookie, if they dont have it go to /dashboard
	redirectTo := "/"
	if redirectCookie, err := r.Cookie("redirect_to"); err == nil {
		if redirectCookie.Value != "" {
			redirectTo = redirectCookie.Value
		}
		// Clear the redirect_to cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "redirect_to",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   false, // set false for localhost dev
			MaxAge:   -1,    // delete cookie
		})
	}

	http.Redirect(w, r, redirectTo, http.StatusFound)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(webpages.CtxUserKey).(structures.User)
	fmt.Fprintf(w, "Welcome, %s!", user.Name)
}

func handleScoreboard(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "This is the public scoreboard.")
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set a cookie for redirect after login
		http.SetCookie(w, &http.Cookie{
			Name:     "redirect_to",
			Value:    r.URL.Path,
			Path:     "/",
			HttpOnly: true,
			Secure:   false, // set false for localhost dev
			MaxAge:   3600,  // 1 hour
		})

		cookie, err := r.Cookie("id_token")
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Verify token
		ctx := r.Context()
		verifier := provider.Verifier(oidcConfig)

		idToken, err := verifier.Verify(ctx, cookie.Value)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Check if the user is an admin we need to look at json claim "groups" and see if it contains adminGroup
		isAdmin := false
		if adminGroup != "" {
			var claims struct {
				Groups []string `json:"groups"`
			}
			if err := idToken.Claims(&claims); err == nil {
				for _, group := range claims.Groups {
					if group == adminGroup {
						isAdmin = true
						break
					}
				}
			}
		}

		// Extract claims from idToken
		var claims struct {
			Email    string `json:"email"`
			Name     string `json:"name"`
			Subject  string `json:"sub"`
			Is_Admin bool   `json:"is_admin"`
		}

		if err := idToken.Claims(&claims); err != nil {
			http.Error(w, "Failed to parse claims", http.StatusInternalServerError)
			return
		}

		// Add user info to context
		ctx = context.WithValue(r.Context(), webpages.CtxUserKey, structures.User{
			Email:    claims.Email,
			Name:     claims.Name,
			Subject:  claims.Subject,
			Is_Admin: isAdmin,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AuthPromptMiddleware behaves like AuthMiddleware but instead of redirecting to /login
// when there is no or an invalid token, it serves a friendly login prompt page so
// the user can choose to sign in or return home. This is intended for user-facing
// pages where forcing an OIDC redirect is undesirable.
func AuthPromptMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("id_token")
		if err != nil || cookie.Value == "" {
			// Serve a login prompt template instead of redirecting
			w.WriteHeader(http.StatusUnauthorized)
			//set the redirect_to cookie to the current page
			http.SetCookie(w, &http.Cookie{
				Name:     "redirect_to",
				Value:    r.URL.Path,
				Path:     "/",
				HttpOnly: true,
				Secure:   false, // set false for localhost dev
				MaxAge:   3600,  // 1 hour
			})
			http.ServeFile(w, r, "templates/login_prompt.html")
			return
		}

		ctx := r.Context()
		verifier := provider.Verifier(oidcConfig)
		idToken, err := verifier.Verify(ctx, cookie.Value)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			http.SetCookie(w, &http.Cookie{
				Name:     "redirect_to",
				Value:    r.URL.Path,
				Path:     "/",
				HttpOnly: true,
				Secure:   false, // set false for localhost dev
				MaxAge:   3600,  // 1 hour
			})
			http.ServeFile(w, r, "templates/login_prompt.html")
			return
		}

		// Extract claims (email/name/groups). We'll use them to populate the user.
		var claims struct {
			Email  string   `json:"email"`
			Name   string   `json:"name"`
			Groups []string `json:"groups"`
			Sub    string   `json:"sub"`
		}
		if err := idToken.Claims(&claims); err != nil {
			// show prompt if claims can't be parsed
			w.WriteHeader(http.StatusUnauthorized)
			http.SetCookie(w, &http.Cookie{
				Name:     "redirect_to",
				Value:    r.URL.Path,
				Path:     "/",
				HttpOnly: true,
				Secure:   false, // set false for localhost dev
				MaxAge:   3600,  // 1 hour
			})
			http.ServeFile(w, r, "templates/login_prompt.html")
			return
		}

		isAdmin := false
		if adminGroup != "" {
			for _, g := range claims.Groups {
				if g == adminGroup {
					isAdmin = true
					break
				}
			}
		}

		user := structures.User{
			Email:    claims.Email,
			Name:     claims.Name,
			Subject:  claims.Sub,
			Is_Admin: isAdmin,
		}
		ctx = context.WithValue(ctx, webpages.CtxUserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AdminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(webpages.CtxUserKey).(structures.User)
		// Check if the user is an admin
		if !user.Is_Admin {
			http.Error(w, "Forbidden: Admins only", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
