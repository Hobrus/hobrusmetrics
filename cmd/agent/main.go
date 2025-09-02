package main

import (
	"fmt"
	"log"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/agent"
)

// Build information is injected via -ldflags at build time.
var buildVersion string
var buildDate string
var buildCommit string

func printBuildInfo() {
	version := buildVersion
	if version == "" {
		version = "N/A"
	}
	date := buildDate
	if date == "" {
		date = "N/A"
	}
	commit := buildCommit
	if commit == "" {
		commit = "N/A"
	}
	fmt.Printf("Build version: %s\n", version)
	fmt.Printf("Build date: %s\n", date)
	fmt.Printf("Build commit: %s\n", commit)
}

// Точка входа агента сбора метрик.
func main() {
	printBuildInfo()
	myAgent := agent.NewAgent()
	log.Println("Agent is starting...")
	myAgent.Run()
}
