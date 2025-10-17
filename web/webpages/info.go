package webpages

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	cfg "BlueDevil-Engine/config"
	sql_wrapper "BlueDevil-Engine/sql"
	structures "BlueDevil-Engine/structures"

	sprig "github.com/Masterminds/sprig/v3"
)

// HandleInfoPage renders the info page server-side (no client JS). It uses the
// authenticated user from the request context to determine team scoping and
// then renders `templates/info.html` with the populated data.
func HandleInfoPage(w http.ResponseWriter, r *http.Request) {
	log.Println("Serving info page")

	// Extract user from context (we expect a structures.User stored under CtxUserKey)
	var subject string
	var userName string
	var isAdmin bool
	var isLoggedIn bool
	if u := r.Context().Value(CtxUserKey); u != nil {
		switch v := u.(type) {
		case structures.User:
			subject = v.Subject
			userName = v.Name
			isAdmin = v.Is_Admin
			isLoggedIn = true
		case *structures.User:
			subject = v.Subject
			userName = v.Name
			isAdmin = v.Is_Admin
			isLoggedIn = true
		case map[string]interface{}:
			if s, ok := v["sub"].(string); ok {
				subject = s
			}
			if n, ok := v["name"].(string); ok {
				userName = n
			}
			if adm, ok := v["is_admin"].(bool); ok {
				isAdmin = adm
			}
			if subject != "" {
				isLoggedIn = true
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

	// Build page data
	// Config-driven service/env data only

	svcIP := []map[string]string{}
	// Use only config values for service IP scheme
	conf := cfg.GetConfig()
	if len(conf.ServiceIPScheme) > 0 {
		for _, si := range conf.ServiceIPScheme {
			nat := ""
			if teamID > 0 {
				// prefer NatTemplate if provided
				if si.NatTemplate != "" {
					// render template with sprig funcs and a `team` helper function
					funcs := sprig.TxtFuncMap()
					// add aliases for multiplication helper names (sprig uses "mul")
					if m, ok := funcs["mul"]; ok {
						funcs["multiply"] = m
						funcs["times"] = m
						funcs["mult"] = m
					}
					// inject team function so templates can reference `team` as a bare identifier
					funcs["team"] = func() int { return teamID }
					funcs["Team"] = func() int { return teamID }
					t, terr := template.New("nat").Funcs(funcs).Parse(si.NatTemplate)
					if terr == nil {
						var tb bytes.Buffer
						if err := t.Execute(&tb, nil); err == nil {
							nat = template.HTMLEscapeString(tb.String())
						} else {
							log.Println("nat template execute error:", err)
						}
					} else {
						log.Println("nat template parse error:", terr)
					}
				} else if si.NatPrefix != "" {
					// calculate nat: prefix + '.' + (nat_base + team) + '.' + suffix
					octet := si.NatBase + teamID
					nat = template.HTMLEscapeString(si.NatPrefix + "." + strconv.Itoa(octet) + "." + strconv.Itoa(si.NatSuffix))
				}
			}
			svcIP = append(svcIP, map[string]string{"service": si.Service, "internal": template.HTMLEscapeString(si.InternalIP), "nat": nat})
		}
	}

	// helper to render any template string with sprig and team helper
	render := func(raw string) string {
		if raw == "" || !strings.Contains(raw, "{{") {
			return raw
		}
		funcs := sprig.TxtFuncMap()
		if m, ok := funcs["mul"]; ok {
			funcs["multiply"] = m
			funcs["times"] = m
			funcs["mult"] = m
		}
		funcs["team"] = func() int { return teamID }
		funcs["Team"] = func() int { return teamID }
		t, terr := template.New("cfg").Funcs(funcs).Parse(raw)
		if terr != nil {
			log.Println("config template parse error:", terr)
			return raw
		}
		var tb bytes.Buffer
		if err := t.Execute(&tb, nil); err != nil {
			log.Println("config template execute error:", err)
			return raw
		}
		return tb.String()
	}

	// Build grouped passwords: map[group] -> []boxes, where each box contains rendered logins
	grouped := map[string][]map[string]interface{}{}
	if len(conf.DefaultPasswords) > 0 {
		for _, dp := range conf.DefaultPasswords {
			grp := dp.Group
			if grp == "" {
				grp = "Misc"
			}

			// render box fields
			boxName := render(dp.Box)
			boxOS := render(dp.OS)
			boxIP := render(dp.IP)

			// render logins into new slice of same struct type
			renderedLogins := []cfg.LoginEntry{}
			for _, ln := range dp.Logins {
				rl := ln
				rl.Username = render(rl.Username)
				rl.Password = render(rl.Password)
				renderedLogins = append(renderedLogins, rl)
			}

			box := map[string]interface{}{
				"box":    boxName,
				"os":     boxOS,
				"ip":     boxIP,
				"logins": renderedLogins,
			}
			grouped[grp] = append(grouped[grp], box)
		}
	}

	envLogins := []map[string]string{}
	if teamID != 0 && len(conf.EnvLoginTemplates) > 0 {
		for _, t := range conf.EnvLoginTemplates {
			url := template.HTMLEscapeString(replaceTeamPlaceholder(t.URLTemplate, teamID))
			user := template.HTMLEscapeString(replaceTeamPlaceholder(t.UsernameTemplate, teamID))
			pass := template.HTMLEscapeString(replaceTeamPlaceholder(t.PasswordTemplate, teamID))
			envLogins = append(envLogins, map[string]string{"service": t.Service, "url": url, "username": user, "password": pass})
		}
	}

	data := map[string]interface{}{
		"Active":           "info",
		"IsAdmin":          isAdmin,
		"IsLoggedIn":       isLoggedIn,
		"UserName":         userName,
		"TeamID":           teamID,
		"ServiceIPScheme":  svcIP,
		"GroupedPasswords": grouped,
		"EnvLogins":        envLogins,
	}

	tmpl, err := template.New("info.html").Funcs(template.FuncMap{
		"eq": func(a, b interface{}) bool { return a == b },
	}).ParseFiles("templates/info.html")
	if err != nil {
		log.Println("Failed to parse template:", err)
		http.ServeFile(w, r, "templates/info.html")
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Println("Failed to execute template:", err)
		http.Error(w, "Template render error", http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		log.Println("Failed to write template response:", err)
	}
}

// replaceTeamPlaceholder replaces occurrences of {team} or TEAM placeholders with the numeric team id.
func replaceTeamPlaceholder(tmpl string, teamID int) string {
	res := tmpl
	res = strings.ReplaceAll(res, "{team}", strconv.Itoa(teamID))
	res = strings.ReplaceAll(res, "TEAM", strconv.Itoa(teamID))
	return res
}
