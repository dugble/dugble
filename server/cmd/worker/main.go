package main

import (
	"os"

	sentrymonitoring "github.com/dugble/dugble/server/internal/integrations/monitoring/sentry"

	workerregistry "github.com/dugble/dugble/server/internal/runtime/worker"
)

func main() {
	if err := workerregistry.Start(); err != nil {
		sentrymonitoring.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
