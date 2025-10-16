package webpages

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"

	dbsql "BlueDevil-Engine/sql"

	"github.com/coreos/go-oidc/v3/oidc"
)

// HomepageViewModel contains data for the public homepage
type HomepageViewModel struct {
	Services          []ServiceMeta
	Teams             []TeamMeta
	LatestStatuses    map[int]map[int]bool    // teamID -> serviceID -> isUp
	ServiceUptime     map[int]float64         // serviceID -> uptime%
	TeamServiceUptime map[int]map[int]float64 // teamID -> serviceID -> uptime%
	Standings         []TeamStandingVM
	ScoresByRound     []RoundScoreVM
	AutoRefreshSec    int
	// Navbar / session info
	IsLoggedIn bool
	IsAdmin    bool
	UserName   string
	Active     string
	HasScoring bool
}

type ServiceMeta struct {
	ID   int
	Name string
}

type TeamMeta struct {
	ID    int
	Name  string
	Color string
	Path  string // SVG path for the line chart
}

type TeamStandingVM struct {
	TeamID int
	Name   string
	Points int
	Rank   int
}

type RoundScoreVM struct {
	Round  int
	TeamID int
	Points int
}

// HandleHomepage serves the public homepage
func HandleHomepage(w http.ResponseWriter, r *http.Request) {
	// Try to identify user from id_token (optional)
	isLoggedIn, isAdmin, userName := getUserInfoFromCookie(r)
	// Load services and teams
	services, err := dbsql.GetAllServices()
	if err != nil {
		http.Error(w, "failed to load services", http.StatusInternalServerError)
		log.Println("homepage: services error:", err)
		return
	}
	teams, err := dbsql.GetAllTeams()
	if err != nil {
		http.Error(w, "failed to load teams", http.StatusInternalServerError)
		log.Println("homepage: teams error:", err)
		return
	}

	// Build meta slices
	var svcMeta []ServiceMeta
	for _, s := range services {
		svcMeta = append(svcMeta, ServiceMeta{ID: s.ID, Name: s.Name})
	}
	var teamMeta []TeamMeta
	palette := []string{"#0072B2", "#D55E00", "#009E73", "#CC79A7", "#F0E442", "#56B4E9", "#E69F00", "#000000"}
	for i, t := range teams {
		color := palette[i%len(palette)]
		teamMeta = append(teamMeta, TeamMeta{ID: t.ID, Name: t.Name, Color: color})
	}

	// Load homepage aggregates
	latest, err := dbsql.GetLatestStatuses()
	if err != nil {
		http.Error(w, "failed to load latest statuses", http.StatusInternalServerError)
		log.Println("homepage: latest statuses error:", err)
		return
	}
	uptime, err := dbsql.GetServiceUptimePercents()
	if err != nil {
		http.Error(w, "failed to load uptime", http.StatusInternalServerError)
		log.Println("homepage: uptime error:", err)
		return
	}
	teamSvcUptime, err := dbsql.GetTeamServiceUptimePercents()
	if err != nil {
		http.Error(w, "failed to load team/service uptime", http.StatusInternalServerError)
		log.Println("homepage: team/service uptime error:", err)
		return
	}
	standings, err := dbsql.GetTeamStandings()
	if err != nil {
		http.Error(w, "failed to load standings", http.StatusInternalServerError)
		log.Println("homepage: standings error:", err)
		return
	}
	roundScores, err := dbsql.GetTeamScoresByRound()
	if err != nil {
		http.Error(w, "failed to load round scores", http.StatusInternalServerError)
		log.Println("homepage: round scores error:", err)
		return
	}

	// Transform
	latestMap := make(map[int]map[int]bool)
	// initialize defaults to false for all team/service combos
	for _, t := range teamMeta {
		if _, ok := latestMap[t.ID]; !ok {
			latestMap[t.ID] = make(map[int]bool)
		}
		for _, s := range svcMeta {
			latestMap[t.ID][s.ID] = false
		}
	}
	// overlay with actual latest readings
	for _, ls := range latest {
		if _, ok := latestMap[ls.TeamID]; !ok {
			latestMap[ls.TeamID] = make(map[int]bool)
		}
		latestMap[ls.TeamID][ls.ServiceID] = ls.IsUp
	}
	// sort standings and assign rank
	sort.SliceStable(standings, func(i, j int) bool {
		if standings[i].Points == standings[j].Points {
			return standings[i].Name < standings[j].Name
		}
		return standings[i].Points > standings[j].Points
	})
	var standingsVM []TeamStandingVM
	for i, st := range standings {
		standingsVM = append(standingsVM, TeamStandingVM{TeamID: st.TeamID, Name: st.Name, Points: st.Points, Rank: i + 1})
	}
	var roundVM []RoundScoreVM
	maxRound := 0
	for _, rs := range roundScores {
		roundVM = append(roundVM, RoundScoreVM{Round: rs.Round, TeamID: rs.TeamID, Points: rs.Points})
		if rs.Round > maxRound {
			maxRound = rs.Round
		}
	}

	// Build SVG paths as cumulative points per team across rounds
	cumByTeam := make(map[int][]int)
	for _, t := range teamMeta {
		cumByTeam[t.ID] = make([]int, maxRound+1) // index by round
	}
	// gather round -> team -> points
	pointsByRound := make(map[int]map[int]int)
	for _, rs := range roundVM {
		if _, ok := pointsByRound[rs.Round]; !ok {
			pointsByRound[rs.Round] = make(map[int]int)
		}
		pointsByRound[rs.Round][rs.TeamID] = rs.Points
	}
	maxCum := 0
	for _, t := range teamMeta {
		cum := 0
		for r := 1; r <= maxRound; r++ {
			cum += pointsByRound[r][t.ID]
			cumByTeam[t.ID][r] = cum
			if cum > maxCum {
				maxCum = cum
			}
		}
	}
	if maxCum == 0 {
		maxCum = 1
	}
	left, bottom := 40.0, 210.0
	width, height := 740.0, 180.0
	toX := func(round int) float64 {
		if maxRound <= 1 {
			return left + width
		}
		return left + (float64(round-1)/float64(maxRound-1))*width
	}
	toY := func(val int) float64 {
		return bottom - (float64(val)/float64(maxCum))*height
	}
	for i := range teamMeta {
		t := &teamMeta[i]
		if maxRound == 0 {
			t.Path = ""
			continue
		}
		path := ""
		for r := 1; r <= maxRound; r++ {
			x := toX(r)
			y := toY(cumByTeam[t.ID][r])
			if r == 1 {
				path = fmt.Sprintf("M %.1f %.1f", x, y)
			} else {
				path += fmt.Sprintf(" L %.1f %.1f", x, y)
			}
		}
		t.Path = path
	}

	// active tab based on path
	active := "scoring"
	switch r.URL.Path {
	case "/info":
		active = "info"
	case "/injects":
		active = "injects"
	case "/practice":
		active = "practice"
	}

	vm := HomepageViewModel{
		Services:          svcMeta,
		Teams:             teamMeta,
		LatestStatuses:    latestMap,
		ServiceUptime:     uptime,
		TeamServiceUptime: teamSvcUptime,
		Standings:         standingsVM,
		ScoresByRound:     roundVM,
		AutoRefreshSec:    5,
		IsLoggedIn:        isLoggedIn,
		IsAdmin:           isAdmin,
		UserName:          userName,
		Active:            active,
		HasScoring:        len(latest) > 0 || len(roundVM) > 0,
	}

	// Parse and execute template
	tmpl, err := template.ParseFiles("templates/homepage.html")
	if err != nil {
		http.Error(w, "template parse error", http.StatusInternalServerError)
		log.Println("homepage: template parse error:", err)
		return
	}
	// buffer the output to avoid superfluous WriteHeader on exec errors
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vm); err != nil {
		log.Println("homepage: template exec error:", err)
		http.Error(w, "template exec error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}

// --- Minimal OIDC parsing for navbar ---
var (
	navbarOnce     sync.Once
	navbarVerifier *oidc.IDTokenVerifier
	navbarInitErr  error
)

func initNavbarVerifier() {
	issuer := os.Getenv("OIDC_ISSUER_URL")
	clientID := os.Getenv("OIDC_CLIENT_ID")
	if issuer == "" || clientID == "" {
		navbarInitErr = fmt.Errorf("OIDC_ISSUER_URL or OIDC_CLIENT_ID not set")
		return
	}
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		navbarInitErr = err
		return
	}
	cfg := &oidc.Config{ClientID: clientID}
	navbarVerifier = provider.Verifier(cfg)
}

func getUserInfoFromCookie(r *http.Request) (loggedIn bool, isAdmin bool, name string) {
	navbarOnce.Do(initNavbarVerifier)
	cookie, err := r.Cookie("id_token")
	if err != nil || cookie.Value == "" || navbarVerifier == nil || navbarInitErr != nil {
		return false, false, ""
	}
	idToken, err := navbarVerifier.Verify(r.Context(), cookie.Value)
	if err != nil {
		return false, false, ""
	}
	var claims struct {
		Email  string   `json:"email"`
		Name   string   `json:"name"`
		Groups []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return true, false, ""
	}
	adminGroup := os.Getenv("ADMIN_GROUP")
	isAdm := false
	if adminGroup != "" {
		for _, g := range claims.Groups {
			if g == adminGroup {
				isAdm = true
				break
			}
		}
	}
	display := claims.Name
	if display == "" {
		display = claims.Email
	}
	return true, isAdm, display
}
