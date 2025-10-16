package structures

type User struct {
	ID       int    `json:"id,omitempty"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Subject  string `json:"sub"`
	Is_Admin bool   `json:"is_admin"`
}

type Service struct {
	ID     int      `json:"id"`
	Name   string   `json:"name"`
	Host   string   `json:"host,omitempty"`
	Checks []Checks `json:"checks,omitempty"`
}

type Checks struct {
	ID      int       `json:"id,omitempty"`
	Name    string    `json:"name"`
	Command string    `json:"command"`
	Regexes []Regexes `json:"regexes,omitempty"`
}

type Regexes struct {
	ID          int    `json:"id,omitempty"`
	Pattern     string `json:"pattern"`
	Description string `json:"description"`
}

type Credential struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Team represents a competition team
type Team struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ScoringBox represents a box with an IP assigned to a team and service
type ScoringBox struct {
	ID        int    `json:"id"`
	IPAddress string `json:"ip_address"`
	TeamID    int    `json:"team_id"`
	ServiceID int    `json:"service_id"`
}
