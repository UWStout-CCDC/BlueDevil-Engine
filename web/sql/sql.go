package sql_wrapper

// Wrapper containing SQL queries used by the application. This is used to configure either sqlite or mysql.

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"

	structures "BlueDevil-Engine/structures"
)

var db *sql.DB

func InitDB(driver, dataSource string) error {
	var err error
	db, err = sql.Open(driver, dataSource)
	if err != nil {
		return err
	}
	return db.Ping()
}

func CloseDB() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

func CreateTables() error {
	userTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		name TEXT,
		subject TEXT NOT NULL UNIQUE
	);`

	servicesTable := `
	CREATE TABLE IF NOT EXISTS services (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		description TEXT
	);`

	serviceCheckTable := `
	CREATE TABLE IF NOT EXISTS service_checks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		service_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		command TEXT NOT NULL,
		FOREIGN KEY(service_id) REFERENCES services(id)
	);`

	regexCheckTable := `
	CREATE TABLE IF NOT EXISTS regex_checks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		service_check_id INTEGER NOT NULL,
		regex TEXT NOT NULL,
		expected BOOLEAN NOT NULL,
		FOREIGN KEY(service_check_id) REFERENCES service_checks(id)
	);`

	teamTable := `
	CREATE TABLE IF NOT EXISTS teams (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE
	);`

	teamMembersTable := `
	CREATE TABLE IF NOT EXISTS team_members (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		team_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		FOREIGN KEY(team_id) REFERENCES teams(id),
		FOREIGN KEY(user_id) REFERENCES users(id),
		UNIQUE(team_id, user_id)
	);`

	scoredBoxTable := `
	CREATE TABLE IF NOT EXISTS scored_boxes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		team_id INTEGER NOT NULL,
		ip_address TEXT NOT NULL,
		service_id INTEGER,
		FOREIGN KEY(team_id) REFERENCES teams(id),
		FOREIGN KEY(service_id) REFERENCES services(id)
	);`

	// box_mappings and team_mappings removed: boxes now store service_id directly

	individualPractice := `
	CREATE TABLE IF NOT EXISTS individual_scores (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		service_id INTEGER NOT NULL,
		scored_box_id INTEGER NOT NULL,
		is_up BOOLEAN NOT NULL,
		errors TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(user_id) REFERENCES users(id),
		FOREIGN KEY(service_id) REFERENCES services(id),
		FOREIGN KEY(scored_box_id) REFERENCES scored_boxes(id)
	);`

	compServiceTable := `
	CREATE TABLE IF NOT EXISTS competition_services (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		team_id INTEGER NOT NULL,
		service_id INTEGER NOT NULL,
		is_up BOOLEAN NOT NULL,
		output TEXT,
		round INTEGER NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(team_id) REFERENCES teams(id),
		FOREIGN KEY(service_id) REFERENCES services(id)
	);`

	compScoresTable := `
	CREATE TABLE IF NOT EXISTS competition_scores (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		team_id INTEGER NOT NULL,
		score INTEGER NOT NULL,
		round INTEGER,
		description TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(team_id) REFERENCES teams(id)
	);`

	competitionTable := `
	CREATE TABLE IF NOT EXISTS competition (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		status TEXT NOT NULL DEFAULT 'stopped',
		scheduled_time DATETIME,
		started_time DATETIME,
		stopped_time DATETIME
	);`

	_, err := db.Exec(servicesTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(teamTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(teamMembersTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(scoredBoxTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(individualPractice)
	if err != nil {
		return err
	}

	_, err = db.Exec(userTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(compServiceTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(serviceCheckTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(regexCheckTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(competitionTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(compScoresTable)
	if err != nil {
		return err
	}

	// injects table
	injectsTable := `
	CREATE TABLE IF NOT EXISTS injects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		inject_id TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL,
		description TEXT,
		filename TEXT,
		release_time DATETIME,
		due_time DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	injectSubTable := `
	CREATE TABLE IF NOT EXISTS inject_submissions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		inject_id TEXT NOT NULL,
		team_id INTEGER NOT NULL,
		filename TEXT NOT NULL,
		submitted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		scored BOOLEAN DEFAULT 0,
		score INTEGER,
		reviewer TEXT,
		notes TEXT,
		FOREIGN KEY(team_id) REFERENCES teams(id)
	);`

	_, err = db.Exec(injectsTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(injectSubTable)
	if err != nil {
		return err
	}

	// no separate mapping tables to create
	return nil
}

