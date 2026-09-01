package main

import (
	"os"

	sentrymonitoring "github.com/dugble/dugble/server/internal/integrations/monitoring/sentry"

	backofficeregistry "github.com/dugble/dugble/server/internal/runtime/backoffice"
)

func main() {
	if err := backofficeregistry.Start(); err != nil {
		sentrymonitoring.Error("backoffice stopped", "error", err)
		os.Exit(1)
	}
}
