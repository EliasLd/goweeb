package main

import (
	"os"

	"github.com/EliasLd/goweeb/internal/app"
	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/update"
	"github.com/EliasLd/goweeb/internal/version"
)

func main() {
	opts := app.ParseFlags()

	logLevel := logger.LevelInfo
	if opts.Debug {
		logLevel = logger.LevelDebug
	}

	log := logger.New(os.Stdout, logLevel)

	if !opts.NoUpdateCheck {
		if err := update.Cleanup(log); err != nil {
			log.Debug("Update cleanup skipped: %v\n", err)
		}

		// A declined or failed update only continues the flow:
		// downloads were asked for and will not wait behind it.
		_ = update.Offer(version.String(), update.FlavorCLI, log)
	}

	app.Run(opts, log)
}
