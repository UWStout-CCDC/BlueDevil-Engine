package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

type ServiceIP struct {
	Service    string `json:"service"`
	InternalIP string `json:"internal_ip"`
	// NatPrefix is the leading parts of the NAT IP, e.g. "10.10"
	NatPrefix string `json:"nat_prefix,omitempty"`
	// NatBase is the integer base that's added to the team ID, e.g. 39
	NatBase int `json:"nat_base,omitempty"`
	// NatSuffix is the final octet (e.g. 9) used after prefix and base+team
	NatSuffix int `json:"nat_suffix,omitempty"`
	// NatTemplate is an optional template string that will be executed with
	// variable 'team' available. Example: "10.10.{{ add 39 team }}.9"
	NatTemplate string `json:"nat_template,omitempty" yaml:"nat_template,omitempty"`
}

type LoginEntry struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Notes    string `json:"notes,omitempty"`
}

type DefaultPassword struct {
	Box    string       `json:"box"`
	OS     string       `json:"os"`
	IP     string       `json:"ip"`
	Group  string       `json:"group,omitempty"`
	Logins []LoginEntry `json:"logins"`
}

type EnvLoginTemplate struct {
	Service          string `json:"service"`
	URLTemplate      string `json:"url_template"`
	UsernameTemplate string `json:"username_template"`
	PasswordTemplate string `json:"password_template"`
}

type Config struct {
	ServiceIPScheme   []ServiceIP        `json:"service_ip_scheme"`
	DefaultPasswords  []DefaultPassword  `json:"default_passwords"`
	EnvLoginTemplates []EnvLoginTemplate `json:"env_logins_templates"`
}

var Global Config

func GetConfig() Config {
	return Global
}

func init() {
	// Try a few likely locations for envinfo.json: same dir, parent, or repo root
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Println("config: unable to determine caller file path")
		return
	}
	dir := filepath.Dir(filename)

	candidates := []string{
		filepath.Join(dir, "envinfo.json"),
		filepath.Join(dir, "..", "envinfo.json"),
		filepath.Join(dir, "..", "..", "envinfo.json"),
		filepath.Join("..", "envinfo.json"),
		"envinfo.json",
	}

	var f *os.File
	var err error
	for _, p := range candidates {
		if fp, e := os.Open(p); e == nil {
			f = fp
			err = nil
			break
		} else {
			err = e
		}
	}

	if err != nil {
		// missing config is not fatal; leave defaults empty
		log.Println("config: could not open envinfo.json in candidates, using defaults:", err)
		return
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	if err := dec.Decode(&Global); err != nil {
		log.Println("config: failed to parse envinfo.json:", err)
		return
	}
}
