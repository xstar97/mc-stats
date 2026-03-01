package main

import (
	"log"
	"os"
	"strings"
	"time"
)

type Config struct {
	Port      string
	BaseDir   string
	Instances []string
}

// Get interval per instance from env
func getInstanceInterval(name string) time.Duration {
	envVar := "INSTANCE_INTERVAL_SECONDS_" + strings.ToUpper(name)
	secStr := getEnv(envVar, getEnv("INTERVAL_SECONDS", "300"))
	sec, err := time.ParseDuration(secStr + "s")
	if err != nil {
		log.Printf("Invalid interval for %s: %v, defaulting to 300s", name, err)
		return 300 * time.Second
	}
	return sec
}

// Load configuration from environment
func LoadConfig() Config {
	instStr := getEnv("INSTANCES", "")
	if instStr == "" {
		log.Fatal("No INSTANCES defined in env")
	}

	return Config{
		Port:      getEnv("PORT", "8080"),
		BaseDir:   "/config",
		Instances: strings.Split(instStr, ","),
	}
}

// Helper
func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}