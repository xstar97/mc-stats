package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
	"strings"
)

type Instance struct {
	Name      string
	Dir       string
	PublicDir string
	Interval  time.Duration

	mu        sync.Mutex
	LastGen   time.Time
	LastDur   time.Duration
	Success   bool
	LastError string
	CurrentPID int
	NextRun   time.Time
}

// Initialize instances with directories, web assets, and start run loops
func InitializeInstances(cfg Config) ([]*Instance, context.CancelFunc, error) {
	var instances []*Instance

	if err := os.MkdirAll(cfg.BaseDir, 0755); err != nil {
		return nil, nil, err
	}

	uid := os.Getuid()
	gid := os.Getgid()

	for _, name := range cfg.Instances {
		name = strings.TrimSpace(name)
		instDir := filepath.Join(cfg.BaseDir, name)
		publicDir := filepath.Join(instDir, "public")

		if err := os.MkdirAll(instDir, 0755); err != nil {
			log.Printf("Failed to create %s: %v", instDir, err)
			continue
		}

		chownRecursive(instDir, uid, gid)

		if _, err := os.Stat(publicDir); os.IsNotExist(err) {
			log.Printf("Initializing web assets for %s", name)
			if err := copyDir("/opt/web", publicDir); err != nil {
				log.Printf("Failed copying web assets for %s: %v", name, err)
			}
			chownRecursive(publicDir, uid, gid)
		}

		inst := &Instance{
			Name:      name,
			Dir:       instDir,
			PublicDir: publicDir,
			Interval:  getInstanceInterval(name),
			NextRun:   time.Now(),
		}

		instances = append(instances, inst)
	}

	// Context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	for _, inst := range instances {
		go inst.runLoop(ctx)
		log.Printf("Started instance %s (interval %v)", inst.Name, inst.Interval)
	}

	return instances, cancel, nil
}

// Run generation on a ticker
func (i *Instance) runLoop(ctx context.Context) {
	ticker := time.NewTicker(i.Interval)
	defer ticker.Stop()

	// Immediate first run
	i.runGeneration()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] Stopping instance due to cancellation", i.Name)
			return
		case <-ticker.C:
			i.runGeneration()
		}
	}
}

// Execute MinecraftStats CLI for instance
func (i *Instance) runGeneration() {
	start := time.Now()
	cmd := exec.Command(
		"java",
		"-jar",
		filepath.Join("/opt/mcstats", "MinecraftStatsCLI.jar"),
		filepath.Join(i.Dir, "config.json"),
	)
	cmd.Dir = i.Dir

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	i.mu.Lock()
	defer i.mu.Unlock()

	i.LastGen = time.Now()
	i.LastDur = duration
	i.Success = err == nil
	if err != nil {
		i.LastError = string(output)
		log.Printf("[%s] Generation failed: %v\n%s", i.Name, err, i.LastError)
	} else {
		i.LastError = ""
		log.Printf("[%s] Generation succeeded in %v", i.Name, duration)
	}
}