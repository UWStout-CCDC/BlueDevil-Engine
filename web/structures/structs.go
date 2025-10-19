package structures

type User struct {
	ID       int    `json:"id,omitempty"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Subject  string `json:"sub"`
	Is_Admin bool   `json:"is_admin"`
	TeamID   *int   `json:"team_id,omitempty"`
	TeamName string `json:"team_name,omitempty"`
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

// Competition represents the current competition state
type Competition struct {
	ID            int    `json:"id"`
	Status        string `json:"status"` // "stopped", "scheduled", "running"
	ScheduledTime string `json:"scheduled_time,omitempty"`
	StartedTime   string `json:"started_time,omitempty"`
	StoppedTime   string `json:"stopped_time,omitempty"`
}

// Inject represents an inject that can be released during a competition
type Inject struct {
	ID          int    `json:"id"`
	InjectID    string `json:"inject_id"` // short unique identifier
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Filename    string `json:"filename,omitempty"` // stored PDF filename under /injects/
	ReleaseTime string `json:"release_time,omitempty"`
	DueTime     string `json:"due_time,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// InjectSubmission represents a team's submission for an inject
type InjectSubmission struct {
	ID          int    `json:"id"`
	InjectID    string `json:"inject_id"`
	TeamID      int    `json:"team_id"`
	Filename    string `json:"filename"`
	SubmittedAt string `json:"submitted_at,omitempty"`
	Scored      bool   `json:"scored,omitempty"`
	Score       *int   `json:"score,omitempty"`
	Reviewer    string `json:"reviewer,omitempty"`
	Notes       string `json:"notes,omitempty"`
}
