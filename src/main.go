package main

import (
	// Should this be called tools? There isn't really much going on here...
	"github.com/sktan/aws-codeartifact-proxy/tools"
)

func main() {
	// Do an initial authentication so that we can initialise the proxy properly
	tools.Authenticate()

	// Run a goroutine to check for reauthentication to the CodeArtifact Service
	go tools.CheckReauth()

	// Start the Proxy listener so that we can intercept the requests
	tools.ProxyInit()
}
