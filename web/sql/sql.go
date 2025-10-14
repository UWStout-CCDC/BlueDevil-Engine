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
		scoreType TEXT NOT NULL,
		description TEXT
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

	scoringBoxTable := `
	CREATE TABLE IF NOT EXISTS scoring_boxes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		service_id INTEGER NOT NULL,
		ip_address TEXT NOT NULL,
		port INTEGER NOT NULL,
		FOREIGN KEY(service_id) REFERENCES services(id)
	);`

	individualPractice := `
	CREATE TABLE IF NOT EXISTS individual_scores (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		service_id INTEGER NOT NULL,
		scoring_box_id INTEGER NOT NULL,
		is_up BOOLEAN NOT NULL,
		errors TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(user_id) REFERENCES users(id),
		FOREIGN KEY(service_id) REFERENCES services(id),
		FOREIGN KEY(scoring_box_id) REFERENCES scoring_boxes(id)
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

	_, err = db.Exec(scoringBoxTable)
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
