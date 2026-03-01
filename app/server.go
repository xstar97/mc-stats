package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type InstanceStatus struct {
	Name             string        `json:"name"`
	LastGenerated    time.Time     `json:"last_generated"`
	LastDuration     time.Duration `json:"last_duration"`
	Success          bool          `json:"success"`
	CurrentPID       int           `json:"current_pid,omitempty"`
	TimeUntilNextRun time.Duration `json:"time_until_next_run"`
	LastErrorSnippet string        `json:"last_error_snippet,omitempty"`
}

func NewServer(instances []*Instance) *http.Server {
	mux := http.NewServeMux()

	// /status
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
		// single instance → serve full public
		fs := http.FileServer(http.Dir(instances[0].PublicDir))
		mux.Handle("/", fs)
	} else {
		// multi-instance landing page
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, "<html><body><h1>Select MinecraftStats Instance</h1><ul>")
			for _, inst := range instances {
				fmt.Fprintf(w, `<li><a href="/%s/">%s</a></li>`, inst.Name, inst.Name)
			}
			fmt.Fprintf(w, "</ul></body></html>")
		})

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