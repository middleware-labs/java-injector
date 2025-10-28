package main

import (
	"os"

	"github.com/middleware-labs/java-injector/pkg/cli"
)

const (
	DefaultAgentDir  = "/opt/middleware/agents"
	DefaultAgentName = "middleware-javaagent-1.8.1.jar"
	DefaultAgentPath = DefaultAgentDir + "/" + DefaultAgentName
)

func main() {
	router := cli.NewRouter()
	if err := router.Run(os.Args); err != nil {
		// Error is already printed by the router or command
		os.Exit(1)
	}
}
