package main

import (
	"os"

	sentrymonitoring "github.com/coffeyvidzro/dugble/server/internal/adapters/monitoring/sentry"

	dugblebackoffice "github.com/coffeyvidzro/dugble/server/internal/dugble/backoffice"
)

func main() {
	if err := dugblebackoffice.Start(); err != nil {
		sentrymonitoring.Error("backoffice stopped", "error", err)
		os.Exit(1)
	}
}
