package main

import (
	"github.com/Hobrus/hobrusmetrics.git/internal/app/agent"
	"log"
)

func main() {
	myAgent := agent.NewAgent()
	log.Println("Agent is starting...")
	myAgent.Run()
}
