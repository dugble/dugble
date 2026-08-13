package main

import (
	"os"

	sentrymonitoring "github.com/coffeyvidzro/dugble/server/internal/adapters/monitoring/sentry"

	dugbleserver "github.com/coffeyvidzro/dugble/server/internal/dugble/server"
)

func main() {
	if err := dugbleserver.Start(); err != nil {
		sentrymonitoring.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
