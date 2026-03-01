package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Instance represents a single MinecraftStats instance
type Instance struct {
	Name      string
	Dir       string
	PublicDir string
	Interval  time.Duration

	mu         sync.Mutex
	LastGen    time.Time
	LastDur    time.Duration
	Success    bool
	LastError  string
	CurrentPID int
	NextRun    time.Time
}

// InitializeInstances sets up all instances, copies web assets if necessary, and starts the run loops
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
		eventsDir := filepath.Join(instDir, "events")
		statsDir := filepath.Join(instDir, "stats")
		initializedFile := filepath.Join(instDir, ".initialized")

		// Ensure directories exist
		for _, dir := range []string{instDir, publicDir, eventsDir, statsDir} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				log.Printf("[%s] Failed to create directory %s: %v", name, dir, err)
				continue
			}
			chownRecursive(dir, uid, gid)
		}

		// Copy web assets if not yet initialized
		if _, err := os.Stat(initializedFile); os.IsNotExist(err) {
			log.Printf("[%s] Initializing web assets", name)
			if err := copyDir("/opt/web", publicDir); err != nil {
				log.Printf("[%s] Failed to copy web assets: %v", name, err)
			} else {
				// Create the marker file
				if f, err := os.Create(initializedFile); err == nil {
					f.Close()
				} else {
					log.Printf("[%s] Failed to create .initialized file: %v", name, err)
				}
			}
			chownRecursive(publicDir, uid, gid)
		} else {
			log.Printf("[%s] Web assets already initialized, skipping", name)
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

	// Context for cancellation of all instance loops
	ctx, cancel := context.WithCancel(context.Background())

	for _, inst := range instances {
		go inst.runLoop(ctx)
		log.Printf("Started instance %s (interval %v)", inst.Name, inst.Interval)
	}

	return instances, cancel, nil
}

// runLoop executes the generation on a ticker
func (i *Instance) runLoop(ctx context.Context) {
	ticker := time.NewTicker(i.Interval)
	defer ticker.Stop()

	// Run immediately once
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

// runGeneration executes MinecraftStatsCLI for this instance
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
	i.CurrentPID = 0
	if err != nil {
		i.LastError = string(output)
		log.Printf("[%s] Generation failed: %v\n%s", i.Name, err, i.LastError)
	} else {
		i.LastError = ""
		log.Printf("[%s] Generation succeeded in %v", i.Name, duration)
	}
}

// chownRecursive changes ownership of dir recursively
func chownRecursive(path string, uid, gid int) {
	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err == nil {
			os.Chown(p, uid, gid)
		}
		return nil
	})
}
