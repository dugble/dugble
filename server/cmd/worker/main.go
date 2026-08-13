package main

import (
	"os"

	sentrymonitoring "github.com/coffeyvidzro/dugble/server/internal/adapters/monitoring/sentry"

	dugbleworker "github.com/coffeyvidzro/dugble/server/internal/dugble/worker"
)

func main() {
	if err := dugbleworker.Start(); err != nil {
		sentrymonitoring.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
