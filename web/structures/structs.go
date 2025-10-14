package structures

type User struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Subject  string `json:"sub"`
	Is_Admin bool   `json:"is_admin"`
}
