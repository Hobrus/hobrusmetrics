package main

import (
	"log"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/agent"
	"github.com/Hobrus/hobrusmetrics.git/internal/pkg/buildinfo"
)

// Build information is injected via -ldflags at build time.
var buildVersion string
var buildDate string
var buildCommit string

func printBuildInfo() { buildinfo.Print(buildVersion, buildDate, buildCommit) }

// Точка входа агента сбора метрик.
func main() {
	printBuildInfo()
	myAgent := agent.NewAgent()
	log.Println("Agent is starting...")
	myAgent.Run()
}
