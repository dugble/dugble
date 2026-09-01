package main

import (
	"os"

	sentrymonitoring "github.com/dugble/dugble/server/internal/integrations/monitoring/sentry"

	serverregistry "github.com/dugble/dugble/server/internal/runtime/server"
)

func main() {
	if err := serverregistry.Start(); err != nil {
		sentrymonitoring.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
