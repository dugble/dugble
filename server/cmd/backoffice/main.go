package main

import (
	"os"

	sentrymonitoring "github.com/dugble/dugble/server/internal/adapters/monitoring/sentry"

	backofficeregistry "github.com/dugble/dugble/server/internal/registry/backoffice"
)

func main() {
	if err := backofficeregistry.Start(); err != nil {
		sentrymonitoring.Error("backoffice stopped", "error", err)
		os.Exit(1)
	}
}