// Teams and scoring boxes helpers
func GetAllTeams() ([]structures.Team, error) {
	rows, err := db.Query("SELECT id, name FROM teams ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var teams []structures.Team
	for rows.Next() {
		var t structures.Team
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, nil
}

func CreateTeam(t *structures.Team) error {
	if t == nil {
		return nil
	}
	if t.ID == 0 {
		res, err := db.Exec("INSERT INTO teams (name) VALUES (?)", t.Name)
		if err != nil {
			return err
		}
		last, err := res.LastInsertId()
		if err == nil {
			t.ID = int(last)
		}
		return nil
	}
	_, err := db.Exec("UPDATE teams SET name = ? WHERE id = ?", t.Name, t.ID)
	return err
}

func DeleteTeam(id int) error {
	// First, remove all team members
	_, err := db.Exec("DELETE FROM team_members WHERE team_id = ?", id)
	if err != nil {
		return err
	}
	// Then delete the team
	_, err = db.Exec("DELETE FROM teams WHERE id = ?", id)
	return err
}

// Team members (map users to teams)
func AddUserToTeam(teamID, userID int) error {
	_, err := db.Exec("INSERT OR IGNORE INTO team_members (team_id, user_id) VALUES (?, ?)", teamID, userID)
	return err
}

func RemoveUserFromTeam(teamID, userID int) error {
	_, err := db.Exec("DELETE FROM team_members WHERE team_id = ? AND user_id = ?", teamID, userID)
	return err
}

func RemoveUserFromAllTeams(userID int) error {
	_, err := db.Exec("DELETE FROM team_members WHERE user_id = ?", userID)
	return err
}

func GetUsersInTeam(teamID int) ([]structures.User, error) {
	rows, err := db.Query("SELECT u.id, u.email, u.name, u.subject FROM users u JOIN team_members tm ON tm.user_id = u.id WHERE tm.team_id = ?", teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []structures.User
	for rows.Next() {
		var u structures.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Subject); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func GetAllUsers() ([]structures.User, error) {
	rows, err := db.Query("SELECT id, email, name, subject FROM users ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []structures.User
	for rows.Next() {
		var u structures.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Subject); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func GetAllUsersWithTeams() ([]structures.User, error) {
	query := `
		SELECT u.id, u.email, u.name, u.subject, tm.team_id, t.name as team_name
		FROM users u
		LEFT JOIN team_members tm ON u.id = tm.user_id
		LEFT JOIN teams t ON tm.team_id = t.id
		ORDER BY u.id ASC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []structures.User
	for rows.Next() {
		var u structures.User
		var teamID sql.NullInt64
		var teamName sql.NullString
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Subject, &teamID, &teamName); err != nil {
			return nil, err
		}
		if teamID.Valid {
			tid := int(teamID.Int64)
			u.TeamID = &tid
		}
		if teamName.Valid {
			u.TeamName = teamName.String
		}
		users = append(users, u)
	}
	return users, nil
}

func GetAllScoringBoxes() ([]structures.ScoringBox, error) {
	rows, err := db.Query("SELECT id, ip_address, team_id, service_id FROM scored_boxes ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var boxes []structures.ScoringBox
	for rows.Next() {
		var b structures.ScoringBox
		if err := rows.Scan(&b.ID, &b.IPAddress, &b.TeamID, &b.ServiceID); err != nil {
			return nil, err
		}
		boxes = append(boxes, b)
	}
	return boxes, nil
}

// Inject helpers
func CreateInject(in *structures.Inject) error {
	if in == nil {
		return nil
	}
	if in.ID == 0 {
		// Try insert; if the inject_id already exists, perform an update instead.
		res, err := db.Exec("INSERT OR IGNORE INTO injects (inject_id, title, description, filename, release_time, due_time) VALUES (?, ?, ?, ?, ?, ?)", in.InjectID, in.Title, in.Description, in.Filename, in.ReleaseTime, in.DueTime)
		if err != nil {
			return err
		}
		ra, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if ra == 0 {
			// row existed; perform update by inject_id
			_, err := db.Exec("UPDATE injects SET title = ?, description = ?, filename = ?, release_time = ?, due_time = ? WHERE inject_id = ?", in.Title, in.Description, in.Filename, in.ReleaseTime, in.DueTime, in.InjectID)
			if err != nil {
				return err
			}
			// fetch id
			row := db.QueryRow("SELECT id FROM injects WHERE inject_id = ?", in.InjectID)
			var id int
			if err := row.Scan(&id); err == nil {
				in.ID = id
			}
			return nil
		}
		last, err := res.LastInsertId()
		if err == nil {
			in.ID = int(last)
		}
		return nil
	}
	_, err := db.Exec("UPDATE injects SET inject_id = ?, title = ?, description = ?, filename = ?, release_time = ?, due_time = ? WHERE id = ?", in.InjectID, in.Title, in.Description, in.Filename, in.ReleaseTime, in.DueTime, in.ID)
	return err
}

func GetAllInjects() ([]structures.Inject, error) {
	rows, err := db.Query("SELECT id, inject_id, title, description, filename, release_time, due_time, created_at FROM injects ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []structures.Inject
	for rows.Next() {
		var i structures.Inject
		if err := rows.Scan(&i.ID, &i.InjectID, &i.Title, &i.Description, &i.Filename, &i.ReleaseTime, &i.DueTime, &i.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, nil
}

func GetInjectByID(injectID string) (*structures.Inject, error) {
	row := db.QueryRow("SELECT id, inject_id, title, description, filename, release_time, due_time, created_at FROM injects WHERE inject_id = ?", injectID)
	var i structures.Inject
	if err := row.Scan(&i.ID, &i.InjectID, &i.Title, &i.Description, &i.Filename, &i.ReleaseTime, &i.DueTime, &i.CreatedAt); err != nil {
		return nil, err
	}
	return &i, nil
}

// Submissions
func AddInjectSubmission(sub *structures.InjectSubmission) error {
	if sub == nil {
		return nil
	}
	res, err := db.Exec("INSERT INTO inject_submissions (inject_id, team_id, filename, scored, score, reviewer, notes) VALUES (?, ?, ?, ?, ?, ?, ?)", sub.InjectID, sub.TeamID, sub.Filename, sub.Scored, sub.Score, sub.Reviewer, sub.Notes)
	if err != nil {
		return err
	}
	last, err := res.LastInsertId()
	if err == nil {
		sub.ID = int(last)
	}
	return nil
}

func GetSubmissionsForInject(injectID string) ([]structures.InjectSubmission, error) {
	rows, err := db.Query("SELECT id, inject_id, team_id, filename, submitted_at, scored, score, reviewer, notes FROM inject_submissions WHERE inject_id = ? ORDER BY submitted_at DESC", injectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []structures.InjectSubmission
	for rows.Next() {
		var s structures.InjectSubmission
		if err := rows.Scan(&s.ID, &s.InjectID, &s.TeamID, &s.Filename, &s.SubmittedAt, &s.Scored, &s.Score, &s.Reviewer, &s.Notes); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func UpdateInjectSubmission(sub *structures.InjectSubmission) error {
	if sub == nil || sub.ID == 0 {
		return fmt.Errorf("invalid submission")
	}
	_, err := db.Exec("UPDATE inject_submissions SET scored = ?, score = ?, reviewer = ?, notes = ? WHERE id = ?", sub.Scored, sub.Score, sub.Reviewer, sub.Notes, sub.ID)
	return err
}

// DeleteInjectByInjectID deletes an inject and its associated submissions by inject_id
func DeleteInjectByInjectID(injectID string) error {
	if injectID == "" {
		return nil
	}
	// delete submissions first
	if _, err := db.Exec("DELETE FROM inject_submissions WHERE inject_id = ?", injectID); err != nil {
		return err
	}
	// delete inject record
	if _, err := db.Exec("DELETE FROM injects WHERE inject_id = ?", injectID); err != nil {
		return err
	}
	return nil
}

func SaveScoringBox(b *structures.ScoringBox) error {
	if b == nil {
		return nil
	}
	if b.ID == 0 {
		res, err := db.Exec("INSERT INTO scored_boxes (team_id, ip_address, service_id) VALUES (?, ?, ?)", b.TeamID, b.IPAddress, b.ServiceID)
		if err != nil {
			return err
		}
		last, err := res.LastInsertId()
		if err == nil {
			b.ID = int(last)
		}
		return nil
	}
	_, err := db.Exec("UPDATE scored_boxes SET team_id = ?, ip_address = ?, service_id = ? WHERE id = ?", b.TeamID, b.IPAddress, b.ServiceID, b.ID)
	return err
}

func DeleteScoringBox(id int) error {
	_, err := db.Exec("DELETE FROM scored_boxes WHERE id = ?", id)
	return err
}

func GetUserByEmail(email string) (*structures.User, error) {
	row := db.QueryRow("SELECT email, name, subject FROM users WHERE email = ?", email)
	var user structures.User
	err := row.Scan(&user.Email, &user.Name, &user.Subject)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func UpdateUser(user *structures.User) error {
	// Try to update; if no rows were affected, insert a new user.
	res, err := db.Exec("UPDATE users SET name = ?, subject = ? WHERE email = ?", user.Name, user.Subject, user.Email)
	if err != nil {
		return err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if ra == 0 {
		_, err = db.Exec("INSERT INTO users (email, name, subject) VALUES (?, ?, ?)", user.Email, user.Name, user.Subject)
	}
	return err
}

// In order to get the services, we need to get the checks and regexes as well.
func GetAllServices() ([]structures.Service, error) {
	services := []structures.Service{}

	serviceRows, err := db.Query("SELECT id, name, description FROM services")
	if err != nil {
		return nil, err
	}
	defer serviceRows.Close()

	for serviceRows.Next() {
		var svc structures.Service
		err := serviceRows.Scan(&svc.ID, &svc.Name, &svc.Host)
		if err != nil {
			return nil, err
		}

		checkRows, err := db.Query("SELECT id, name, command FROM service_checks WHERE service_id = ?", svc.ID)
		if err != nil {
			return nil, err
		}

		for checkRows.Next() {
			var chk structures.Checks
			// service_checks.id is an integer
			var checkID int
			err := checkRows.Scan(&checkID, &chk.Name, &chk.Command)
			if err != nil {
				checkRows.Close()
				return nil, err
			}
			chk.ID = checkID

			regexRows, err := db.Query("SELECT id, regex, expected FROM regex_checks WHERE service_check_id = ?", checkID)
			if err != nil {
				checkRows.Close()
				return nil, err
			}

			for regexRows.Next() {
				var rgx structures.Regexes
				var regexID int
				err := regexRows.Scan(&regexID, &rgx.Pattern, &rgx.Description)
				if err != nil {
					regexRows.Close()
					checkRows.Close()
					return nil, err
				}
				rgx.ID = regexID
				chk.Regexes = append(chk.Regexes, rgx)
			}
			regexRows.Close()
			svc.Checks = append(svc.Checks, chk)
		}
		checkRows.Close()
		services = append(services, svc)
	}

	if err := serviceRows.Err(); err != nil {
		return nil, err
	}

	return services, nil
}

// SaveServiceHandler saves a service (add or update).
func SaveService(svc *structures.Service) error {
	// If ID is 0, it's a new service; otherwise update existing.
	if svc.ID == 0 {
		res, err := db.Exec("INSERT INTO services (name, description) VALUES (?, ?)", svc.Name, svc.Host)
		if err != nil {
			return err
		}
		lastID, err := res.LastInsertId()
		if err != nil {
			return err
		}
		svc.ID = int(lastID)
	} else {
		_, err := db.Exec("UPDATE services SET name = ?, description = ? WHERE id = ?", svc.Name, svc.Host, svc.ID)
		if err != nil {
			return err
		}
	}

	// For simplicity, delete all existing checks and regexes and re-insert.
	_, err := db.Exec("DELETE FROM service_checks WHERE service_id = ?", svc.ID)
	if err != nil {
		return err
	}

	for _, chk := range svc.Checks {
		res, err := db.Exec("INSERT INTO service_checks (service_id, name, command) VALUES (?, ?, ?)", svc.ID, chk.Name, chk.Command)
		if err != nil {
			return err
		}
		checkID, err := res.LastInsertId()
		if err != nil {
			return err
		}

		for _, rgx := range chk.Regexes {
			_, err := db.Exec("INSERT INTO regex_checks (service_check_id, regex, expected) VALUES (?, ?, ?)", checkID, rgx.Pattern, rgx.Description)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func DeleteServiceByID(id int) error {
	// Delete regexes, checks, and then the service itself.
	_, err := db.Exec("DELETE FROM regex_checks WHERE service_check_id IN (SELECT id FROM service_checks WHERE service_id = ?)", id)
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM service_checks WHERE service_id = ?", id)
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM services WHERE id = ?", id)
	return err
}

// =========================
// Homepage data helpers
// =========================

// LatestStatus represents the latest up/down status for a team/service
type LatestStatus struct {
	TeamID    int
	ServiceID int
	IsUp      bool
}

// ServiceUptime represents uptime percentage for a service
type ServiceUptime struct {
	ServiceID int
	UptimePct float64
}

// TeamStanding represents total points for a team
type TeamStanding struct {
	TeamID int
	Name   string
	Points int
}

// RoundScore represents points per team per round
type RoundScore struct {
	Round  int
	TeamID int
	Points int
}

// GetLatestStatuses returns the latest status per team/service based on max round
func GetLatestStatuses() ([]LatestStatus, error) {
	// Join with subquery to get latest round per team/service
	q := `
		SELECT cs.team_id, cs.service_id, cs.is_up
		FROM competition_services cs
		JOIN (
			SELECT team_id, service_id, MAX(round) AS mr
			FROM competition_services
			GROUP BY team_id, service_id
		) t
		ON cs.team_id = t.team_id AND cs.service_id = t.service_id AND cs.round = t.mr
	`
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LatestStatus
	for rows.Next() {
		var ls LatestStatus
		var isUpBool bool
		if err := rows.Scan(&ls.TeamID, &ls.ServiceID, &isUpBool); err != nil {
			return nil, err
		}
		ls.IsUp = isUpBool
		out = append(out, ls)
	}
	return out, nil
}

// GetServiceUptimePercents returns uptime percentage for each service across all teams/rounds
func GetServiceUptimePercents() (map[int]float64, error) {
	q := `
		SELECT service_id,
			   SUM(CASE WHEN is_up THEN 1 ELSE 0 END) AS up_count,
			   COUNT(*) AS total_count
		FROM competition_services
		GROUP BY service_id
	`
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[int]float64)
	for rows.Next() {
		var sid int
		var upCount, total int
		if err := rows.Scan(&sid, &upCount, &total); err != nil {
			return nil, err
		}
		if total > 0 {
			m[sid] = float64(upCount) * 100.0 / float64(total)
		} else {
			m[sid] = 0.0
		}
	}
	return m, nil
}

// GetTeamStandings returns total points per team (1 point per up per record)
func GetTeamStandings() ([]TeamStanding, error) {
	q := `
		SELECT t.id, t.name, COALESCE(SUM(cs.score), 0) AS points
		FROM teams t
		LEFT JOIN competition_scores cs ON cs.team_id = t.id
		GROUP BY t.id, t.name
		ORDER BY points DESC, t.name ASC
	`
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TeamStanding
	for rows.Next() {
		var ts TeamStanding
		if err := rows.Scan(&ts.TeamID, &ts.Name, &ts.Points); err != nil {
			return nil, err
		}
		out = append(out, ts)
	}
	return out, nil
}

// GetTeamScoresByRound returns points per team per round
func GetTeamScoresByRound() ([]RoundScore, error) {
	q := `
		SELECT round, team_id, SUM(score) AS points
		FROM competition_scores
		GROUP BY round, team_id
		ORDER BY round ASC, team_id ASC
	`
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RoundScore
	for rows.Next() {
		var r RoundScore
		if err := rows.Scan(&r.Round, &r.TeamID, &r.Points); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

// Competition management functions

// GetCompetition returns the current competition state (there should only be one)
func GetCompetition() (*structures.Competition, error) {
	row := db.QueryRow("SELECT id, status, scheduled_time, started_time, stopped_time FROM competition ORDER BY id DESC LIMIT 1")
	var comp structures.Competition
	var scheduledTime, startedTime, stoppedTime sql.NullString
	err := row.Scan(&comp.ID, &comp.Status, &scheduledTime, &startedTime, &stoppedTime)
	if err == sql.ErrNoRows {
		// No competition exists, create a default one
		_, err = db.Exec("INSERT INTO competition (status) VALUES ('stopped')")
		if err != nil {
			return nil, err
		}
		// Fetch the newly created competition
		return GetCompetition()
	}
	if err != nil {
		return nil, err
	}
	comp.ScheduledTime = scheduledTime.String
	comp.StartedTime = startedTime.String
	comp.StoppedTime = stoppedTime.String
	return &comp, nil
}

// UpdateCompetition updates the competition state
func UpdateCompetition(comp *structures.Competition) error {
	if comp == nil {
		return nil
	}

	// Get or create the competition
	existing, err := GetCompetition()
	if err != nil {
		return err
	}

	// Update the existing competition
	query := "UPDATE competition SET status = ?, scheduled_time = ?, started_time = ?, stopped_time = ? WHERE id = ?"
	var scheduledTime, startedTime, stoppedTime interface{}

	if comp.ScheduledTime != "" {
		scheduledTime = comp.ScheduledTime
	}
	if comp.StartedTime != "" {
		startedTime = comp.StartedTime
	}
	if comp.StoppedTime != "" {
		stoppedTime = comp.StoppedTime
	}

	_, err = db.Exec(query, comp.Status, scheduledTime, startedTime, stoppedTime, existing.ID)
	return err
}

// ResetCompetitionServices deletes all competition services
func ResetCompetitionServices() error {
	if _, err := db.Exec("DELETE FROM competition_services"); err != nil {
		return err
	}

	if _, err := db.Exec("DELETE FROM competition_scores"); err != nil {
		return err
	}

	return nil
}

// ResetAllScoringData clears all competition-related data: scores, service status records,
// individual practice scores, and service checks (including regex checks). It also
// resets the competition status to 'stopped'. Use with care.
func ResetAllScoringData() error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	// Clear competition scores and service status history
	if _, err = tx.Exec("DELETE FROM competition_scores"); err != nil {
		return err
	}
	if _, err = tx.Exec("DELETE FROM competition_services"); err != nil {
		return err
	}

	// Reset competition metadata to stopped and clear times
	if _, err = tx.Exec("UPDATE competition SET status = 'stopped', scheduled_time = NULL, started_time = NULL, stopped_time = NULL"); err != nil {
		return err
	}

	return nil
}

func GetTeamScore(teamID int) (int, error) {
	row := db.QueryRow("SELECT SUM(score) FROM competition_scores WHERE team_id = ?", teamID)
	var score sql.NullInt64
	err := row.Scan(&score)
	if err != nil {
		return 0, err
	}
	if score.Valid {
		return int(score.Int64), nil
	}
	return 0, nil
}

// GetTeamServiceUptimePercents returns uptime percentage for each team/service
func GetTeamServiceUptimePercents() (map[int]map[int]float64, error) {
	q := `
		SELECT team_id, service_id,
			   SUM(CASE WHEN is_up THEN 1 ELSE 0 END) AS up_count,
			   COUNT(*) AS total_count
		FROM competition_services
		GROUP BY team_id, service_id
	`
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[int]map[int]float64)
	for rows.Next() {
		var teamID, serviceID, upCount, total int
		if err := rows.Scan(&teamID, &serviceID, &upCount, &total); err != nil {
			return nil, err
		}
		if _, ok := m[teamID]; !ok {
			m[teamID] = make(map[int]float64)
		}
		if total > 0 {
			m[teamID][serviceID] = float64(upCount) * 100.0 / float64(total)
		} else {
			m[teamID][serviceID] = 0.0
		}
	}
	return m, nil
}

// Competition service history record returned to UI
type CompetitionServiceRecord struct {
	TeamID    int    `json:"team_id"`
	ServiceID int    `json:"service_id"`
	IsUp      bool   `json:"is_up"`
	Output    string `json:"output"`
	Round     int    `json:"round"`
	Timestamp string `json:"timestamp"`
}

// GetCompetitionServiceHistory returns competition_services rows for a given team/service
// If teamID or serviceID is 0, that filter is ignored.
func GetCompetitionServiceHistory(teamID, serviceID int) ([]CompetitionServiceRecord, error) {
	q := `SELECT team_id, service_id, is_up, output, round, timestamp FROM competition_services`
	var args []interface{}
	var where []string
	if teamID != 0 {
		where = append(where, "team_id = ?")
		args = append(args, teamID)
	}
	if serviceID != 0 {
		where = append(where, "service_id = ?")
		args = append(args, serviceID)
	}
	if len(where) > 0 {
		q = q + " WHERE " + strings.Join(where, " AND ")
	}
	q = q + " ORDER BY round ASC, timestamp ASC"

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CompetitionServiceRecord
	for rows.Next() {
		var r CompetitionServiceRecord
		var isUp bool
		if err := rows.Scan(&r.TeamID, &r.ServiceID, &isUp, &r.Output, &r.Round, &r.Timestamp); err != nil {
			return nil, err
		}
		r.IsUp = isUp
		out = append(out, r)
	}
	return out, nil
}

// AddCompetitionScoreAdjustment inserts an adjustment into competition_scores
func AddCompetitionScoreAdjustment(teamID int, score int, round int, description string) (int, error) {
	if teamID == 0 {
		return 0, fmt.Errorf("team_id required")
	}
	var res sql.Result
	var err error
	if round > 0 {
		res, err = db.Exec("INSERT INTO competition_scores (team_id, score, round, description) VALUES (?, ?, ?, ?)", teamID, score, round, description)
	} else {
		res, err = db.Exec("INSERT INTO competition_scores (team_id, score, description) VALUES (?, ?, ?)", teamID, score, description)
	}
	if err != nil {
		return 0, err
	}
	last, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(last), nil
}

// GetUserTeamBySubject returns the team (if any) that the user identified by the
// given subject belongs to. If the user is not a member of any team, (nil, nil)
// is returned.
func GetUserTeamBySubject(subject string) (*structures.Team, error) {
	row := db.QueryRow(`
		SELECT t.id, t.name
		FROM teams t
		JOIN team_members tm ON t.id = tm.team_id
		JOIN users u ON u.id = tm.user_id
		WHERE u.subject = ?
		LIMIT 1
	`, subject)
	var t structures.Team
	if err := row.Scan(&t.ID, &t.Name); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}
