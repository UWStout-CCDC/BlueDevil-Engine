package sql_wrapper

// Wrapper containing SQL queries used by the application. This is used to configure either sqlite or mysql.

import (
	"database/sql"

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
		port INTEGER NOT NULL,
		FOREIGN KEY(team_id) REFERENCES teams(id)
	);`

	boxMappingsTable := `
	CREATE TABLE IF NOT EXISTS box_mappings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		scored_box_id INTEGER NOT NULL,
		service_id INTEGER NOT NULL,
		FOREIGN KEY(scored_box_id) REFERENCES scored_boxes(id),
		FOREIGN KEY(service_id) REFERENCES services(id)
	);`

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

	compScoreTable := `
	CREATE TABLE IF NOT EXISTS competition_scores (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		team_id INTEGER NOT NULL,
		service_id INTEGER NOT NULL,
		is_up BOOLEAN NOT NULL,
		errors TEXT,
		round INTEGER NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(team_id) REFERENCES teams(id),
		FOREIGN KEY(service_id) REFERENCES services(id)
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

	_, err = db.Exec(compScoreTable)
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

	_, err = db.Exec(boxMappingsTable)
	if err != nil {
		return err
	}
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

// Team members (map users to teams)
func AddUserToTeam(teamID, userID int) error {
	_, err := db.Exec("INSERT OR IGNORE INTO team_members (team_id, user_id) VALUES (?, ?)", teamID, userID)
	return err
}

func RemoveUserFromTeam(teamID, userID int) error {
	_, err := db.Exec("DELETE FROM team_members WHERE team_id = ? AND user_id = ?", teamID, userID)
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

func GetAllScoringBoxes() ([]structures.ScoringBox, error) {
	rows, err := db.Query("SELECT id, team_id, ip_address, port FROM scored_boxes ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var boxes []structures.ScoringBox
	for rows.Next() {
		var b structures.ScoringBox
		if err := rows.Scan(&b.ID, &b.TeamID, &b.IPAddress, &b.Port); err != nil {
			return nil, err
		}
		boxes = append(boxes, b)
	}
	return boxes, nil
}

func SaveScoringBox(b *structures.ScoringBox) error {
	if b == nil {
		return nil
	}
	if b.ID == 0 {
		res, err := db.Exec("INSERT INTO scored_boxes (team_id, ip_address, port) VALUES (?, ?, ?)", b.TeamID, b.IPAddress, b.Port)
		if err != nil {
			return err
		}
		last, err := res.LastInsertId()
		if err == nil {
			b.ID = int(last)
		}
		return nil
	}
	_, err := db.Exec("UPDATE scored_boxes SET team_id = ?, ip_address = ?, port = ? WHERE id = ?", b.TeamID, b.IPAddress, b.Port, b.ID)
	return err
}

func DeleteScoringBox(id int) error {
	_, err := db.Exec("DELETE FROM box_mappings WHERE scored_box_id = ?", id)
	if err != nil {
		return err
	}
	_, err = db.Exec("DELETE FROM scored_boxes WHERE id = ?", id)
	return err
}

func GetAllBoxMappings() ([]structures.BoxMapping, error) {
	rows, err := db.Query("SELECT id, scored_box_id, service_id FROM box_mappings ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var mappings []structures.BoxMapping
	for rows.Next() {
		var m structures.BoxMapping
		if err := rows.Scan(&m.ID, &m.ScoringBoxID, &m.ServiceID); err != nil {
			return nil, err
		}
		mappings = append(mappings, m)
	}
	return mappings, nil
}

func SaveBoxMapping(m *structures.BoxMapping) error {
	if m == nil {
		return nil
	}
	if m.ID == 0 {
		res, err := db.Exec("INSERT INTO box_mappings (scored_box_id, service_id) VALUES (?, ?)", m.ScoringBoxID, m.ServiceID)
		if err != nil {
			return err
		}
		last, err := res.LastInsertId()
		if err == nil {
			m.ID = int(last)
		}
		return nil
	}
	_, err := db.Exec("UPDATE box_mappings SET scored_box_id = ?, service_id = ? WHERE id = ?", m.ScoringBoxID, m.ServiceID, m.ID)
	return err
}

func DeleteBoxMapping(id int) error {
	_, err := db.Exec("DELETE FROM box_mappings WHERE id = ?", id)
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
