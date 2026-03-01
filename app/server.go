package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// InstanceStatus is the JSON structure returned for /status
type InstanceStatus struct {
	Name             string        `json:"name"`
	LastGenerated    time.Time     `json:"last_generated"`
	LastDuration     time.Duration `json:"last_duration"`
	Success          bool          `json:"success"`
	CurrentPID       int           `json:"current_pid,omitempty"`
	TimeUntilNextRun time.Duration `json:"time_until_next_run"`
	LastErrorSnippet string        `json:"last_error_snippet,omitempty"`
}

// NewServer creates and returns an HTTP server for the given instances
func NewServer(instances []*Instance) *http.Server {
	mux := http.NewServeMux()

	// /status endpoint
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		var statuses []InstanceStatus
		for _, inst := range instances {
			inst.mu.Lock()
			snippet := ""
			if inst.LastError != "" {
				lines := strings.Split(inst.LastError, "\n")
				if len(lines) > 20 {
					lines = lines[:20]
				}
				snippet = strings.Join(lines, "\n")
			}
			statuses = append(statuses, InstanceStatus{
				Name:             inst.Name,
				LastGenerated:    inst.LastGen,
				LastDuration:     inst.LastDur,
				Success:          inst.Success,
				CurrentPID:       inst.CurrentPID,
				TimeUntilNextRun: time.Until(inst.NextRun),
				LastErrorSnippet: snippet,
			})
			inst.mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(statuses)
	})

	if len(instances) == 1 {
		// single instance → serve the full public directory
		fs := http.FileServer(http.Dir(instances[0].PublicDir))
		mux.Handle("/", fs)
	} else {
		// multi-instance landing page
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>MinecraftStats Instances</title>
<style>
body {
	font-family: Arial, sans-serif;
	background-color: #f5f5f5;
	margin: 0;
	padding: 20px;
}
h1 {
	text-align: center;
	color: #333;
}
.container {
	display: flex;
	flex-wrap: wrap;
	justify-content: center;
	margin-top: 30px;
}
.card {
	background: white;
	border-radius: 8px;
	box-shadow: 0 2px 6px rgba(0,0,0,0.15);
	margin: 10px;
	padding: 20px;
	width: 180px;
	text-align: center;
	transition: transform 0.2s, box-shadow 0.2s;
}
.card:hover {
	transform: translateY(-5px);
	box-shadow: 0 4px 12px rgba(0,0,0,0.25);
}
.card a {
	text-decoration: none;
	color: #0078d7;
	font-weight: bold;
	font-size: 16px;
	display: block;
}
.status {
	margin-top: 10px;
	font-size: 12px;
}
.status.success { color: green; }
.status.fail { color: red; }
</style>
</head>
<body>
<h1>Select a MinecraftStats Instance</h1>
<div class="container">
`)

			for _, inst := range instances {
				inst.mu.Lock()
				statusClass := "success"
				statusText := "OK"
				if !inst.Success {
					statusClass = "fail"
					statusText = "Failed"
				}
				inst.mu.Unlock()
				fmt.Fprintf(w,
					`<div class="card">
						<a href="/%s/">%s</a>
						<div class="status %s">%s</div>
					</div>`,
					inst.Name, inst.Name, statusClass, statusText)
			}

			fmt.Fprintf(w, `
</div>
</body>
</html>
`)
		})

		// Serve each instance's public directory under /<instance>/
		for _, inst := range instances {
			pathPrefix := "/" + inst.Name + "/"
			fs := http.StripPrefix(pathPrefix, http.FileServer(http.Dir(inst.PublicDir)))
			mux.Handle(pathPrefix, fs)
		}
	}

	return &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
}