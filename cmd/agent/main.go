package main

import (
	"log"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/agent"
)

func main() {
	myAgent := agent.NewAgent()
	log.Println("Agent is starting...")
	myAgent.Run()
}
